package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"olympsis-server/aggregations"
	"olympsis-server/database"
	"olympsis-server/utils"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/olympsis/models"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

/*
Event Service Struct
*/
type Service struct {
	Database *database.Database // database for read/write operations
	Logger   *logrus.Logger     // logger for logging errors
	Router   *mux.Router        // router for handling incoming requests
}

/*
Create new event service struct
*/
func NewEventService(l *logrus.Logger, r *mux.Router, d *database.Database) *Service {
	return &Service{Logger: l, Router: r, Database: d}
}

/*
Create Event Data (POST)

	http handler
	create an event and add it into the database
*/
func (e *Service) CreateEvent() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		uuid := r.Header.Get("UUID")

		// decode request
		var req models.EventDao
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			e.Logger.Error("Failed to decode request: " + err.Error())
			http.Error(rw, `{ "msg": "failed to decode event" }`, http.StatusBadRequest)
			return
		}

		// create timestamp & participant
		newID := primitive.NewObjectID()
		rsvpStatus := int8(1)
		timestamp := time.Now().Unix()
		participant := models.ParticipantDao{
			ID:        &newID,
			UUID:      &uuid,
			Status:    &rsvpStatus,
			CreatedAt: &timestamp,
		}

		// create new event model
		event := models.EventDao{
			Type:            req.Type,
			Poster:          &uuid,
			Organizers:      req.Organizers,
			Venues:          req.Venues,
			ImageURL:        req.ImageURL,
			Title:           req.Title,
			Body:            req.Body,
			Sports:          req.Sports,
			StartTime:       req.StartTime,
			StopTime:        req.StopTime,
			MinParticipants: req.MinParticipants,
			MaxParticipants: req.MaxParticipants,
			Participants:    &[]models.ParticipantDao{participant},
			Level:           req.Level,
			Visibility:      req.Visibility,
			ExternalLink:    req.ExternalLink,
			CreatedAt:       &timestamp,
		}

		// insert event into database
		id, err := e.InsertEvent(context.Background(), &event)
		if err != nil {
			e.Logger.Error("Failed to insert event: ", err.Error())
			http.Error(rw, `{ "msg": "failed to insert event" }`, http.StatusInternalServerError)
			return
		}

		// TODO: NOTIFICATION UPDATE
		// subscribe owner to notifications
		// utils.CreateNotificationTopic(id.Hex())
		// utils.AddTokenToTopic(id.Hex(), uuid)

		// notify all of the organizers and their members about this new event
		// organizers := *event.Organizers
		// for i := range organizers {
		// 	organizer := organizers[i]
		// 	note := models.Notification{
		// 		Title: "New Event Created",
		// 		Body:  *event.Title,
		// 		Topic: organizer.ID.Hex(),
		// 		Data:  fmt.Sprintf(`"id": "%s"`, id.Hex()),
		// 	}
		// 	utils.SendNotificationToTopic(&note)
		// }

		rw.WriteHeader(http.StatusCreated)
		rw.Write([]byte(fmt.Sprintf(`{"id": "%s"}`, id.Hex())))
	}
}

/*
Get Event (GET)

	http handler
	get an event by it's id
*/
func (e *Service) GetEvent() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(rw, `{ "msg": "no/bad event id found in request" }`, http.StatusBadRequest)
			return
		}

		oid, _ := primitive.ObjectIDFromHex(id)

		event, err := aggregations.AggregateEvent(oid, e.Database)
		if err != nil {
			e.Logger.Error("Failed to find event", err.Error())
			http.Error(rw, `{ "msg": "failed to find event" }`, http.StatusInternalServerError)
		}

		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(event)
	}
}

/*
Get Events (GET)

	http handler
	gets a list of events by a location
*/
func (e *Service) GetEventsByLocation() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// longitude query
		longitude, err := strconv.ParseFloat(r.URL.Query().Get("longitude"), 64)
		if err != nil || longitude == 0 {
			e.Logger.Error("Failed to decode longitude. Error: ", err.Error())
			http.Error(rw, `{ "msg": "missing query param - longitude" }`, http.StatusBadRequest)
			return
		}

		// latitude query
		latitude, err := strconv.ParseFloat(r.URL.Query().Get("latitude"), 64)
		if err != nil || latitude == 0 {
			e.Logger.Error("Failed to decode latitude. Error: ", err.Error())
			http.Error(rw, `{ "msg": "missing query param - latitude" }`, http.StatusBadRequest)
			return
		}

		// radius query
		radius, err := strconv.ParseInt(r.URL.Query().Get("radius"), 0, 32)
		if (err != nil) || (radius == 0) {
			e.Logger.Error("Failed to decode location queries. Error: ", err.Error())
			http.Error(rw, `{ "msg": "missing query param - radius" }`, http.StatusBadRequest)
			return
		}

		// sports & status query params
		sports_query := r.URL.Query().Get("sports")
		status_query := r.URL.Query().Get("status")
		if sports_query == "" || status_query == "" {
			e.Logger.Error("Failed to decode sports & status queries.")
			http.Error(rw, `{ "msg": "missing query param - sports or status" }`, http.StatusBadRequest)
			return
		}

		// pagination query params
		skip := 0
		limit := 0
		skip_query, err := strconv.ParseInt(r.URL.Query().Get("skip"), 0, 16)
		if err != nil {
			skip = 0
		}
		limit_query, err := strconv.ParseInt(r.URL.Query().Get("limit"), 0, 16)
		if err != nil || limit_query == 0 {
			limit = 20
		}

		// create local data
		status := 1
		skip = int(skip_query)
		limit = int(limit_query)
		sports := strings.Split(sports_query, ",")
		location := models.GeoJSON{
			Type:        "Point",
			Coordinates: []float64{longitude, latitude},
		}

		switch status_query {
		case "completed":
			status = 0
		default:
			status = 1
		}

		var wg sync.WaitGroup
		var wgError *error
		var user *models.UserData
		var venues []primitive.ObjectID

		// venues go-routine
		wg.Add(1)
		go func() {
			defer wg.Done()

			// documents filter
			filter := bson.M{
				"location": bson.M{
					"$near": bson.M{
						"$geometry":    location,
						"$maxDistance": radius,
					},
				},
				"sports": bson.M{
					"$in": sports,
				},
			}

			// fetch the nearby venues
			ctx := r.Context()
			cursor, err := e.Database.FieldCol.Find(ctx, filter)
			if err != nil {
				if err == mongo.ErrNoDocuments {
					wgError = &err
					venues = []primitive.ObjectID{}
				} else {
					wgError = &err
					venues = []primitive.ObjectID{}
					e.Logger.Error("Failed to find venues. Error: ", err.Error())
				}
			}
			defer cursor.Close(ctx)

			// decode fields
			venues = []primitive.ObjectID{}
			for cursor.Next(ctx) {
				var venue models.Venue
				if err := cursor.Decode(&venue); err != nil {
					e.Logger.Error("Failed to decode field: ", err)
					continue
				}
				venues = append(venues, venue.ID)
			}
		}()

		// user data go-routine
		wg.Add(1)
		go func() {
			defer wg.Done()

			// grab user data
			uuid := r.Header.Get("UUID")
			user, err = aggregations.AggregateUser(&uuid, e.Database)
			if err != nil || user == nil {
				wgError = &err
				e.Logger.Error("Failed to get user data. Error: ", err.Error())
				return
			}
		}()

		// wait on go-routines & handle error
		wg.Wait()
		if wgError != nil {
			http.Error(rw, `{ "msg": "Failed to get venue data." }`, http.StatusInternalServerError)
			return
		}

		// clubs & organizations
		clubs := []primitive.ObjectID{}
		orgs := []primitive.ObjectID{}
		if user.Clubs != nil {
			clubs = append(clubs, *user.Clubs...)
		}
		if user.Organizations != nil {
			orgs = append(orgs, *user.Organizations...)
		}

		// find the events
		events, err := aggregations.AggregateEvents(
			user.UUID,
			sports,
			location,
			venues,
			clubs,
			orgs,
			int(radius),
			int(limit),
			int(skip),
			status,
			e.Database,
		)
		if err != nil || events == nil { // unexpected error
			e.Logger.Error("Failed to find events. Error: ", err.Error())
			http.Error(rw, `{ "msg": "failed to find events" }`, http.StatusInternalServerError)
			return
		}
		if len(*events) == 0 { // no content error
			http.Error(rw, `{ "msg": "no events found" }`, http.StatusNoContent)
			return
		}

		resp := models.EventsResponse{
			TotalEvents: int16(len(*events)),
			Events:      *events,
		}

		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(resp)
	}
}

/*
Get Events (GET)

	http handler
	gets a list of events by a field id
*/
func (e *Service) GetEventsByField() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "no field id found in request." }`))
			return
		}
		oid, _ := primitive.ObjectIDFromHex(id)
		events, err := aggregations.AggregateEventsByField(oid, 100, e.Database)
		if err != nil {
			e.Logger.Error("Failed to find events", err.Error())
			http.Error(rw, `{ "msg": "failed to find events"}`, http.StatusInternalServerError)
			return
		}
		if len(*events) == 0 {
			http.Error(rw, `{ "msg": "no events found" }`, http.StatusNoContent)
			return
		}

		resp := models.EventsResponse{
			TotalEvents: int16(len(*events)),
			Events:      *events,
		}

		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(resp)
	}
}

/*
Update Event (UPDATE)

	http handler
	updates an event in the database
*/
func (e *Service) UpdateAnEvent() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// grab event id from path
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(rw, `{ "msg": "no event Id found in request." }`, http.StatusInternalServerError)
			return
		}

		// decode request
		var req models.EventDao
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(rw, `{ "msg": "failed to decode user" }`, http.StatusBadRequest)
			return
		}

		oid, _ := primitive.ObjectIDFromHex(id)
		filter := bson.M{"_id": oid}
		changes := bson.M{}
		updates := bson.M{"$set": changes}

		if req.Title != nil {
			changes["title"] = req.Title
		}
		if req.Body != nil {
			changes["body"] = req.Body
		}
		if req.ImageURL != nil {
			changes["image_url"] = req.ImageURL
		}
		if req.StartTime != nil {
			changes["start_time"] = req.StartTime
		}
		if req.Visibility != nil {
			changes["visibility"] = req.Visibility
		}

		if req.MinParticipants != nil {
			changes["min_participants"] = req.MinParticipants
		}

		if req.MaxParticipants != nil {
			changes["max_participants"] = req.MaxParticipants
		}

		if req.Level != nil {
			changes["level"] = req.Level
		}

		if req.ExternalLink != nil {
			changes["external_link"] = req.ExternalLink
		}

		if req.StopTime == nil {
			updates["$unset"] = bson.M{"stop_time": 1}
		} else {
			changes["stop_time"] = req.StopTime
		}

		err = e.UpdateEvent(context.Background(), filter, updates)
		if err != nil {
			e.Logger.Error("failed to find event", err.Error())
			http.Error(rw, `{ "msg": "failed to update event" }`, http.StatusInternalServerError)
			return
		}

		event, err := e.FindEvent(context.Background(), bson.M{"_id": oid})
		if err != nil {
			e.Logger.Error("failed to find event", err.Error())
			http.Error(rw, `{ "msg": "failed to find event" }`, http.StatusInternalServerError)
			return
		}

		json.NewEncoder(rw).Encode(event)
		rw.WriteHeader(http.StatusOK)
	}
}

/*
Delete Event Data (Delete)

	http handler
	deletes an event from the database
*/
func (e *Service) DeleteAnEvent() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// grab id from path
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(rw, `{ "msg": "bad event id" }`, http.StatusBadRequest)
			return
		}

		oid, _ := primitive.ObjectIDFromHex(id)
		filter := bson.M{"_id": oid}
		err := e.DeleteEvent(context.Background(), filter)
		if err != nil {
			e.Logger.Error("failed to delete event", err.Error())
			http.Error(rw, `{ "msg": "failed to delete event" }`, http.StatusInternalServerError)
		}

		// delete notification topic
		utils.DeleteNotificationTopic(id)
		rw.WriteHeader(http.StatusOK)
	}
}

/*
Add Participant (POST)

	http handler
	adds a participant to an event
	adds a participant to an event's topic
*/
func (e *Service) AddParticipant() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		uuid := r.Header.Get("UUID")

		// grab id from path
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(rw, `"msg": "bad event id" `, http.StatusBadRequest)
			return
		}

		// decode request
		var req models.ParticipantDao
		json.NewDecoder(r.Body).Decode(&req)

		oid, _ := primitive.ObjectIDFromHex(id)
		timestamp := time.Now().Unix()

		// find event data in database
		filter := bson.M{"_id": oid}
		event, err := e.FindEvent(context.Background(), filter)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				http.Error(rw, `{ "msg": "event not found" }`, http.StatusNotFound)
				return
			}
			e.Logger.Error("failed to find event", err.Error())
			http.Error(rw, `{ "msg": "failed to find event" }`, http.StatusInternalServerError)
			return
		}

		// check if participant already exists
		participants := *event.Participants
		for i := range participants {
			if *participants[i].UUID == uuid {
				rw.WriteHeader(http.StatusOK)
				rw.Write([]byte(fmt.Sprintf(`{ "id": "%s" }`, participants[i].ID)))
				return
			}
		}

		// if event is full
		if event.MaxParticipants != nil {
			if *event.MaxParticipants != 0 {
				if len(participants) >= int(*event.MaxParticipants) {
					http.Error(rw, `{ "msg": "event capacity is full" }`, http.StatusBadRequest)
					return
				}
			}
		}

		// participant object
		newID := primitive.NewObjectID()
		part := &models.ParticipantDao{
			ID:        &newID,
			UUID:      &uuid,
			Status:    req.Status,
			CreatedAt: &timestamp,
		}
		change := bson.M{"$push": bson.M{"participants": part}}
		err = e.UpdateEvent(context.Background(), filter, change)
		if err != nil {
			e.Logger.Error("failed to update event", err.Error())
			http.Error(rw, "failed to update event", http.StatusInternalServerError)
			return
		}

		// notify all participants
		notif := models.Notification{
			Title: *event.Title,
			Body:  "New Participant RSVP'd!",
			Topic: id,
		}
		utils.SendNotificationToTopic(&notif)

		// subscribe user to notifications
		utils.AddTokenToTopic(id, uuid)

		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{ "msg": "successful" }`))
	}
}

/*
Remove Participant (DELETE)

	http handler
	removes participant from the event object
	removes participant from the event topic
*/
func (e *Service) RemoveParticipant() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		uuid := r.Header.Get("UUID")

		// grab id from path
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(rw, `{ "msg": "bad event id" }`, http.StatusBadRequest)
			return
		}

		oid, _ := primitive.ObjectIDFromHex(id)
		match := bson.M{"_id": oid}
		change := bson.M{"$pull": bson.M{"participants": bson.M{"uuid": uuid}}}

		err := e.UpdateEvent(context.Background(), match, change)
		if err != nil {
			e.Logger.Error("failed to update event: ", err.Error())
			http.Error(rw, `{ "msg": "failed to update event" }`, http.StatusInternalServerError)
			return
		}

		// unsubscribe user from notifications
		utils.RemoveTokenFromTopic(id, uuid)
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{ "msg": "successful" }`))
	}
}

/*
Notify Event Participants (POST)

	http handler
	sends a custom notification to the event's participants
*/
func (e *Service) NotifyParticipants() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// grab id from path
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(rw, `{ "msg": "bad event id" }`, http.StatusBadRequest)
			return
		}

		// decode request
		var req models.Notification
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			e.Logger.Error("failed to decode notification", err.Error())
			http.Error(rw, `{ "msg": "failed to decode notification" }`, http.StatusInternalServerError)
			return
		}

		req.Topic = id
		utils.SendNotificationToTopic(&req)
		rw.WriteHeader(http.StatusOK)
	}
}

/*
Notify Club members (POST)

	http handler
	notifies all club members of an event
*/
func (e *Service) NotifyClubMembers() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// grab id from path
		vars := mux.Vars(r)
		id := vars["id"]
		if len(vars["id"]) < 24 {
			http.Error(rw, `{ "msg": "bad event id" }`, http.StatusBadRequest)
			return
		}

		// decode request
		var req models.Notification
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(rw, `{ "msg": "failed to decode request" }`, http.StatusInternalServerError)
			return
		}

		req.Topic = id
		utils.SendNotificationToTopic(&req)
		rw.WriteHeader(http.StatusOK)
	}
}

/*
Location (POST)

	http handler
	gets all of the events and venues in a location
*/
func (e *Service) Location() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// longitude query
		longitude, err := strconv.ParseFloat(r.URL.Query().Get("longitude"), 64)
		if err != nil || longitude == 0 {
			e.Logger.Error("Failed to decode longitude. Error: ", err.Error())
			http.Error(rw, `{ "msg": "missing query param - longitude" }`, http.StatusBadRequest)
			return
		}

		// latitude query
		latitude, err := strconv.ParseFloat(r.URL.Query().Get("latitude"), 64)
		if err != nil || latitude == 0 {
			e.Logger.Error("Failed to decode latitude. Error: ", err.Error())
			http.Error(rw, `{ "msg": "missing query param - latitude" }`, http.StatusBadRequest)
			return
		}

		// radius query
		radius, err := strconv.ParseInt(r.URL.Query().Get("radius"), 0, 32)
		if (err != nil) || (radius == 0) {
			e.Logger.Error("Failed to decode location queries. Error: ", err.Error())
			http.Error(rw, `{ "msg": "missing query param - radius" }`, http.StatusBadRequest)
			return
		}

		// sports & status query params
		sports_query := r.URL.Query().Get("sports")
		status_query := r.URL.Query().Get("status")
		if sports_query == "" || status_query == "" {
			e.Logger.Error("Failed to decode sports & status queries.")
			http.Error(rw, `{ "msg": "missing query param - sports or status" }`, http.StatusBadRequest)
			return
		}

		// pagination query params
		skip := 0
		limit := 0
		skip_query, err := strconv.ParseInt(r.URL.Query().Get("skip"), 0, 16)
		if err != nil {
			skip = 0
		}
		limit_query, err := strconv.ParseInt(r.URL.Query().Get("limit"), 0, 16)
		if err != nil || limit_query == 0 {
			limit = 20
		}

		// create local data
		status := 1
		skip = int(skip_query)
		limit = int(limit_query)
		sports := strings.Split(sports_query, ",")
		location := models.GeoJSON{
			Type:        "Point",
			Coordinates: []float64{longitude, latitude},
		}

		switch status_query {
		case "completed":
			status = 0
		default:
			status = 1
		}

		var wg sync.WaitGroup
		var wgError *error

		var user models.UserData
		var venues []models.Venue
		var venueIDs []primitive.ObjectID

		// venues go-routine
		wg.Add(1)
		go func() {
			defer wg.Done()

			// documents filter
			filter := bson.M{
				"location": bson.M{
					"$near": bson.M{
						"$geometry":    location,
						"$maxDistance": radius,
					},
				},
				"sports": bson.M{
					"$in": sports,
				},
			}

			// fetch the nearby venues
			ctx := r.Context()
			cursor, err := e.Database.FieldCol.Find(ctx, filter)
			if err != nil {
				if err == mongo.ErrNoDocuments {
					wgError = &err
					venues = []models.Venue{}
					venueIDs = []primitive.ObjectID{}
				} else {
					wgError = &err
					venues = []models.Venue{}
					venueIDs = []primitive.ObjectID{}
					e.Logger.Error("Failed to find venues. Error: ", err.Error())
				}
			}
			defer cursor.Close(ctx)

			// decode fields
			for cursor.Next(ctx) {
				var venue models.Venue
				if err := cursor.Decode(&venue); err != nil {
					e.Logger.Error("Failed to decode field: Error: ", err)
					continue
				}
				venues = append(venues, venue)
				venueIDs = append(venueIDs, venue.ID)
			}
		}()

		// user data go-routine
		wg.Add(1)
		go func() {
			defer wg.Done()

			// grab user data
			uuid := r.Header.Get("UUID")
			user, err := aggregations.AggregateUser(&uuid, e.Database)
			if err != nil || user == nil {
				wgError = &err
				e.Logger.Error("Failed to get user data. Error: ", err.Error())
				return
			}
		}()

		// wait on go-routines
		wg.Wait()
		if wgError != nil {
			http.Error(rw, `{ "msg": "Failed to get venue data." }`, http.StatusInternalServerError)
			return
		}

		// clubs & organizations
		clubs := []primitive.ObjectID{}
		orgs := []primitive.ObjectID{}
		if user.Clubs != nil {
			clubs = append(clubs, *user.Clubs...)
		}
		if user.Organizations != nil {
			orgs = append(orgs, *user.Organizations...)
		}

		// find the events
		events, err := aggregations.AggregateEvents(
			user.UUID,
			sports,
			location,
			venueIDs,
			clubs,
			orgs,
			int(radius),
			int(limit),
			int(skip),
			status,
			e.Database,
		)
		if err != nil || events == nil { // unexpected error
			e.Logger.Error("Failed to find events. Error: ", err.Error())
			http.Error(rw, `{ "msg": "failed to find events" }`, http.StatusInternalServerError)
			return
		}
		if len(*events) == 0 { // no content error
			http.Error(rw, `{ "msg": "no events found" }`, http.StatusNoContent)
			return
		}

		resp := models.LocationResponse{
			Venues: &venues,
			Events: events,
		}

		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(resp)
	}
}
