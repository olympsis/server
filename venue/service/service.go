package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"olympsis-server/aggregations"
	"olympsis-server/database"
	"olympsis-server/notifications"
	"olympsis-server/server"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/olympsis/models"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

/*
Field Service Struct
*/
type Service struct {
	Database *database.Database

	// logrus logger to Log information about service and errors
	Log *logrus.Logger

	// mux Router to complete http requests
	Router *mux.Router

	// Notification service for sending notifications
	Notification *notifications.Service
}

/*
Create new field service struct
*/
func NewVenueService(i *server.ServerInterface) *Service {
	return &Service{
		Log:          i.Logger,
		Router:       i.Router,
		Database:     i.Database,
		Notification: i.Notification,
	}
}

/*
Create Field Data (POST)

  - Creates new field for olympsis

  - Grab request body

  - Create field data in user database

Returns:

	Http handler
		- Writes object back to client
*/
func (f *Service) CreateVenue() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// Decode request
		var req models.VenueCreationRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			f.Log.Errorf(`Failed to decode venue request body. Error: %s`, err.Error())
			http.Error(rw, `{ "msg": "invalid request body" }`, http.StatusBadRequest)
			return
		}

		// Validate required fields
		if err := validateCreateVenueRequest(&req); err != nil {
			http.Error(rw, fmt.Sprintf(`{ "msg": "%s" }`, err.Error()), http.StatusBadRequest)
			return
		}

		ctx := context.Background()
		now := bson.NewDateTimeFromTime(time.Now())
		venue := &req.Venue
		venue.CreatedAt = &now
		venue.UpdatedAt = &now

		// Insert units into their own collection and collect their IDs
		if len(req.Units) > 0 {
			// Generate the venue ID upfront so units can reference it
			venueID := bson.NewObjectID()
			venue.ID = &venueID

			docs := make([]interface{}, len(req.Units))
			unitIDs := make([]bson.ObjectID, len(req.Units))
			for i := range req.Units {
				req.Units[i].ID = bson.NewObjectID()
				req.Units[i].VenueID = venueID
				unitIDs[i] = req.Units[i].ID
				docs[i] = req.Units[i]
			}

			_, err := f.Database.VenueUnitsCollection.InsertMany(ctx, docs)
			if err != nil {
				f.Log.Errorf(`Failed to insert venue units. Error: %s`, err.Error())
				http.Error(rw, `{ "msg": "failed to create venue units" }`, http.StatusInternalServerError)
				return
			}

			venue.Units = &unitIDs
		}

		// Insert venue
		res, err := f.Database.VenuesCollection.InsertOne(ctx, venue)
		if err != nil {
			f.Log.Errorf(`Failed to add venue to the database. Error: %s`, err.Error())
			http.Error(rw, `{ "msg": "failed to create venue" }`, http.StatusInternalServerError)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusCreated)
		fmt.Fprintf(rw, `{ "id": "%s" }`, res.InsertedID.(bson.ObjectID))
	}
}

/*
Get Fields  (Get)

  - Grab params and filter fields

  - Grabs field data from database

Returns:

	Http handler
		- Writes list of fields back to client
*/
func (f *Service) GetVenues() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// Validate query parameters
		err := validateVenuesQuery(r)
		if err != nil {
			http.Error(rw, fmt.Sprintf(`{ "msg": "%s" }`, err.Error()), http.StatusBadRequest)
			return
		}

		// Parse pagination with defaults
		skip := 0
		if s := r.URL.Query().Get("skip"); s != "" {
			if val, err := strconv.ParseInt(s, 10, 32); err == nil && val >= 0 {
				skip = int(val)
			}
		}
		limit := 50
		if l := r.URL.Query().Get("limit"); l != "" {
			if val, err := strconv.ParseInt(l, 10, 32); err == nil && val > 0 {
				limit = int(val)
			}
		}

		// Build the aggregation query pipeline from request params
		queryPipeline := generateVenuesQuery(r)

		// Run the aggregation with core lookups and pagination
		venues, err := aggregations.AggregateVenues(queryPipeline, limit, skip, f.Database)
		if err != nil {
			f.Log.Error(err.Error())
			http.Error(rw, `{ "msg": "failed to search venues" }`, http.StatusInternalServerError)
			return
		}

		if len(*venues) == 0 {
			rw.WriteHeader(http.StatusNoContent)
			return
		}

		resp := models.VenuesResponse{
			TotalVenues: len(*venues),
			Venues:      *venues,
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(resp)
	}
}

/*
Get Field Data (Get)
-	Grab uuid from params
-	Grabs field data from database

Returns:

	Http handler
		- Writes user data back to client
*/
func (f *Service) GetVenue() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		if len(vars["id"]) == 0 || len(vars["id"]) < 24 {
			http.Error(rw, `{ "msg": "bad venue id" }`, http.StatusBadRequest)
			return
		}

		oid, err := bson.ObjectIDFromHex(vars["id"])
		if err != nil {
			http.Error(rw, `{ "msg": "invalid venue id" }`, http.StatusBadRequest)
			return
		}

		venue, err := aggregations.AggregateVenue(oid, f.Database)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				http.Error(rw, `{ "msg": "venue not found" }`, http.StatusNotFound)
				return
			}
			f.Log.Error(err.Error())
			http.Error(rw, `{ "msg": "failed to get venue" }`, http.StatusInternalServerError)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(venue)
	}
}

/*
Get Venue Units (GET)
  - Grab venue id from path params
  - Query venue units collection by venue_id

Returns:

	Http handler
		- Writes list of venue units back to client
*/
func (f *Service) GetVenueUnits() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		if len(vars["id"]) == 0 || len(vars["id"]) < 24 {
			http.Error(rw, `{ "msg": "bad venue id" }`, http.StatusBadRequest)
			return
		}

		oid, err := bson.ObjectIDFromHex(vars["id"])
		if err != nil {
			http.Error(rw, `{ "msg": "invalid venue id" }`, http.StatusBadRequest)
			return
		}

		filter := bson.M{"venue_id": oid}
		cursor, err := f.Database.VenueUnitsCollection.Find(context.Background(), filter)
		if err != nil {
			f.Log.Error(err.Error())
			http.Error(rw, `{ "msg": "failed to get venue units" }`, http.StatusInternalServerError)
			return
		}
		defer cursor.Close(context.Background())

		var units []models.VenueUnit
		err = cursor.All(context.Background(), &units)
		if err != nil {
			f.Log.Error(err.Error())
			http.Error(rw, `{ "msg": "failed to decode venue units" }`, http.StatusInternalServerError)
			return
		}

		if len(units) == 0 {
			rw.WriteHeader(http.StatusNoContent)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(units)
	}
}

/*
Update Field Data (PUT)

  - Updates field data

  - Grab parameters and update

Returns:

	Http handler
		- Writes updated field back to client
*/
func (f *Service) UpdateVenue() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		var req models.Venue

		// decode request
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(rw, "bad body", http.StatusBadRequest)
			return
		}

		// grab club id from path
		vars := mux.Vars(r)
		if len(vars["id"]) == 0 || len(vars["id"]) < 24 {
			http.Error(rw, "bad field id", http.StatusBadRequest)
			return
		}

		id := vars["id"]
		oid, _ := bson.ObjectIDFromHex(id)
		filter := bson.D{bson.E{Key: "_id", Value: oid}}
		changes := bson.M{}
		updates := bson.M{"$set": changes}

		if req.Name != "" {
			changes["name"] = req.Name
		}
		if req.Description != "" {
			changes["description"] = req.Description
		}
		if len(req.Sports) != 0 {
			changes["sports"] = req.Sports
		}
		if len(req.Media) != 0 {
			changes["media"] = req.Media
		}
		if req.URL != "" {
			changes["url"] = req.URL
		}
		if req.Location.Type != "" {
			changes["location"] = req.Location
		}
		if req.Address != "" {
			changes["address"] = req.Address
		}
		if req.AdministrativeArea != "" {
			changes["administrative_area"] = req.AdministrativeArea
		}
		if req.CountryCode != "" {
			changes["country_code"] = req.CountryCode
		}
		if req.Timezone != "" {
			changes["timezone"] = req.Timezone
		}

		// Set updated_at timestamp
		now := bson.NewDateTimeFromTime(time.Now())
		changes["updated_at"] = now

		var field models.Venue
		err = f.ModifyVenue(context.Background(), filter, updates, &field)
		if err != nil {
			f.Log.Error(err.Error())
			http.Error(rw, "failed to update field", http.StatusInternalServerError)
		}

		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(field)
	}
}

/*
Delete Field Data (Delete)

  - Updates field data

  - Grab parameters and update

Returns:

	Http handler
		- Writes token back to client
*/
func (f *Service) RemoveVenue() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// grab club id from path
		vars := mux.Vars(r)
		if len(vars["id"]) == 0 || len(vars["id"]) < 24 {
			http.Error(rw, "bad field id", http.StatusBadRequest)
			return
		}

		id := vars["id"]
		oid, _ := bson.ObjectIDFromHex(id)

		filter := bson.D{bson.E{Key: "_id", Value: oid}}
		err := f.DeleteField(context.Background(), filter)
		if err != nil {
			f.Log.Debug(err.Error())
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`OK`))
	}
}
