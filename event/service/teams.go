package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// CreateTeam adds a team to an event and invites the requested users to it.
//
// There is no GET counterpart on purpose: teams come back on the event itself
// via the aggregation's `teams` lookup (GET /v1/events/{id}).
//
// Invitees are NOT written onto the team. They are published on `team.created`,
// invite-service fans them into individual invite records, and a user only
// becomes a TeamMember once they accept.
func (s *Service) CreateTeam() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID := r.Header.Get("userID")

		// Grab event id from path
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(w, `{"msg": "bad event id"}`, http.StatusBadRequest)
			return
		}
		oid, err := bson.ObjectIDFromHex(id)
		if err != nil {
			s.Logger.Error("Failed to convert id to ObjectID. Error: ", err.Error())
			http.Error(w, `{"msg": "failed to encode id"}`, http.StatusBadRequest)
			return
		}

		// Decode request
		var req models.NewTeamDao
		if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.Logger.Error("Failed to decode request. Error: ", err.Error())
			http.Error(w, `{"msg":"failed to decode request"}`, http.StatusBadRequest)
			return
		}
		if req.Team.Name == nil || *req.Team.Name == "" {
			http.Error(w, `{"msg": "team name is required"}`, http.StatusBadRequest)
			return
		}

		// Confirm the event exists before writing a team that would dangle.
		event, err := s.FindEvent(ctx, bson.M{"_id": oid})
		if err != nil {
			s.Logger.Error("Failed to find event. Error: ", err.Error())
			http.Error(w, `{"msg": "event not found"}`, http.StatusNotFound)
			return
		}
		if event.CancelledAt != nil || event.ArchivedAt != nil {
			http.Error(w, `{"msg": "event is no longer accepting teams"}`, http.StatusConflict)
			return
		}

		// Teams are only valid on team-RSVP events. On an individual-RSVP event
		// there is nothing to attach a team to, so refuse rather than create a
		// dangling roster.
		if !teamRSVPRequired(event) {
			http.Error(w, `{"msg": "this event does not accept teams"}`, http.StatusForbidden)
			return
		}

		// A user can only be on one team per event. Creating a second would both
		// double-count their RSVP and trip the unique {event_id, members.user_id}
		// index on insert.
		if existing, _ := s.FindTeams(ctx, bson.M{"event_id": oid, "members.user_id": userID}, nil); len(existing) > 0 {
			http.Error(w, `{"msg": "you are already on a team for this event"}`, http.StatusConflict)
			return
		}

		// Enforce the event's team cap, mirroring how AddParticipant honours
		// MaxParticipants.
		teams, err := s.FindTeams(ctx, bson.M{"event_id": oid}, nil)
		if err != nil {
			s.Logger.Error("Failed to find event's teams. Error: ", err.Error())
			http.Error(w, `{"msg": "failed to find event's teams"}`, http.StatusInternalServerError)
			return
		}
		if event.TeamsConfig != nil && event.TeamsConfig.MaxTeams != nil && *event.TeamsConfig.MaxTeams != 0 {
			if len(teams) >= int(*event.TeamsConfig.MaxTeams) {
				http.Error(w, `{"msg": "event is at its team limit"}`, http.StatusConflict)
				return
			}
		}

		// The creator is the team's first member and its OWNER; everyone else has
		// to accept an invite, self-join (open team), or be approved (closed team).
		timestamp := bson.NewDateTimeFromTime(time.Now())
		isAnonymous := false
		ownerRole := models.OwnerMember
		isOpen := false
		if req.Team.IsOpen != nil {
			isOpen = *req.Team.IsOpen
		}
		team := models.TeamDao{
			Name:    req.Team.Name,
			IsOpen:  &isOpen,
			EventID: &oid,
			Members: &[]models.TeamMemberDao{{
				UserID:      &userID,
				Role:        &ownerRole,
				IsAnonymous: &isAnonymous,
				JoinedAt:    &timestamp,
			}},
			CreatedAt: &timestamp,
		}

		tid, err := s.InsertTeam(ctx, &team)
		if err != nil {
			s.Logger.Error("Failed to insert team into the database. Error: ", err.Error())
			http.Error(w, `{"msg": "failed to create team"}`, http.StatusInternalServerError)
			return
		}

		// Fan the invitees out to invite records via invite-service.
		s.publishInviteRequest(ctx, models.InviteTypeTeam, tid.Hex(), userID, req.Invitees)

		w.WriteHeader(http.StatusCreated)
		w.Write(fmt.Appendf(nil, `{"id": "%s"}`, tid.Hex()))
	}
}

// RemoveTeam deletes a team from an event.
func (s *Service) RemoveTeam() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		vars := mux.Vars(r)
		eventID := vars["id"]
		teamID := vars["teamID"]
		if len(eventID) < 24 || len(teamID) < 24 {
			http.Error(w, `{"msg": "bad id"}`, http.StatusBadRequest)
			return
		}

		eventOID, err := bson.ObjectIDFromHex(eventID)
		if err != nil {
			http.Error(w, `{"msg": "failed to encode event id"}`, http.StatusBadRequest)
			return
		}
		teamOID, err := bson.ObjectIDFromHex(teamID)
		if err != nil {
			http.Error(w, `{"msg": "failed to encode team id"}`, http.StatusBadRequest)
			return
		}

		// Authorize before deleting. UserMiddleware only proves WHO the caller is;
		// without this check any authenticated user could enumerate team ids from
		// the public event payload and delete other people's teams.
		userID := r.Header.Get("userID")
		team, err := s.FindTeam(ctx, bson.M{"_id": teamOID, "event_id": eventOID})
		if err != nil {
			http.Error(w, `{"msg": "team not found"}`, http.StatusNotFound)
			return
		}
		event, err := s.FindEvent(ctx, bson.M{"_id": eventOID})
		if err != nil {
			s.Logger.Error("Failed to find event. Error: ", err.Error())
			http.Error(w, `{"msg": "event not found"}`, http.StatusNotFound)
			return
		}
		if !canManageTeam(event, team, userID) {
			http.Error(w, `{"msg": "not authorized to remove this team"}`, http.StatusForbidden)
			return
		}

		// Capture the roster before deleting so we can tell members their RSVP was
		// cancelled. Skip the actor (owner/host) — they initiated this.
		var recipients []string
		for _, uid := range teamMemberIDs(team) {
			if uid != userID {
				recipients = append(recipients, uid)
			}
		}

		// Scope the delete by event id too, so a team id from another event can't
		// be removed through this route.
		if err = s.DeleteTeam(ctx, bson.M{"_id": teamOID, "event_id": eventOID}); err != nil {
			s.Logger.Errorf("Failed to delete team. Error: %s", err.Error())
			http.Error(w, `{"msg": "failed to remove team"}`, http.StatusInternalServerError)
			return
		}

		// Cascade: purge any pending/closed applications for this team so they
		// don't dangle. (Stale team invites are harmless — they no-op on accept
		// because the team no longer exists.)
		if err = s.DeleteTeamApplications(ctx, bson.M{"team_id": teamOID}); err != nil {
			s.Logger.Errorf("Failed to delete team applications. Team: %s - Error: %s", teamID, err.Error())
		}

		// Disbanding a team cancels every member's RSVP; let them know.
		if len(recipients) > 0 {
			if err = s.Notification.TeamDeleted(event, team, recipients); err != nil {
				s.Logger.Errorf("Failed to notify members of team deletion. Team: %s - Error: %s", teamID, err.Error())
			}
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"msg": "OK"}`))
	}
}

// Sentinel errors for the pure team-membership helpers. Handlers map these to
// HTTP statuses; the invite-accept consumer treats them as non-fatal.
var (
	errTeamFull         = errors.New("team is at capacity")
	errAlreadyMember    = errors.New("user is already a member")
	errNotAMember       = errors.New("user is not a member")
	errNewOwnerNotFound = errors.New("new owner is not a member of the team")
)

// teamRSVPRequired reports whether the event is in team-RSVP mode, i.e. users
// must RSVP by creating or joining a team rather than as individuals.
func teamRSVPRequired(event *models.EventDao) bool {
	return event != nil &&
		event.TeamsConfig != nil &&
		event.TeamsConfig.Required != nil &&
		*event.TeamsConfig.Required
}

// teamMembers returns the team's members as a plain slice (nil-safe).
func teamMembers(team *models.TeamDao) []models.TeamMemberDao {
	if team == nil || team.Members == nil {
		return nil
	}
	return *team.Members
}

// teamOwnerID returns the user id of the team's OWNER. For teams created before
// roles existed (no member carries OWNER), it falls back to the first member —
// which is who CreateTeam has always seeded as the creator.
func teamOwnerID(team *models.TeamDao) (string, bool) {
	members := teamMembers(team)
	for i := range members {
		if members[i].Role != nil && *members[i].Role == models.OwnerMember && members[i].UserID != nil {
			return *members[i].UserID, true
		}
	}
	if len(members) > 0 && members[0].UserID != nil {
		return *members[0].UserID, true
	}
	return "", false
}

// isTeamOwner reports whether userID owns the team.
func isTeamOwner(team *models.TeamDao, userID string) bool {
	owner, ok := teamOwnerID(team)
	return ok && userID != "" && owner == userID
}

// isTeamMember reports whether userID is on the team.
func isTeamMember(team *models.TeamDao, userID string) bool {
	for _, m := range teamMembers(team) {
		if m.UserID != nil && *m.UserID == userID {
			return true
		}
	}
	return false
}

// teamMemberIDs returns every member's user id (skipping any nil ids).
func teamMemberIDs(team *models.TeamDao) []string {
	members := teamMembers(team)
	ids := make([]string, 0, len(members))
	for i := range members {
		if members[i].UserID != nil {
			ids = append(ids, *members[i].UserID)
		}
	}
	return ids
}

// canManageTeam reports whether userID may manage/remove the given team: either
// they own it (OWNER role, falling back to creator for legacy teams) or they
// host the event.
func canManageTeam(event *models.EventDao, team *models.TeamDao, userID string) bool {
	if userID == "" {
		return false
	}
	if event != nil && event.PosterID != nil && *event.PosterID == userID {
		return true
	}
	return isTeamOwner(team, userID)
}

// canLeaveTeam reports whether userID may leave the team on their own. Any member
// may leave EXCEPT the owner, who must first transfer ownership or delete the
// team (so a team is never left ownerless).
func canLeaveTeam(team *models.TeamDao, userID string) bool {
	return isTeamMember(team, userID) && !isTeamOwner(team, userID)
}

// canAddTeamMember reports whether userID could be added to members: it errors
// with errAlreadyMember if they're already on the roster, or errTeamFull if the
// cap is reached. maxTeamSize == nil or <= 0 means unlimited.
func canAddTeamMember(members []models.TeamMemberDao, userID string, maxTeamSize *int32) error {
	for _, m := range members {
		if m.UserID != nil && *m.UserID == userID {
			return errAlreadyMember
		}
	}
	if maxTeamSize != nil && *maxTeamSize > 0 && len(members) >= int(*maxTeamSize) {
		return errTeamFull
	}
	return nil
}

// addTeamMember is the pure core of the three join paths (self-join, application
// approval, invite acceptance). It appends userID as a MEMBER, enforcing
// capacity and rejecting duplicates via canAddTeamMember. The DB-level atomic
// add mirrors this exactly so concurrent joins can't race past the cap.
func addTeamMember(members []models.TeamMemberDao, userID string, maxTeamSize *int32, joinedAt bson.DateTime) ([]models.TeamMemberDao, error) {
	if err := canAddTeamMember(members, userID, maxTeamSize); err != nil {
		return members, err
	}
	role := models.MemberMember
	isAnonymous := false
	return append(members, models.TeamMemberDao{
		UserID:      &userID,
		Role:        &role,
		IsAnonymous: &isAnonymous,
		JoinedAt:    &joinedAt,
	}), nil
}

// applyOwnershipTransfer moves the OWNER role from the current owner to
// newOwnerID. Both must be members; afterwards exactly one OWNER exists. Pure so
// the invariant can be unit-tested; the handler performs the same change via an
// atomic Mongo update with arrayFilters.
func applyOwnershipTransfer(members []models.TeamMemberDao, currentOwnerID, newOwnerID string) ([]models.TeamMemberDao, error) {
	var foundNew bool
	for i := range members {
		if members[i].UserID != nil && *members[i].UserID == newOwnerID {
			foundNew = true
		}
	}
	if !foundNew {
		return members, errNewOwnerNotFound
	}

	owner := models.OwnerMember
	member := models.MemberMember
	for i := range members {
		if members[i].UserID == nil {
			continue
		}
		switch *members[i].UserID {
		case newOwnerID:
			members[i].Role = &owner
		case currentOwnerID:
			members[i].Role = &member
		}
	}
	return members, nil
}

// JoinTeam adds the caller to an OPEN team (self-join on RSVP). Closed teams
// reject this and require an application instead.
func (s *Service) JoinTeam() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID := r.Header.Get("userID")
		vars := mux.Vars(r)
		eventID, teamID := vars["id"], vars["teamID"]
		if len(eventID) < 24 || len(teamID) < 24 {
			http.Error(w, `{"msg": "bad id"}`, http.StatusBadRequest)
			return
		}
		eventOID, err := bson.ObjectIDFromHex(eventID)
		if err != nil {
			http.Error(w, `{"msg": "failed to encode event id"}`, http.StatusBadRequest)
			return
		}
		teamOID, err := bson.ObjectIDFromHex(teamID)
		if err != nil {
			http.Error(w, `{"msg": "failed to encode team id"}`, http.StatusBadRequest)
			return
		}

		event, err := s.FindEvent(ctx, bson.M{"_id": eventOID})
		if err != nil {
			http.Error(w, `{"msg": "event not found"}`, http.StatusNotFound)
			return
		}
		if event.CancelledAt != nil || event.ArchivedAt != nil {
			http.Error(w, `{"msg": "event is no longer accepting RSVPs"}`, http.StatusConflict)
			return
		}

		team, err := s.FindTeam(ctx, bson.M{"_id": teamOID, "event_id": eventOID})
		if err != nil {
			http.Error(w, `{"msg": "team not found"}`, http.StatusNotFound)
			return
		}
		if team.IsOpen == nil || !*team.IsOpen {
			http.Error(w, `{"msg": "this team is closed; apply to join instead"}`, http.StatusForbidden)
			return
		}

		// One team per user per event.
		if existing, _ := s.FindTeams(ctx, bson.M{"event_id": eventOID, "members.user_id": userID}, nil); len(existing) > 0 {
			http.Error(w, `{"msg": "you are already on a team for this event"}`, http.StatusConflict)
			return
		}

		added, err := s.AddTeamMemberAtomic(ctx, teamOID, userID, maxTeamSizeFor(event))
		if err != nil {
			s.Logger.Errorf("Failed to join team. Team: %s - Error: %s", teamID, err.Error())
			http.Error(w, `{"msg": "failed to join team"}`, http.StatusInternalServerError)
			return
		}
		if !added {
			http.Error(w, `{"msg": "team is full or you are already a member"}`, http.StatusConflict)
			return
		}

		// Reminders: add to the event topic, mirroring AddParticipant.
		if err = s.Notification.AddUsersToTopic(eventID, []string{userID}); err != nil {
			s.Logger.Errorf("Failed to add user to event topic. Event: %s - Error: %s", eventID, err.Error())
		}

		w.WriteHeader(http.StatusCreated)
		w.Write(fmt.Appendf(nil, `{"id": "%s"}`, teamID))
	}
}

// InviteToTeam lets the owner invite more users to their team after creation.
// Invitees are fanned out through invite-service exactly like CreateTeam; they
// become members only once they accept (which lands on AddTeamMemberViaInvite).
func (s *Service) InviteToTeam() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID := r.Header.Get("userID")
		vars := mux.Vars(r)
		eventID, teamID := vars["id"], vars["teamID"]
		if len(eventID) < 24 || len(teamID) < 24 {
			http.Error(w, `{"msg": "bad id"}`, http.StatusBadRequest)
			return
		}
		eventOID, err := bson.ObjectIDFromHex(eventID)
		if err != nil {
			http.Error(w, `{"msg": "failed to encode event id"}`, http.StatusBadRequest)
			return
		}
		teamOID, err := bson.ObjectIDFromHex(teamID)
		if err != nil {
			http.Error(w, `{"msg": "failed to encode team id"}`, http.StatusBadRequest)
			return
		}

		team, err := s.FindTeam(ctx, bson.M{"_id": teamOID, "event_id": eventOID})
		if err != nil {
			http.Error(w, `{"msg": "team not found"}`, http.StatusNotFound)
			return
		}
		if !isTeamOwner(team, userID) {
			http.Error(w, `{"msg": "only the team owner can invite"}`, http.StatusForbidden)
			return
		}

		var req struct {
			Invitees []string `json:"invitees"`
		}
		if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"msg": "failed to decode request"}`, http.StatusBadRequest)
			return
		}

		s.publishInviteRequest(ctx, models.InviteTypeTeam, teamID, userID, req.Invitees)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"msg": "OK"}`))
	}
}

// KickTeamMember removes a member from the team (owner only). Removal cancels
// that member's RSVP and frees a seat.
func (s *Service) KickTeamMember() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		callerID := r.Header.Get("userID")
		vars := mux.Vars(r)
		eventID, teamID, targetID := vars["id"], vars["teamID"], vars["memberUserID"]
		if len(eventID) < 24 || len(teamID) < 24 || targetID == "" {
			http.Error(w, `{"msg": "bad id"}`, http.StatusBadRequest)
			return
		}
		eventOID, err := bson.ObjectIDFromHex(eventID)
		if err != nil {
			http.Error(w, `{"msg": "failed to encode event id"}`, http.StatusBadRequest)
			return
		}
		teamOID, err := bson.ObjectIDFromHex(teamID)
		if err != nil {
			http.Error(w, `{"msg": "failed to encode team id"}`, http.StatusBadRequest)
			return
		}

		event, err := s.FindEvent(ctx, bson.M{"_id": eventOID})
		if err != nil {
			http.Error(w, `{"msg": "event not found"}`, http.StatusNotFound)
			return
		}
		team, err := s.FindTeam(ctx, bson.M{"_id": teamOID, "event_id": eventOID})
		if err != nil {
			http.Error(w, `{"msg": "team not found"}`, http.StatusNotFound)
			return
		}

		if !isTeamOwner(team, callerID) {
			http.Error(w, `{"msg": "only the team owner can remove members"}`, http.StatusForbidden)
			return
		}
		if isTeamOwner(team, targetID) {
			http.Error(w, `{"msg": "the owner cannot be removed; transfer ownership or delete the team"}`, http.StatusConflict)
			return
		}
		if !isTeamMember(team, targetID) {
			http.Error(w, `{"msg": "member not found on this team"}`, http.StatusNotFound)
			return
		}

		if err = s.UpdateTeam(ctx, bson.M{"_id": teamOID}, bson.M{"$pull": bson.M{"members": bson.M{"user_id": targetID}}}); err != nil {
			s.Logger.Errorf("Failed to remove team member. Team: %s - Error: %s", teamID, err.Error())
			http.Error(w, `{"msg": "failed to remove member"}`, http.StatusInternalServerError)
			return
		}

		if err = s.Notification.RemoveUsersFromTopic(eventID, []string{targetID}); err != nil {
			s.Logger.Errorf("Failed to remove user from event topic. Event: %s - Error: %s", eventID, err.Error())
		}
		if err = s.Notification.TeamKick(event, team, targetID); err != nil {
			s.Logger.Errorf("Failed to notify kicked member. Team: %s - Error: %s", teamID, err.Error())
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"msg": "OK"}`))
	}
}

// LeaveTeam removes the caller from the team. Any member may leave EXCEPT the
// owner, who must transfer ownership or delete the team first — signalled with a
// 409 and the OWNER_MUST_TRANSFER code so the client can prompt for the choice.
func (s *Service) LeaveTeam() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID := r.Header.Get("userID")
		vars := mux.Vars(r)
		eventID, teamID := vars["id"], vars["teamID"]
		if len(eventID) < 24 || len(teamID) < 24 {
			http.Error(w, `{"msg": "bad id"}`, http.StatusBadRequest)
			return
		}
		eventOID, err := bson.ObjectIDFromHex(eventID)
		if err != nil {
			http.Error(w, `{"msg": "failed to encode event id"}`, http.StatusBadRequest)
			return
		}
		teamOID, err := bson.ObjectIDFromHex(teamID)
		if err != nil {
			http.Error(w, `{"msg": "failed to encode team id"}`, http.StatusBadRequest)
			return
		}

		team, err := s.FindTeam(ctx, bson.M{"_id": teamOID, "event_id": eventOID})
		if err != nil {
			http.Error(w, `{"msg": "team not found"}`, http.StatusNotFound)
			return
		}
		if !isTeamMember(team, userID) {
			http.Error(w, `{"msg": "you are not on this team"}`, http.StatusNotFound)
			return
		}
		if isTeamOwner(team, userID) {
			http.Error(w, `{"msg": "the owner must transfer ownership or delete the team", "code": "OWNER_MUST_TRANSFER"}`, http.StatusConflict)
			return
		}

		if err = s.UpdateTeam(ctx, bson.M{"_id": teamOID}, bson.M{"$pull": bson.M{"members": bson.M{"user_id": userID}}}); err != nil {
			s.Logger.Errorf("Failed to leave team. Team: %s - Error: %s", teamID, err.Error())
			http.Error(w, `{"msg": "failed to leave team"}`, http.StatusInternalServerError)
			return
		}

		if err = s.Notification.RemoveUsersFromTopic(eventID, []string{userID}); err != nil {
			s.Logger.Errorf("Failed to remove user from event topic. Event: %s - Error: %s", eventID, err.Error())
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"msg": "OK"}`))
	}
}

// TransferTeamOwnership moves the OWNER role from the caller to another member
// (owner only). The new owner must already be on the team. This is how an owner
// can then leave — promote someone, then LeaveTeam.
func (s *Service) TransferTeamOwnership() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID := r.Header.Get("userID")
		vars := mux.Vars(r)
		eventID, teamID := vars["id"], vars["teamID"]
		if len(eventID) < 24 || len(teamID) < 24 {
			http.Error(w, `{"msg": "bad id"}`, http.StatusBadRequest)
			return
		}
		eventOID, err := bson.ObjectIDFromHex(eventID)
		if err != nil {
			http.Error(w, `{"msg": "failed to encode event id"}`, http.StatusBadRequest)
			return
		}
		teamOID, err := bson.ObjectIDFromHex(teamID)
		if err != nil {
			http.Error(w, `{"msg": "failed to encode team id"}`, http.StatusBadRequest)
			return
		}

		var req struct {
			NewOwnerID string `json:"new_owner_id"`
		}
		if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"msg": "failed to decode request"}`, http.StatusBadRequest)
			return
		}
		if req.NewOwnerID == "" {
			http.Error(w, `{"msg": "new_owner_id is required"}`, http.StatusBadRequest)
			return
		}

		event, err := s.FindEvent(ctx, bson.M{"_id": eventOID})
		if err != nil {
			http.Error(w, `{"msg": "event not found"}`, http.StatusNotFound)
			return
		}
		team, err := s.FindTeam(ctx, bson.M{"_id": teamOID, "event_id": eventOID})
		if err != nil {
			http.Error(w, `{"msg": "team not found"}`, http.StatusNotFound)
			return
		}

		if !isTeamOwner(team, userID) {
			http.Error(w, `{"msg": "only the owner can transfer ownership"}`, http.StatusForbidden)
			return
		}
		if req.NewOwnerID == userID {
			http.Error(w, `{"msg": "you are already the owner"}`, http.StatusBadRequest)
			return
		}

		updated, err := applyOwnershipTransfer(teamMembers(team), userID, req.NewOwnerID)
		if err != nil {
			http.Error(w, `{"msg": "new owner must be a member of the team"}`, http.StatusBadRequest)
			return
		}

		if err = s.UpdateTeam(ctx, bson.M{"_id": teamOID}, bson.M{"$set": bson.M{"members": updated}}); err != nil {
			s.Logger.Errorf("Failed to transfer team ownership. Team: %s - Error: %s", teamID, err.Error())
			http.Error(w, `{"msg": "failed to transfer ownership"}`, http.StatusInternalServerError)
			return
		}

		if err = s.Notification.TeamMemberRoleChange(event, team, req.NewOwnerID, models.OwnerMember); err != nil {
			s.Logger.Errorf("Failed to notify new owner. Team: %s - Error: %s", teamID, err.Error())
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"msg": "OK"}`))
	}
}

/*****************
DATABASE FUNCTIONS
******************/

// Insert new team into database
func (s *Service) InsertTeam(ctx context.Context, team *models.TeamDao) (bson.ObjectID, error) {
	resp, err := s.Database.EventTeamsCollection.InsertOne(ctx, team)
	if err != nil {
		return bson.NilObjectID, err
	}
	id := resp.InsertedID.(bson.ObjectID)
	return id, nil
}

// Get team from database
func (s *Service) FindTeam(ctx context.Context, filter bson.M) (*models.TeamDao, error) {
	var team models.TeamDao
	err := s.Database.EventTeamsCollection.FindOne(ctx, filter).Decode(&team)
	if err != nil {
		return nil, err
	}
	return &team, nil
}

// Get teams from database
func (s *Service) FindTeams(ctx context.Context, filter bson.M, opts *options.FindOptionsBuilder) ([]models.TeamDao, error) {
	var teams []models.TeamDao
	cursor, err := s.Database.EventTeamsCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &teams); err != nil {
		return nil, err
	}
	return teams, nil
}

// Update team in database
func (s *Service) UpdateTeam(ctx context.Context, filter bson.M, update bson.M) error {
	_, err := s.Database.EventTeamsCollection.UpdateOne(ctx, filter, update)
	return err
}

// Delete team from database
func (s *Service) DeleteTeam(ctx context.Context, filter bson.M) error {
	_, err := s.Database.EventTeamsCollection.DeleteOne(ctx, filter)
	return err
}

// Delete multiple teams from database
func (s *Service) DeleteTeams(ctx context.Context, filter bson.M) error {
	_, err := s.Database.EventTeamsCollection.DeleteMany(ctx, filter)
	return err
}
