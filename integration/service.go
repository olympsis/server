package integration

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"olympsis-server/database"
	"olympsis-server/server"
	"olympsis-server/utils"
	"time"

	"github.com/gorilla/mux"
	"github.com/olympsis/models"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// linkCodeTTL bounds how long a pending link code is valid before an admin must
// regenerate it.
const linkCodeTTL = 30 * time.Minute

// Service handles club <-> chat (Telegram/Discord) integration management.
type Service struct {
	Database *database.Database
	Logger   *logrus.Logger
	Router   *mux.Router
	Config   *utils.ServerConfig
}

func NewIntegrationService(i *server.ServerInterface, cfg *utils.ServerConfig) *Service {
	return &Service{
		Database: i.Database,
		Logger:   i.Logger,
		Router:   i.Router,
		Config:   cfg,
	}
}

// StartLink begins linking a club to a Telegram group or Discord channel. It validates
// admin permissions, issues a one-time code, and returns setup instructions. The admin
// then runs /link <code> inside the target chat to complete the binding.
//
// POST /v1/clubs/{id}/integrations/{platform}
func (s *Service) StartLink() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		oid, err := utils.ValidateObjectID(mux.Vars(r)["id"])
		if err != nil {
			http.Error(rw, `{"msg": "invalid club id"}`, http.StatusBadRequest)
			return
		}

		platform, ok := parsePlatform(mux.Vars(r)["platform"])
		if !ok {
			http.Error(rw, `{"msg": "unsupported platform"}`, http.StatusBadRequest)
			return
		}

		// Only club owners/admins may link a chat.
		uuid := r.Header.Get("userID")
		if !s.isClubAdmin(ctx, uuid, oid) {
			http.Error(rw, `{"msg": "invalid permission"}`, http.StatusUnauthorized)
			return
		}

		code, err := generateCode()
		if err != nil {
			s.Logger.Errorf("Failed to generate link code. Error: %s", err.Error())
			http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			return
		}

		timestamp := bson.NewDateTimeFromTime(time.Now())
		pending := models.ChatLinkPending
		filter := bson.M{"club_id": oid, "platform": platform}
		update := bson.M{
			"$set": bson.M{
				"club_id":    oid,
				"platform":   platform,
				"status":     pending,
				"link_code":  code,
				"linked_by":  uuid,
				"created_at": timestamp,
			},
			"$unset": bson.M{"chat_id": "", "guild_id": "", "channel_id": "", "title": ""},
		}
		if _, err := s.Database.ClubChatLinksCollection.UpdateOne(ctx, filter, update, options.UpdateOne().SetUpsert(true)); err != nil {
			s.Logger.Errorf("Failed to upsert chat link. Error: %s", err.Error())
			http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			return
		}

		resp := models.IntegrationLinkResponse{
			Platform:     platform,
			LinkCode:     code,
			Instructions: s.instructions(platform, code),
			ExpiresAt:    bson.NewDateTimeFromTime(time.Now().Add(linkCodeTTL)),
		}
		if platform == models.PlatformDiscord {
			resp.InviteURL = discordInviteURL(s.Config.DiscordAppID)
		}

		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(resp)
	}
}

// GetLinks lists the chat links for a club. Any club member may view them.
//
// GET /v1/clubs/{id}/integrations
func (s *Service) GetLinks() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		oid, err := utils.ValidateObjectID(mux.Vars(r)["id"])
		if err != nil {
			http.Error(rw, `{"msg": "invalid club id"}`, http.StatusBadRequest)
			return
		}

		cursor, err := s.Database.ClubChatLinksCollection.Find(ctx, bson.M{"club_id": oid})
		if err != nil {
			s.Logger.Errorf("Failed to find chat links. Error: %s", err.Error())
			http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			return
		}
		var links []models.ClubChatLink
		if err := cursor.All(ctx, &links); err != nil {
			s.Logger.Errorf("Failed to decode chat links. Error: %s", err.Error())
			http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			return
		}

		// Never leak active link codes to clients.
		for i := range links {
			links[i].LinkCode = ""
		}

		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(links)
	}
}

// DeleteLink unlinks a club from a platform's chat. Requires admin permissions.
//
// DELETE /v1/clubs/{id}/integrations/{platform}
func (s *Service) DeleteLink() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		oid, err := utils.ValidateObjectID(mux.Vars(r)["id"])
		if err != nil {
			http.Error(rw, `{"msg": "invalid club id"}`, http.StatusBadRequest)
			return
		}
		platform, ok := parsePlatform(mux.Vars(r)["platform"])
		if !ok {
			http.Error(rw, `{"msg": "unsupported platform"}`, http.StatusBadRequest)
			return
		}

		uuid := r.Header.Get("userID")
		if !s.isClubAdmin(ctx, uuid, oid) {
			http.Error(rw, `{"msg": "invalid permission"}`, http.StatusUnauthorized)
			return
		}

		if _, err := s.Database.ClubChatLinksCollection.DeleteOne(ctx, bson.M{"club_id": oid, "platform": platform}); err != nil {
			s.Logger.Errorf("Failed to delete chat link. Error: %s", err.Error())
			http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{"msg": "OK"}`))
	}
}

// ConfirmLink is called by the bots service when an admin runs /link <code> inside the
// target chat. It finalizes the binding and returns the activated link. Internal-only.
//
// POST /v1/integrations/confirm
func (s *Service) ConfirmLink() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		var req models.IntegrationConfirmRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(rw, `{"msg": "bad request"}`, http.StatusBadRequest)
			return
		}
		if req.LinkCode == "" {
			http.Error(rw, `{"msg": "missing link code"}`, http.StatusBadRequest)
			return
		}

		var link models.ClubChatLink
		err := s.Database.ClubChatLinksCollection.FindOne(ctx, bson.M{"link_code": req.LinkCode}).Decode(&link)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				http.Error(rw, `{"msg": "invalid code"}`, http.StatusNotFound)
				return
			}
			s.Logger.Errorf("Failed to find chat link. Error: %s", err.Error())
			http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			return
		}

		// Validate the code is still pending, matches the platform, and hasn't expired.
		if link.Status != models.ChatLinkPending || link.Platform != req.Platform ||
			time.Since(link.CreatedAt.Time()) > linkCodeTTL {
			http.Error(rw, `{"msg": "code expired or already used"}`, http.StatusConflict)
			return
		}

		timestamp := bson.NewDateTimeFromTime(time.Now())
		set := bson.M{
			"status":     models.ChatLinkActive,
			"title":      req.Title,
			"updated_at": timestamp,
		}
		switch req.Platform {
		case models.PlatformTelegram:
			set["chat_id"] = req.ChatID
		case models.PlatformDiscord:
			set["guild_id"] = req.GuildID
			set["channel_id"] = req.ChannelID
		}
		update := bson.M{"$set": set, "$unset": bson.M{"link_code": ""}}
		if _, err := s.Database.ClubChatLinksCollection.UpdateOne(ctx, bson.M{"_id": link.ID}, update); err != nil {
			s.Logger.Errorf("Failed to activate chat link. Error: %s", err.Error())
			http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			return
		}

		// Return the finalized link.
		_ = s.Database.ClubChatLinksCollection.FindOne(ctx, bson.M{"_id": link.ID}).Decode(&link)
		link.LinkCode = ""

		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(link)
	}
}

// isClubAdmin returns true when the user is an OWNER or ADMIN of the club.
func (s *Service) isClubAdmin(ctx context.Context, userID string, clubID bson.ObjectID) bool {
	var member models.MemberDao
	err := s.Database.ClubMembersCollection.FindOne(ctx, bson.M{"user_id": userID, "club_id": clubID}).Decode(&member)
	if err != nil {
		return false
	}
	return member.Role == string(models.OwnerMember) || member.Role == string(models.AdminMember)
}

func (s *Service) instructions(platform models.ChatPlatform, code string) string {
	switch platform {
	case models.PlatformTelegram:
		return fmt.Sprintf("Add the Olympsis bot to your Telegram group, then send: /link %s", code)
	case models.PlatformDiscord:
		return fmt.Sprintf("Invite the Olympsis bot to your server using the invite URL, then send in the target channel: !link %s", code)
	default:
		return ""
	}
}

func parsePlatform(p string) (models.ChatPlatform, bool) {
	switch models.ChatPlatform(p) {
	case models.PlatformTelegram:
		return models.PlatformTelegram, true
	case models.PlatformDiscord:
		return models.PlatformDiscord, true
	default:
		return "", false
	}
}

func generateCode() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func discordInviteURL(appID string) string {
	if appID == "" {
		return ""
	}
	return fmt.Sprintf("https://discord.com/oauth2/authorize?client_id=%s&scope=bot&permissions=3072", appID)
}
