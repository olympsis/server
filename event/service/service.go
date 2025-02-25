package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"olympsis-server/aggregations"
	"olympsis-server/database"
	"olympsis-server/server"
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
	Database     *database.Database           // database for read/write operations
	Logger       *logrus.Logger               // logger for logging errors
	Router       *mux.Router                  // router for handling incoming requests
	Notification *utils.NotificationInterface // notification service for sending notifications
}

/*
Create new event service struct
*/
func NewEventService(i *server.ServerInterface) *Service {
	return &Service{
		Logger:       i.Logger,
		Router:       i.Router,
		Database:     i.Database,
		Notification: i.Notification,
	}
}

/*
Create Event Data (POST)

	http handler
	create an event and add it into the database
*/
// CreateEvent handles both single and recurring event creation
func (e *Service) CreateEvent() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// Create a context with 25s timeout (leaving 5s buffer from the 30s server timeout)
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()

		uuid := r.Header.Get("UUID")

		// Decode request with a timeout
		var req models.NewEventDao
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			e.Logger.Error("Failed to decode request: " + err.Error())
			http.Error(rw, `{ "msg": "failed to decode event" }`, http.StatusBadRequest)
			return
		}

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
			// Handle single event creation with timeout context
			id, err := e.InsertEvent(ctx, &event)
			if err != nil {
				if ctx.Err() == context.DeadlineExceeded {
					e.Logger.Error("Timeout inserting event")
					http.Error(rw, `{ "msg": "operation timed out" }`, http.StatusGatewayTimeout)
					return
				}
				e.Logger.Error("Failed to insert event: ", err.Error())
				http.Error(rw, `{ "msg": "failed to insert event" }`, http.StatusInternalServerError)
				return
			}

			eventID := id.Hex()
			topic := models.NotificationTopicDao{
				Name:  &eventID,
				Users: &[]string{uuid},
			}
			err = e.Notification.CreateTopic(r.Header.Get("Authorization"), topic)
			if err != nil {
				e.Logger.Errorf("Failed to create new event topic. Error: %s", err.Error())
			}

			// Notify group members
			note := models.PushNotification{
				Title:    "New Event Created!",
				Body:     *event.Title,
				Type:     "push",
				Category: "events",
				Data: map[string]interface{}{
					"event_id": eventID,
				},
			}
			if event.Organizers != nil {
				notifyOrganizers(*event.Organizers, &note, r.Header.Get("Authorization"), e.Notification)
			}

			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusCreated)
			rw.Write([]byte(fmt.Sprintf(`{ "id": "%s" }`, id.Hex())))
			return
		}

		// Handle recurring event creation
		event.RecurrenceRule = &req.Recurrence.Pattern
		event.RecurrenceEnd = &req.Recurrence.EndTime

		// Insert parent event with timeout context
		parentID, err := e.InsertEvent(ctx, &event)
		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				e.Logger.Error("Timeout inserting parent event")
				http.Error(rw, `{ "msg": "operation timed out" }`, http.StatusGatewayTimeout)
				return
			}
			e.Logger.Error("Failed to insert parent event: ", err.Error())
			http.Error(rw, `{ "msg": "failed to insert parent event" }`, http.StatusInternalServerError)
			return
		}

		// Create recurring instances with batch processing
		instances := generateEventInstancesBatched(parentID, &event, req.Recurrence)

		// Process instances in batches of 100
		batchSize := 100
		for i := 0; i < len(instances); i += batchSize {
			end := i + batchSize
			if end > len(instances) {
				end = len(instances)
			}

			batch := instances[i:end]
			documents := make([]interface{}, len(batch))
			for j, instance := range batch {
				documents[j] = instance
			}

			// Insert batch with timeout context
			_, err = e.Database.EventCol.InsertMany(ctx, documents)
			if err != nil {
				if ctx.Err() == context.DeadlineExceeded {
					e.Logger.Error("Timeout inserting recurring events batch")
					http.Error(rw, `{ "msg": "operation timed out" }`, http.StatusGatewayTimeout)
					return
				}
				e.Logger.Error("Failed to insert recurring events: ", err.Error())
				http.Error(rw, `{ "msg": "failed to insert recurring events" }`, http.StatusInternalServerError)
				return
			}
		}

		// Notify group members of recurring event
		note := models.PushNotification{
			Title:    "New Events Created!",
			Body:     *event.Title,
			Type:     "push",
			Category: "events",
			Data: map[string]interface{}{
				"event_id": parentID,
			},
		}
		if event.Organizers != nil {
			notifyOrganizers(*event.Organizers, &note, r.Header.Get("Authorization"), e.Notification)
		}

		rw.Header().Set("Content-Type", "application/json")
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
			cursor, err := e.Database.VenueCol.Find(ctx, filter)
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
			rw.WriteHeader(http.StatusNoContent)
			resp := models.EventsResponse{
				TotalEvents: 0,
				Events:      []models.Event{},
			}
			json.NewEncoder(rw).Encode(resp)
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
		ctx := r.Context()
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(rw, `{ "msg": "bad event id" }`, http.StatusBadRequest)
			return
		}

		deleteAll := r.URL.Query().Get("deleteAll") == "true"
		oid, _ := primitive.ObjectIDFromHex(id)

		// First, log what we're trying to delete
		e.Logger.Infof("Attempting to delete event %s, deleteAll=%v", id, deleteAll)

		// Find the event we're trying to delete
		event, err := e.FindEvent(ctx, bson.M{"_id": oid})
		if err != nil {
			e.Logger.Error("failed to find event", err.Error())
			http.Error(rw, `{ "msg": "failed to find event" }`, http.StatusInternalServerError)
			return
		}

		// Log event details
		e.Logger.Infof("Found event: ID=%s, IsRecurring=%v, ParentEventID=%v",
			id,
			event.IsRecurring != nil && *event.IsRecurring,
			event.ParentEventID)

		if deleteAll && event.IsRecurring != nil && *event.IsRecurring {
			var filter bson.M
			if event.ParentEventID != nil {
				// This is a child event, delete parent and all siblings
				parentID := event.ParentEventID
				e.Logger.Infof("Deleting child event and siblings. ParentID=%s", parentID.Hex())

				filter = bson.M{
					"$or": []bson.M{
						{"_id": parentID},
						{"parent_event_id": parentID},
					},
				}

				// Log how many events we expect to delete
				count, _ := e.Database.EventCol.CountDocuments(ctx, filter)
				e.Logger.Infof("Expected to delete %d events for parent %s", count, parentID.Hex())
			} else {
				// This is a parent event, delete it and all children
				e.Logger.Infof("Deleting parent event and all children. ParentID=%s", id)

				filter = bson.M{
					"$or": []bson.M{
						{"_id": oid},
						{"parent_event_id": oid},
					},
				}

				// Log how many events we expect to delete
				count, _ := e.Database.EventCol.CountDocuments(ctx, filter)
				e.Logger.Infof("Expected to delete %d events with parent %s", count, id)
			}

			// Execute deletion
			result, err := e.Database.EventCol.DeleteMany(ctx, filter)
			if err != nil {
				e.Logger.Error("failed to delete events", err.Error())
				http.Error(rw, `{ "msg": "failed to delete events" }`, http.StatusInternalServerError)
				return
			}

			// Log deletion result
			e.Logger.Infof("Actually deleted %d documents", result.DeletedCount)

			if result.DeletedCount == 0 {
				e.Logger.Warn("No documents were deleted with filter:", filter)
			}

		} else {
			// Delete single instance
			e.Logger.Info("Deleting single event instance")

			filter := bson.M{"_id": oid}
			result, err := e.Database.EventCol.DeleteOne(ctx, filter)
			if err != nil {
				e.Logger.Error("failed to delete event", err.Error())
				http.Error(rw, `{ "msg": "failed to delete event" }`, http.StatusInternalServerError)
				return
			}

			e.Logger.Infof("Deleted %d document", result.DeletedCount)

			// Track deletion in parent if this is part of a series
			if event.ParentEventID != nil {
				e.Logger.Info("Updating parent event with deleted instance")

				parentFilter := bson.M{"_id": event.ParentEventID}
				update := bson.M{
					"$addToSet": bson.M{
						"deleted_instances": oid,
					},
				}

				updateResult, err := e.Database.EventCol.UpdateOne(ctx, parentFilter, update)
				if err != nil {
					e.Logger.Error("failed to update parent event", err.Error())
				} else {
					e.Logger.Infof("Updated %d parent document", updateResult.ModifiedCount)
				}
			}
		}

		// Cleanup notifications
		// TODO - CLEAN UP NOTIF OF REPEATED CHILD EVENTS
		e.Logger.Info("Deleting single event notification topic:", id)
		e.Notification.DeleteTopic(r.Header.Get("Authorization"), id)

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

		// Grab event id from path
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(rw, `{ "msg": "bad event id" }`, http.StatusBadRequest)
			return
		}

		// Decode request
		var req models.ParticipantDao
		json.NewDecoder(r.Body).Decode(&req)
		oid, _ := primitive.ObjectIDFromHex(id)
		timestamp := time.Now().Unix()

		// Find event data in database
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

		// Check if participant already exists
		participants := *event.Participants
		for i := range participants {
			if *participants[i].UUID == uuid {
				rw.WriteHeader(http.StatusOK)
				rw.Write([]byte(fmt.Sprintf(`{ "id": "%s" }`, participants[i].ID)))
				return
			}
		}

		// If event is full add the user to the wait-list
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

					// Add participant to notifications
					e.Notification.ModifyTopic(r.Header.Get("Authorization"), id, models.NotificationTopicUpdateRequest{
						Action: "subscribe",
						Users:  []string{uuid},
					})

					rw.WriteHeader(http.StatusOK)
					rw.Write([]byte(`{ "msg": "OK" }`))
					return
				}
			}
		}

		// New participant object
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

		// Add participant to notifications
		e.Notification.ModifyTopic(r.Header.Get("Authorization"), id, models.NotificationTopicUpdateRequest{
			Action: "subscribe",
			Users:  []string{uuid},
		})

		// Notify event participants
		note := models.PushNotification{
			Title:    *event.Title,
			Body:     fmt.Sprintf("New Participant RSVP'ed %s", req.Status),
			Type:     "push",
			Category: "events",
			Data: map[string]interface{}{
				"event_id": oid.Hex(),
			},
		}

		// Notify event participants
		topicName := oid.Hex()
		e.Notification.SendNotification(r.Header.Get("Authorization"), models.NotificationPushRequest{
			Topic:        &topicName,
			Notification: note,
		})

		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{ "msg": "OK" }`))
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

		// grab ids from path
		vars := mux.Vars(r)
		eventID := vars["id"]
		if len(eventID) < 24 {
			http.Error(rw, `{ "msg": "bad event id" }`, http.StatusBadRequest)
			return
		}
		oid, _ := primitive.ObjectIDFromHex(eventID)

		// First, fetch the event to check the wait-list
		event, err := e.FindEvent(context.Background(), bson.M{"_id": oid})
		if err != nil {
			e.Logger.Error("failed to fetch event: ", err.Error())
			http.Error(rw, `{ "msg": "failed to fetch event" }`, http.StatusInternalServerError)
			return
		}

		var pullQuery bson.M
		var participant models.ParticipantDao

		if participantID, exists := vars["participantID"]; exists && participantID != "" {
			// Verify that the requestor is the event organizer
			// TODO:  FIGURE OUT CLUB AUTH
			// if *event.Poster != uuid {
			// 	http.Error(rw, `{ "msg": "unauthorized - only event organizer can remove other participants" }`, http.StatusUnauthorized)
			// 	return
			// }

			// Remove by participant _id
			participantOID, _ := primitive.ObjectIDFromHex(participantID)
			pullQuery = bson.M{"$pull": bson.M{"participants": bson.M{"_id": participantOID}}}

			// Assign Participant
			for _, v := range *event.Participants {
				if *v.ID == participantOID {
					participant = v
				}
			}
		} else {
			// Remove by UUID (user removing themselves)
			pullQuery = bson.M{"$pull": bson.M{"participants": bson.M{"uuid": uuid}}}

			// Assign Participant
			for _, v := range *event.Participants {
				if *v.UUID == uuid {
					participant = v
				}
			}
		}

		// First operation: Remove participant
		err = e.UpdateEvent(context.Background(),
			bson.M{"_id": oid},
			pullQuery,
		)
		if err != nil {
			e.Logger.Error("failed to remove participant: ", err.Error())
			http.Error(rw, `{ "msg": "failed to update event" }`, http.StatusInternalServerError)
			return
		}

		// Handle wait-list if someone was successfully removed
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

			// Notify the promoted participant
			if event.Title != nil {
				notif := models.PushNotification{
					Title: *event.Title,
					Body:  "You've been promoted from the wait-list!",
				}
				e.Notification.SendNotification(r.Header.Get("Authorization"), models.NotificationPushRequest{
					Users:        &[]string{*firstWaitListed.UUID},
					Notification: notif,
				})
			}
		}

		// unsubscribe removed user from notifications
		e.Notification.ModifyTopic(r.Header.Get("Authorization"), eventID, models.NotificationTopicUpdateRequest{
			Action: "unsubscribe",
			Users:  []string{*participant.UUID},
		})

		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{ "msg": "participant successfully removed" }`))
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
		var req models.PushNotification
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			e.Logger.Error("failed to decode notification", err.Error())
			http.Error(rw, `{ "msg": "failed to decode notification" }`, http.StatusInternalServerError)
			return
		}

		e.Notification.SendNotification(r.Header.Get("Authorization"), models.NotificationPushRequest{
			Topic:        &id,
			Notification: req,
		})
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{ "msg": "OK" }`))
	}
}

/*
Notify Club members (POST)

	http handler
	notifies all club members of an event
*/
func (e *Service) NotifyOrganizers() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// grab id from path
		vars := mux.Vars(r)
		id := vars["id"]
		if len(vars["id"]) < 24 {
			http.Error(rw, `{ "msg": "bad event id" }`, http.StatusBadRequest)
			return
		}
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			http.Error(rw, `{ "msg": "failed to encode id to OID" }`, http.StatusBadRequest)
			return
		}

		// Decode request
		var req models.PushNotification
		err = json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(rw, `{ "msg": "failed to decode request" }`, http.StatusBadRequest)
			return
		}

		// Grab event dao object
		event, err := e.FindEvent(context.Background(), bson.M{"_id": oid})
		if err != nil {
			if err == mongo.ErrNoDocuments {
				http.Error(rw, `{ "msg": "event not found" }`, http.StatusNotFound)
				return
			}
			e.Logger.Error("failed to find event", err.Error())
			http.Error(rw, `{ "msg": "failed to find event" }`, http.StatusInternalServerError)
			return
		}

		// Notify organizers
		if event.Organizers != nil {
			notifyOrganizers(*event.Organizers, &req, r.Header.Get("Authorization"), e.Notification)
		}

		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{ "msg": "OK" }`))
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
			cursor, err := e.Database.VenueCol.Find(ctx, filter)
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
