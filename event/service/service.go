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
// CreateEvent handles both single and recurring event creation
func (e *Service) CreateEvent() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		uuid := r.Header.Get("UUID")

		// Decode request
		var req models.NewEventDao
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			e.Logger.Error("Failed to decode request: " + err.Error())
			http.Error(rw, `{ "msg": "failed to decode event" }`, http.StatusBadRequest)
			return
		}

		// // Validate the request
		// if err := req.Validate(); err != nil {
		//     e.Logger.Error("Invalid request: " + err.Error())
		//     http.Error(rw, fmt.Sprintf(`{ "msg": "invalid request: %s" }`, err.Error()), http.StatusBadRequest)
		//     return
		// }

		timestamp := time.Now().Unix()
		isRecurring := req.Recurrence != nil

		// Create base event
		event := req.Event
		newID := primitive.NewObjectID()
		rsvpStatus := int8(1)
		participant := models.ParticipantDao{
			ID:        &newID,
			UUID:      &uuid,
			Status:    &rsvpStatus,
			CreatedAt: &timestamp,
		}

		event.Poster = &uuid
		event.CreatedAt = &timestamp
		event.IsRecurring = &isRecurring
		event.Participants = &[]models.ParticipantDao{participant}

		if !isRecurring {
			// Handle single event creation
			id, err := e.InsertEvent(context.Background(), &event)
			if err != nil {
				e.Logger.Error("Failed to insert event: ", err.Error())
				http.Error(rw, `{ "msg": "failed to insert event" }`, http.StatusInternalServerError)
				return
			}

			rw.WriteHeader(http.StatusCreated)
			rw.Write([]byte(fmt.Sprintf(`{"id": "%s"}`, id.Hex())))
			return
		}

		// Handle recurring event creation
		event.RecurrenceRule = &req.Recurrence.Pattern
		event.RecurrenceEnd = &req.Recurrence.EndTime

		// Insert parent event
		parentID, err := e.InsertEvent(context.Background(), &event)
		if err != nil {
			e.Logger.Error("Failed to insert parent event: ", err.Error())
			http.Error(rw, `{ "msg": "failed to insert parent event" }`, http.StatusInternalServerError)
			return
		}

		// Create recurring instances
		instances := generateEventInstances(&event, req.Recurrence)
		for _, instance := range instances {
			instance.ParentEventID = parentID
			_, err := e.InsertEvent(context.Background(), instance)
			if err != nil {
				e.Logger.Error("Failed to insert child event: ", err.Error())
				continue
			}
		}

		rw.WriteHeader(http.StatusCreated)
		rw.Write([]byte(fmt.Sprintf(`{"id": "%s"}`, parentID.Hex())))
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
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(rw, `{ "msg": "no event id found in request." }`, http.StatusInternalServerError)
			return
		}

		updateAll := r.URL.Query().Get("updateAll") == "true"
		var req models.EventDao
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(rw, `{ "msg": "failed to decode request" }`, http.StatusBadRequest)
			return
		}

		oid, _ := primitive.ObjectIDFromHex(id)
		currentEvent, err := e.FindEvent(context.Background(), bson.M{"_id": oid})
		if err != nil {
			e.Logger.Error("failed to find event", err.Error())
			http.Error(rw, `{ "msg": "failed to find event" }`, http.StatusInternalServerError)
			return
		}

		changes := buildUpdateChanges(&req)
		currentTime := time.Now().Unix()

		if updateAll && currentEvent.IsRecurring != nil && *currentEvent.IsRecurring {
			// Update all future instances
			filter := buildRecurringUpdateFilter(oid, currentEvent, currentTime)
			err = e.UpdateEvents(context.Background(), filter, changes)
		} else {
			// Update single instance
			filter := bson.M{"_id": oid}
			err = e.UpdateEvent(context.Background(), filter, changes)
		}

		if err != nil {
			e.Logger.Error("failed to update event(s)", err.Error())
			http.Error(rw, `{ "msg": "failed to update event(s)" }`, http.StatusInternalServerError)
			return
		}

		updatedEvent, _ := e.FindEvent(context.Background(), bson.M{"_id": oid})
		json.NewEncoder(rw).Encode(updatedEvent)
	}
}

/*
Delete Event Data (Delete)

	http handler
	deletes an event from the database
*/
func (e *Service) DeleteAnEvent() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(rw, `{ "msg": "bad event id" }`, http.StatusBadRequest)
			return
		}

		deleteAll := r.URL.Query().Get("deleteAll") == "true"
		oid, _ := primitive.ObjectIDFromHex(id)

		event, err := e.FindEvent(context.Background(), bson.M{"_id": oid})
		if err != nil {
			e.Logger.Error("failed to find event", err.Error())
			http.Error(rw, `{ "msg": "failed to find event" }`, http.StatusInternalServerError)
			return
		}

		if deleteAll && event.IsRecurring != nil && *event.IsRecurring {
			// Delete entire series
			filter := bson.M{
				"$or": []bson.M{
					{"_id": oid},
					{"parent_event_id": oid},
					{"parent_event_id": event.ParentEventID},
				},
			}
			err = e.DeleteEvents(context.Background(), filter)
		} else {
			// Delete single instance
			filter := bson.M{"_id": oid}
			err = e.DeleteEvent(context.Background(), filter)

			// Track deletion in parent if this is part of a series
			if event.ParentEventID != nil {
				parentFilter := bson.M{"_id": event.ParentEventID}
				update := bson.M{
					"$addToSet": bson.M{
						"deleted_instances": oid,
					},
				}
				e.UpdateEvent(context.Background(), parentFilter, update)
			}
		}

		if err != nil {
			e.Logger.Error("failed to delete event(s)", err.Error())
			http.Error(rw, `{ "msg": "failed to delete event(s)" }`, http.StatusInternalServerError)
			return
		}

		// Cleanup notifications
		if deleteAll {
			if event.ParentEventID != nil {
				utils.DeleteNotificationTopic(event.ParentEventID.Hex())
			}
			utils.DeleteNotificationTopic(id)
		} else {
			utils.DeleteNotificationTopic(id)
		}

		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{ "msg": "OK" }`))
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

		// if event is full add the user to the wait-list
		if event.MaxParticipants != nil {
			if *event.MaxParticipants != 0 {
				if len(participants) >= int(*event.MaxParticipants) {

					// participant object
					newID := primitive.NewObjectID()
					part := &models.ParticipantDao{
						ID:        &newID,
						UUID:      &uuid,
						Status:    req.Status,
						CreatedAt: &timestamp,
					}

					change := bson.M{"$push": bson.M{"wait_list": part}}
					err = e.UpdateEvent(context.Background(), filter, change)
					if err != nil {
						e.Logger.Error("Failed to update event", err.Error())
						http.Error(rw, `{ "msg": "failed to update event" }`, http.StatusInternalServerError)
						return
					}

					rw.WriteHeader(http.StatusOK)
					rw.Write([]byte(`{ "msg": "successful" }`))
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
			e.Logger.Error("Failed to update event", err.Error())
			http.Error(rw, `{ "msg": "failed to update event" }`, http.StatusInternalServerError)
			return
		}

		// notify all participants
		// notif := models.Notification{
		// 	Title: *event.Title,
		// 	Body:  "New Participant RSVP'd!",
		// 	Topic: id,
		// }
		// utils.SendNotificationToTopic(&notif)

		// subscribe user to notifications
		// utils.AddTokenToTopic(id, uuid)

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

		// First, fetch the event to check the wait-list
		event, err := e.FindEvent(context.Background(), bson.M{"_id": oid})
		if err != nil {
			e.Logger.Error("failed to fetch event: ", err.Error())
			http.Error(rw, `{ "msg": "failed to fetch event" }`, http.StatusInternalServerError)
			return
		}

		// First operation: Remove participant
		err = e.UpdateEvent(context.Background(),
			bson.M{"_id": oid},
			bson.M{"$pull": bson.M{"participants": bson.M{"uuid": uuid}}},
		)
		if err != nil {
			e.Logger.Error("failed to remove participant: ", err.Error())
			http.Error(rw, `{ "msg": "failed to update event" }`, http.StatusInternalServerError)
			return
		}

		if event.WaitList != nil && len(*event.WaitList) > 0 {
			waitList := *event.WaitList
			firstWaitListed := waitList[0]

			// Second operation: Remove first person from wait-list
			err = e.UpdateEvent(context.Background(),
				bson.M{"_id": oid},
				bson.M{"$pop": bson.M{"wait_list": -1}},
			)
			if err != nil {
				e.Logger.Error("failed to pop from wait-list: ", err.Error())
				http.Error(rw, `{ "msg": "failed to update event" }`, http.StatusInternalServerError)
				return
			}

			// Third operation: Add wait-listed person to participants
			err = e.UpdateEvent(context.Background(),
				bson.M{"_id": oid},
				bson.M{"$push": bson.M{"participants": firstWaitListed}},
			)
			if err != nil {
				e.Logger.Error("failed to add wait-listed participant: ", err.Error())
				http.Error(rw, `{ "msg": "failed to update event" }`, http.StatusInternalServerError)
				return
			}
		}

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

		resp := models.LocationResponse{
			Venues: &venues,
			Events: events,
		}

		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(resp)
	}
}
