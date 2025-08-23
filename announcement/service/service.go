package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"olympsis-server/aggregations"
	"olympsis-server/database"
	"olympsis-server/notifications"
	"olympsis-server/utils"
	"time"

	"github.com/gorilla/mux"
	"github.com/olympsis/models"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Service struct {
	Database     *database.Database
	Logger       *logrus.Logger
	Router       *mux.Router
	Notification *notifications.Service
}

/*
Create new announcement service struct
*/
func NewAnnouncementService(l *logrus.Logger, r *mux.Router, d *database.Database) *Service {
	return &Service{Logger: l, Router: r, Database: d}
}

/*
Create Announcement (POST)

	http handler
	creates a new announcement and adds it to the database
*/
func (s *Service) CreateAnnouncement() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// Get user UUID from header
		uuid := r.Header.Get("UUID")
		if uuid == "" {
			s.Logger.Error("Failed to get uuid from authorization token")
			http.Error(rw, `{ "msg": "unauthorized" }`, http.StatusUnauthorized)
			return
		}

		// Decode request
		var req models.AnnouncementDao
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.Logger.Error("Failed to decode request: ", err.Error())
			http.Error(rw, `{"msg": "invalid request body"}`, http.StatusBadRequest)
			return
		}

		// Set default values for announcement
		timestamp := primitive.NewDateTimeFromTime(time.Now())

		// Default to title emphasis if not specified
		textEmphasis := models.EmphasisTitle
		if req.TextEmphasis == nil {
			req.TextEmphasis = &textEmphasis
		}

		// Get default styles
		defaultTitleStyle, defaultSubtitleStyle := utils.GetDefaultTextStyles()

		// Create announcement DAO object
		req.Creator = &uuid

		// Default created event status
		status := models.StatusPending
		req.Status = &status

		// Copy fields from request where provided
		if req.Title == nil {
			s.Logger.Error("request is missing announcement title")
			http.Error(rw, `{"msg":"announcement title required"}`, http.StatusBadRequest)
			return
		}
		if req.Subtitle == nil {
			s.Logger.Error("request is missing announcement subtitle")
			http.Error(rw, `{"msg":"announcement subtitle required"}`, http.StatusBadRequest)
			return
		}
		if req.MediaURL == nil {
			s.Logger.Error("request is missing media url")
			http.Error(rw, `{"msg":"announcement media url required"}`, http.StatusBadRequest)
			return
		}
		if req.MediaType == nil {
			s.Logger.Error("request is missing media type")
			http.Error(rw, `{"msg":"announcement media type required"}`, http.StatusBadRequest)
			return
		}
		if req.ActionButton == nil {
			s.Logger.Error("request is missing action button")
			http.Error(rw, `{"msg":"announcement action button required"}`, http.StatusBadRequest)
			return
		}
		if req.Position == nil {
			defaultPos := models.PositionConfig{
				Alignment:   "left",
				VerticalPos: "bottom",
				Width:       "100%",
				Height:      "100%",
			}
			req.Position = &defaultPos
		}
		if req.Scope == nil {
			s.Logger.Error("request is missing scope")
			http.Error(rw, `{"msg":"announcement Scope required"}`, http.StatusBadRequest)
			return
		} else {
			if *req.Scope == models.ScopeLocal && req.Location == nil {
				s.Logger.Error("request is missing location")
				http.Error(rw, `{"msg":"announcement with local scope requires a location"}`, http.StatusBadRequest)
				return
			}
		}
		if req.ActiveDate == nil {
			// Default to current time
			req.ActiveDate = &timestamp
		}
		if req.ExpiryDate == nil {
			// Default to 30 days
			expiry := primitive.NewDateTimeFromTime(time.Now().Add((24 * time.Hour) * 30))
			req.ExpiryDate = &expiry
		}

		// Fill missing text styles
		utils.FillMissingTextStyles(req.TitleStyle, defaultTitleStyle)
		utils.FillMissingTextStyles(req.SubtitleStyle, defaultSubtitleStyle)

		// Last time updated announcement
		req.UpdatedAt = &timestamp

		// Insert announcement into database
		res, err := s.InsertAnnouncement(r.Context(), &req)
		if err != nil || res == nil {
			s.Logger.Error("Failed to insert announcement: ", err.Error())
			http.Error(rw, `{"msg": "failed to create announcement"}`, http.StatusInternalServerError)
			return
		}

		// Return newly created announcement ID
		rw.WriteHeader(http.StatusCreated)
		rw.Write(fmt.Appendf(nil, `{"id": "%s"}`, res.Hex()))
	}
}

/*
Get Announcement (GET)

	http handler
	retrieves a specific announcement by ID
*/
func (s *Service) GetAnnouncement() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// Extract ID from URL
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(rw, `{"msg": "invalid announcement id"}`, http.StatusBadRequest)
			return
		}

		// Convert string ID to ObjectID
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			http.Error(rw, `{"msg": "invalid announcement id format"}`, http.StatusBadRequest)
			return
		}

		// Create context with timeout
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		// Get announcement with aggregation
		announcement, err := aggregations.AggregateAnnouncement(ctx, objID, s.Database)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				http.Error(rw, `{"msg": "announcement not found"}`, http.StatusNotFound)
				return
			}
			s.Logger.Error("Failed to fetch announcement: ", err.Error())
			http.Error(rw, `{"msg": "failed to fetch announcement"}`, http.StatusInternalServerError)
			return
		}

		// Return announcement data
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(announcement)
	}
}

/*
Get Active Announcements (GET)

	http handler
	retrieves all active announcements, with optional location filtering
*/
func (s *Service) GetAnnouncements() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// Create context with timeout
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		// Parse query parameters for pagination and location
		// limitStr := r.URL.Query().Get("limit")
		// limit := 10 // Default limit
		// if limitStr != "" {
		// 	if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
		// 		limit = val
		// 	}
		// }

		// Get announcements with location filter applied
		announcements, err := aggregations.AggregateAnnouncements(ctx, bson.M{}, &options.AggregateOptions{}, s.Database)
		if err != nil {
			s.Logger.Error("Failed to fetch announcements: ", err.Error())
			http.Error(rw, `{"msg": "failed to fetch announcements"}`, http.StatusInternalServerError)
			return
		}

		// Handle empty result set
		if len(announcements) == 0 {
			response := models.AnnouncementsResponse{
				TotalAnnouncements: 0,
				Announcements:      []models.Announcement{},
			}
			json.NewEncoder(rw).Encode(response)
			return
		}

		// Return announcements
		response := models.AnnouncementsResponse{
			TotalAnnouncements: len(announcements),
			Announcements:      announcements,
		}

		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(response)
	}
}

/*
Update Announcement (PUT)

	http handler
	updates an existing announcement
*/
func (s *Service) UpdateAnnouncement() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// Get user UUID from header
		uuid := r.Header.Get("UUID")
		if uuid == "" {
			http.Error(rw, `{"msg": "unauthorized"}`, http.StatusUnauthorized)
			return
		}

		// Extract ID from URL
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(rw, `{"msg": "invalid announcement id"}`, http.StatusBadRequest)
			return
		}

		// Convert string ID to ObjectID
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			http.Error(rw, `{"msg": "invalid announcement id format"}`, http.StatusBadRequest)
			return
		}

		// Create context with timeout
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		// Decode update request
		var updateData models.AnnouncementDao
		if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
			s.Logger.Error("Failed to decode request: ", err.Error())
			http.Error(rw, `{"msg": "invalid request body"}`, http.StatusBadRequest)
			return
		}

		// Build update document
		updateFields := bson.M{}
		timestamp := time.Now().Unix()

		// Update fields if provided in request
		if updateData.Title != nil {
			updateFields["title"] = *updateData.Title
		}
		if updateData.Subtitle != nil {
			updateFields["subtitle"] = *updateData.Subtitle
		}
		if updateData.TextEmphasis != nil {
			updateFields["text_emphasis"] = *updateData.TextEmphasis
		}
		if updateData.TitleStyle != nil {
			updateFields["title_style"] = *updateData.TitleStyle
		}
		if updateData.SubtitleStyle != nil {
			updateFields["subtitle_style"] = *updateData.SubtitleStyle
		}
		if updateData.MediaURL != nil {
			updateFields["media_url"] = *updateData.MediaURL
		}
		if updateData.MediaType != nil {
			updateFields["media_type"] = *updateData.MediaType
		}
		if updateData.ActionButton != nil {
			updateFields["action_button"] = *updateData.ActionButton
		}
		if updateData.Position != nil {
			updateFields["position"] = *updateData.Position
		}
		if updateData.Scope != nil {
			updateFields["scope"] = *updateData.Scope
		}
		if updateData.Location != nil {
			updateFields["location"] = *updateData.Location
		}
		if updateData.Status != nil {
			updateFields["status"] = *updateData.Status
		}
		if updateData.ActiveDate != nil {
			updateFields["active_date"] = *updateData.ActiveDate
		}
		if updateData.ExpiryDate != nil {
			updateFields["expiry_date"] = *updateData.ExpiryDate
		}

		// Always update the updated_at timestamp
		updateFields["updated_at"] = timestamp

		// If there's nothing to update, return early
		if len(updateFields) == 0 {
			http.Error(rw, `{"msg": "no updates provided"}`, http.StatusBadRequest)
			return
		}

		// Perform update
		update := bson.M{"$set": updateFields}
		if err := s.ModifyAnnouncement(ctx, bson.M{"_id": objID}, update); err != nil {
			s.Logger.Error("Failed to update announcement: ", err.Error())
			http.Error(rw, `{"msg": "failed to update announcement"}`, http.StatusInternalServerError)
			return
		}

		// Get updated announcement
		updatedAnnouncement, err := aggregations.AggregateAnnouncement(ctx, objID, s.Database)
		if err != nil {
			s.Logger.Error("Failed to fetch updated announcement: ", err.Error())
			http.Error(rw, `{"msg": "announcement updated but failed to retrieve"}`, http.StatusInternalServerError)
			return
		}

		// Return updated announcement
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(updatedAnnouncement)
	}
}

/*
Delete Announcement (DELETE)

	http handler
	deletes an announcement from the database
*/
func (s *Service) DeleteAnnouncement() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// Get user UUID from header
		uuid := r.Header.Get("UUID")
		if uuid == "" {
			http.Error(rw, `{"msg": "unauthorized"}`, http.StatusUnauthorized)
			return
		}

		// Extract ID from URL
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(rw, `{"msg": "invalid announcement id"}`, http.StatusBadRequest)
			return
		}

		// Convert string ID to ObjectID
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			http.Error(rw, `{"msg": "invalid announcement id format"}`, http.StatusBadRequest)
			return
		}

		// Create context with timeout
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		// Delete the announcement
		if err := s.RemoveAnnouncement(ctx, bson.M{"_id": objID}); err != nil {
			s.Logger.Error("Failed to delete announcement: ", err.Error())
			http.Error(rw, `{"msg": "failed to delete announcement"}`, http.StatusInternalServerError)
			return
		}

		// Return success message
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{"msg": "announcement deleted successfully"}`))
	}
}
