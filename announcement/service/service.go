package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"olympsis-server/database"
	"olympsis-server/utils"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/olympsis/models"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Service struct {
	Database     *database.Database
	Logger       *logrus.Logger
	Router       *mux.Router
	Notification *utils.NotificationInterface
}

/*
Create new announcement service struct
*/
func NewAnnouncementService(l *logrus.Logger, r *mux.Router, d *database.Database) *Service {
	return &Service{Logger: l, Router: r, Database: d}
}

/*
Create Announcement Data (POST)
*/
func (a *Service) CreateAnnouncement() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		uuid := r.Header.Get("UUID")

		// Decode request
		var req models.AnnouncementDao
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			a.Logger.Error("Failed to decode request: " + err.Error())
			http.Error(rw, `{ "msg": "failed to decode announcement" }`, http.StatusBadRequest)
			return
		}

		// Validate required fields
		if req.Title == nil || req.MediaURL == nil {
			a.Logger.Error("Missing required fields")
			http.Error(rw, `{ "msg": "title and media_url are required" }`, http.StatusBadRequest)
			return
		}

		// Set creation metadata
		timestamp := time.Now().Unix()
		req.CreatedAt = &timestamp
		req.CreatedBy = &uuid

		// Insert the announcement
		id, err := a.InsertAnnouncement(ctx, &req)
		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				a.Logger.Error("Timeout inserting announcement")
				http.Error(rw, `{ "msg": "operation timed out" }`, http.StatusGatewayTimeout)
				return
			}
			a.Logger.Error("Failed to insert announcement: ", err.Error())
			http.Error(rw, `{ "msg": "failed to insert announcement" }`, http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusCreated)
		rw.Write([]byte(fmt.Sprintf(`{"id": "%s"}`, id.Hex())))
	}
}

/*
Get Announcements (GET)
*/
func (a *Service) GetAnnouncements() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// Check if location parameters are provided
		hasLocation := false
		locationQuery := bson.M{}

		// Check for coordinates
		longitude, err1 := strconv.ParseFloat(r.URL.Query().Get("longitude"), 64)
		latitude, err2 := strconv.ParseFloat(r.URL.Query().Get("latitude"), 64)
		if err1 == nil && err2 == nil && longitude != 0 && latitude != 0 {
			hasLocation = true
			// Default max distance to 5000m
			maxDistance := int64(5000)

			locationQuery = bson.M{
				"location": bson.M{
					"$near": bson.M{
						"$geometry": models.GeoJSON{
							Type:        "Point",
							Coordinates: []float64{longitude, latitude},
						},
						"$maxDistance": maxDistance,
					},
				},
			}
		}

		// Check for region-based filtering
		country := r.URL.Query().Get("country")
		state := r.URL.Query().Get("state")
		city := r.URL.Query().Get("city")

		// Build region filter
		regionFilter := bson.M{}
		if country != "" {
			hasLocation = true
			regionFilter["country"] = country
			if state != "" {
				regionFilter["state"] = state
				if city != "" {
					regionFilter["city"] = city
				}
			}
		}

		// Pagination parameters
		skip, _ := strconv.ParseInt(r.URL.Query().Get("skip"), 0, 16)
		limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 0, 16)
		if limit == 0 {
			limit = 20
		}

		// Current time for filtering active announcements
		currentTime := time.Now().Unix()

		// Build the main filter
		filter := bson.M{
			"start_time": bson.M{"$lte": currentTime},
			"$or": []bson.M{
				{"end_time": bson.M{"$exists": false}},
				{"end_time": bson.M{"$gt": currentTime}},
			},
		}

		// Add location filtering if provided
		if hasLocation {
			locationFilters := []bson.M{}

			// Only add non-empty filters
			if len(locationQuery) > 0 {
				locationFilters = append(locationFilters, locationQuery)
			}

			if len(regionFilter) > 0 {
				locationFilters = append(locationFilters, regionFilter)
			}

			// Add a filter for announcements with no location constraints
			locationFilters = append(locationFilters, bson.M{
				"$and": []bson.M{
					{"location": bson.M{"$exists": false}},
					{"country": bson.M{"$exists": false}},
					{"state": bson.M{"$exists": false}},
					{"city": bson.M{"$exists": false}},
				},
			})

			if len(locationFilters) > 0 {
				filter["$or"] = locationFilters
			}
		}

		// Find announcements with pagination
		options := options.Find().
			SetSort(bson.D{{Key: "start_time", Value: -1}}).
			SetSkip(skip).
			SetLimit(limit)

		cursor, err := a.Database.AnnouncementCol.Find(r.Context(), filter, options)
		if err != nil {
			a.Logger.Error("Failed to find announcements", err.Error())
			http.Error(rw, `{ "msg": "failed to find announcements" }`, http.StatusInternalServerError)
			return
		}
		defer cursor.Close(r.Context())

		var announcements []models.Announcement
		if err := cursor.All(r.Context(), &announcements); err != nil {
			a.Logger.Error("Failed to decode announcements", err.Error())
			http.Error(rw, `{ "msg": "failed to decode announcements" }`, http.StatusInternalServerError)
			return
		}

		if len(announcements) == 0 {
			rw.WriteHeader(http.StatusOK)
			resp := models.AnnouncementsResponse{
				TotalAnnouncements: 0,
				Announcements:      []models.Announcement{},
			}
			json.NewEncoder(rw).Encode(resp)
			return
		}

		resp := models.AnnouncementsResponse{
			TotalAnnouncements: int16(len(announcements)),
			Announcements:      announcements,
		}

		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(resp)
	}
}

/*
Update Announcement (PUT)
*/
func (a *Service) UpdateAnnouncement() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(rw, `{ "msg": "invalid announcement id" }`, http.StatusBadRequest)
			return
		}

		var req models.AnnouncementDao
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(rw, `{ "msg": "failed to decode request" }`, http.StatusBadRequest)
			return
		}

		oid, _ := primitive.ObjectIDFromHex(id)

		// Build update document
		update := bson.M{"$set": bson.M{}}
		if req.Title != nil {
			update["$set"].(bson.M)["title"] = *req.Title
		}
		if req.Body != nil {
			update["$set"].(bson.M)["body"] = *req.Body
		}
		if req.MediaURL != nil {
			update["$set"].(bson.M)["media_url"] = *req.MediaURL
		}
		if req.Action != nil {
			update["$set"].(bson.M)["action"] = *req.Action
		}
		if req.ActionURL != nil {
			update["$set"].(bson.M)["action_url"] = *req.ActionURL
		}
		if req.StartTime != nil {
			update["$set"].(bson.M)["start_time"] = *req.StartTime
		}
		if req.EndTime != nil {
			update["$set"].(bson.M)["end_time"] = *req.EndTime
		}
		if req.Location != nil {
			update["$set"].(bson.M)["location"] = *req.Location
		}
		if req.Country != nil {
			update["$set"].(bson.M)["country"] = *req.Country
		}
		if req.State != nil {
			update["$set"].(bson.M)["state"] = *req.State
		}
		if req.City != nil {
			update["$set"].(bson.M)["city"] = *req.City
		}

		err := a.ModifyAnnouncement(r.Context(), bson.M{"_id": oid}, update)
		if err != nil {
			a.Logger.Error("Failed to update announcement", err.Error())
			http.Error(rw, `{ "msg": "failed to update announcement" }`, http.StatusInternalServerError)
			return
		}

		announcement, _ := a.FindAnnouncement(r.Context(), bson.M{"_id": oid})
		if announcement != nil {
			json.NewEncoder(rw).Encode(announcement)
		} else {
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte(`{ "msg": "announcement updated" }`))
		}
	}
}
