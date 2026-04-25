package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"olympsis-server/database"
	"olympsis-server/notifications"
	"olympsis-server/server"
	"strconv"
	"strings"
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
		var req models.Venue
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			f.Log.Errorf(`Failed to decode venue request body. Error: %s`, err.Error())
			http.Error(rw, `{ "msg": "Invalid request body" }`, http.StatusBadRequest)
			return
		}

		// Add timestamps
		now := bson.NewDateTimeFromTime(time.Now())
		req.CreatedAt = &now
		req.UpdatedAt = &now

		// Create venue in database
		res, err := f.InsertVenue(context.Background(), &req)
		if err != nil {
			f.Log.Errorf(`Failed to add venue to the database. Error: %s`, err.Error())
			http.Error(rw, `{ "msg": "failed create venue" }`, http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusCreated)
		fmt.Fprintf(rw, `{ "id" : "%s" }`, res.InsertedID.(bson.ObjectID))
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

		// Validate query headers
		err := validateVenuesQuery(r)
		if err != nil {
			http.Error(rw, fmt.Sprintf(`{ "msg": "%s" }`, err.Error()), http.StatusBadRequest)
			return
		}

		longitude, _ := strconv.ParseFloat(r.URL.Query().Get("longitude"), 64)
		latitude, _ := strconv.ParseFloat(r.URL.Query().Get("latitude"), 64)
		radius, _ := strconv.ParseFloat(r.URL.Query().Get("radius"), 64)
		sports := r.URL.Query().Get("sports")

		if longitude == 0 || latitude == 0 || sports == "" {
			http.Error(rw, "you need longitude/latitude", http.StatusBadRequest)
			return
		}

		splicedSports := strings.Split(sports, ",")

		var fields []models.Venue
		loc := models.GeoJSON{
			Type:        "Point",
			Coordinates: []float64{longitude, latitude},
		}

		filter := bson.M{
			"location": bson.M{
				"$near": bson.M{
					"$geometry":    loc,
					"$maxDistance": radius,
				},
			},
			"sports": bson.M{
				"$in": splicedSports,
			},
		}

		err = f.FindVenues(context.Background(), filter, &fields)
		if err != nil {
			f.Log.Error(err.Error())
			http.Error(rw, "failed to search fields", http.StatusInternalServerError)
			return
		}

		if len(fields) == 0 {
			http.Error(rw, "no fields found", http.StatusNoContent)
			return
		}

		resp := models.VenuesResponse{
			TotalVenues: len(fields),
			Venues:      fields,
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
		// grab club id from path
		vars := mux.Vars(r)
		if len(vars["id"]) == 0 || len(vars["id"]) < 24 {
			http.Error(rw, "bad field id", http.StatusBadRequest)
			return
		}

		id := vars["id"]

		// find field data in database
		var field models.Venue
		oid, _ := bson.ObjectIDFromHex(id)
		filter := bson.D{bson.E{Key: "_id", Value: oid}}
		err := f.FindVenue(context.Background(), filter, &field)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				http.Error(rw, "field not found", http.StatusNotFound)
				return
			}
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(field)
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

		changes["owner"] = req.Owner

		if req.Name != "" {
			changes["name"] = req.Name
		}
		if req.Description != "" {
			changes["notes"] = req.Description
		}
		if len(req.Sports) != 0 {
			changes["sports"] = req.Sports
		}
		if len(req.Images) != 0 {
			changes["images"] = req.Images
		}
		if req.Location.Type != "" {
			changes["location"] = req.Location
		}
		if req.City != "" {
			changes["city"] = req.City
		}
		if req.State != "" {
			changes["state"] = req.State
		}
		if req.Country != "" {
			changes["country"] = req.Country
		}

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
