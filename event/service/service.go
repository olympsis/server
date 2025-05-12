package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"olympsis-server/aggregations"
	"olympsis-server/server"
	"time"

	"github.com/gorilla/mux"
	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

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

func (s *Service) CreateEvent() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// Create a context with 25s timeout (leaving 5s buffer from the 30s server timeout)
		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()

		// Decode request with a timeout
		var req models.NewEventDao
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.Logger.Error("Failed to decode request: " + err.Error())
			http.Error(rw, `{ "msg": "failed to decode event from request body" }`, http.StatusBadRequest)
			return
		}

		uuid := r.Header.Get("UUID")
		isRecurring := req.Recurrence != nil
		authToken := r.Header.Get("Authorization")
		timestamp := primitive.NewDateTimeFromTime(time.Now())

		// Create base event
		event := req.Event
		event.PosterID = &uuid
		event.CreatedAt = &timestamp

		// Single event creation & returns early
		if !isRecurring {
			id, err := s.InsertEvent(ctx, &event)
			if err != nil {
				s.Logger.Error("Failed to insert event: ", err.Error())
				http.Error(rw, `{ "msg": "failed to insert event" }`, http.StatusInternalServerError)
				return
			}

			// Add the host as the participant
			if req.IncludeHost != nil && *req.IncludeHost == true {
				rsvp := models.RSVPYes
				participant := models.ParticipantDao{
					UserID:    &uuid,
					Status:    &rsvp,
					EventID:   id,
					CreatedAt: &timestamp,
				}
				_, err := s.InsertParticipant(ctx, &participant)
				if err != nil {
					s.Logger.Error("Failed to add host as participant. Error: ", err.Error())
				}

				// Add participant to notifications
				s.Notification.ModifyTopic(authToken, id.Hex(), models.NotificationTopicUpdateRequest{
					Action: "subscribe",
					Users:  []string{uuid},
				})
			}

			// Create event topic
			eventID := id.Hex()
			err = s.Notification.CreateTopic(
				authToken,
				models.NotificationTopicDao{
					Name:  &eventID,
					Users: &[]string{uuid},
				},
			)
			if err != nil {
				s.Logger.Errorf("Failed to create new event topic. Error: %s", err.Error())
			}

			// Notify group members
			note := GenerateNewEventNotification(eventID, *event.Title)
			if event.Organizers != nil {
				notifyOrganizers(*event.Organizers, &note, authToken, s.Notification)
			}

			rw.WriteHeader(http.StatusCreated)
			rw.Header().Set("Content-Type", "application/json")
			rw.Write(fmt.Appendf(nil, `{ "id": "%s" }`, id.Hex()))
			return
		}

		// Handle recurring event creation
		recurrenceConfig := models.EventRecurrenceConfig{
			RecurrenceRule: &req.Recurrence.Pattern,
			RecurrenceEnd:  &req.Recurrence.EndTime,
		}
		event.RecurrenceConfig = &recurrenceConfig
		parentID, err := s.InsertEvent(ctx, &event)
		if err != nil {
			s.Logger.Error("Failed to insert parent event: ", err.Error())
			http.Error(rw, `{ "msg": "failed to insert parent event" }`, http.StatusInternalServerError)
			return
		}

		// Create recurring instances with batch processing
		instances := GenerateEventInstancesBatched(*parentID, &event, req.Recurrence)

		// Process instances in batches of 100
		batchSize := 100
		for i := 0; i < len(instances); i += batchSize {
			end := min(i+batchSize, len(instances))
			batch := instances[i:end]
			documents := make([]interface{}, len(batch))
			for j, instance := range batch {
				documents[j] = instance
			}

			// Insert batch with timeout context
			_, err = s.Database.EventsCollection.InsertMany(ctx, documents)
			if err != nil {
				s.Logger.Error("Failed to insert recurring events: ", err.Error())
				http.Error(rw, `{ "msg": "failed to insert recurring events" }`, http.StatusInternalServerError)
				return
			}
		}

		rw.WriteHeader(http.StatusCreated)
		rw.Header().Set("Content-Type", "application/json")
		rw.Write(fmt.Appendf(nil, `{"id": "%s"}`, parentID.Hex()))
	}
}

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
			e.Logger.Error("Failed to find event. Error: ", err.Error())
			http.Error(rw, `{ "msg": "failed to find event" }`, http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusOK)
		rw.Header().Set("Content-Type", "application/json")
		json.NewEncoder(rw).Encode(event)
	}
}

func (s *Service) GetEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get query parameters with validation
		queryParams, err := parseEventsQueryParams(r)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"msg":"%s"}`, err.Error()), http.StatusBadRequest)
			return
		}

		// Parse user info (if authenticated)
		userID := r.Header.Get("UUID")
		var user *models.UserData
		var clubs, orgs []primitive.ObjectID
		var sportsList []string

		// Initialize empty slices to avoid nil references
		venues := []primitive.ObjectID{}

		// If location is provided, get nearby venues
		if queryParams.Location != nil {
			_, venueIDs, err := s.FindNearbyVenues(r.Context(), *queryParams.Location, queryParams.Radius)
			if err != nil {
				s.Logger.Error("Failed to find venues: ", err.Error())
				// Continue with empty venues list instead of failing
				venues = []primitive.ObjectID{}
			} else {
				venues = venueIDs
			}
		} else if queryParams.VenueIDs != nil && len(queryParams.VenueIDs) > 0 {
			// Use directly provided venue IDs
			venues = queryParams.VenueIDs
		}

		// If user is authenticated, fetch their data
		if userID != "" {
			user, err = aggregations.AggregateUser(&userID, s.Database)
			if err != nil {
				s.Logger.Error("Failed to get user data: ", err.Error())
				// Fall back to unauthenticated mode instead of failing entirely
			}
		}

		// Set up the user-specific filter values
		if user != nil {
			// User is authenticated and data was successfully retrieved
			if queryParams.Sports == nil {
				sportsList = user.Sports
			}

			if user.Clubs != nil {
				clubs = *user.Clubs
			}

			if user.Organizations != nil {
				orgs = *user.Organizations
			}
		} else {
			// Handle unauthenticated case or failed user data retrieval
			// Convert sports string to array if provided in query
			if queryParams.Sports != nil && len(queryParams.Sports) > 0 {
				sportsList = queryParams.Sports
			}

			// Use empty slices for clubs and orgs to avoid nil references
			clubs = []primitive.ObjectID{}
			orgs = []primitive.ObjectID{}
		}

		// Retrieve events
		events, err := aggregations.AggregateEvents(
			&userID, // Will be empty string for unauthenticated users
			&sportsList,
			queryParams.Location,
			&venues,
			&clubs,
			&orgs,
			queryParams.Radius,
			queryParams.Limit,
			queryParams.Skip,
			s.Database,
		)

		// Handle errors and respond appropriately
		if err != nil {
			s.Logger.Error("Failed to find events: ", err.Error())
			http.Error(w, `{"msg":"failed to find events"}`, http.StatusInternalServerError)
			return
		}

		// Return appropriate response
		if events == nil || len(*events) == 0 {
			w.WriteHeader(http.StatusOK) // Changed from NoContent to OK with empty array
			resp := models.EventsResponse{
				TotalEvents: 0,
				Events:      []models.Event{},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Success response
		resp := models.EventsResponse{
			TotalEvents: int16(len(*events)),
			Events:      *events,
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}
}

func (s *Service) GetGroupPastEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		if len(id) < 24 {
			http.Error(w, `{ "msg": "no/bad group id found in request" }`, http.StatusBadRequest)
			return
		}

		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			http.Error(w, `{ "msg": "failed to encode id" }`, http.StatusInternalServerError)
			return
		}

		events, err := aggregations.AggregateGroupPastEvents(oid, 100, 0, s.Database)
		if err != nil {
			s.Logger.Error("Failed to find events. Error: ", err.Error())
			http.Error(w, `{ "msg": "failed to get past events" }`, http.StatusInternalServerError)
			return
		}

		if events != nil {
			eventsList := *events
			if len(eventsList) > 0 {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(events)
				return
			} else {
				w.WriteHeader(http.StatusNoContent)
				w.Write([]byte(`{ "msg": "no events" }`))
				return
			}
		} else {
			http.Error(w, `{ "msg": "failed to get past events" }`, http.StatusInternalServerError)
		}
	}
}

func (s *Service) GetUserPastEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["uuid"]
		if id == "" {
			http.Error(w, `{ "msg": "no/bad group id found in request" }`, http.StatusBadRequest)
			return
		}

		events, err := aggregations.AggregateUserPastEvents(id, 100, 0, s.Database)
		if err != nil {
			s.Logger.Error("Failed to find events. Error: ", err.Error())
			http.Error(w, `{ "msg": "failed to find events" }`, http.StatusInternalServerError)
			return
		}

		if events != nil {
			eventsList := *events
			if len(eventsList) > 0 {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(events)
				return
			} else {
				w.WriteHeader(http.StatusNoContent)
				w.Write([]byte(`{ "msg": "no events" }`))
				return
			}
		} else {
			http.Error(w, `{ "msg": "failed to get past events" }`, http.StatusInternalServerError)
		}
	}
}

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
		currentTime := primitive.NewDateTimeFromTime(time.Now())

		if updateAll && currentEvent.RecurrenceConfig != nil {
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
		if deleteAll && event.RecurrenceConfig != nil {
			var filter bson.M
			if event.RecurrenceConfig.ParentEventID != nil {
				// This is a child event, delete parent and all siblings
				parentID := event.RecurrenceConfig.ParentEventID
				e.Logger.Infof("Deleting child event and siblings. ParentID=%s", parentID.Hex())

				filter = bson.M{
					"$or": []bson.M{
						{"_id": parentID},
						{"parent_event_id": parentID},
					},
				}

				// Log how many events we expect to delete
				count, _ := e.Database.EventsCollection.CountDocuments(ctx, filter)
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
				count, _ := e.Database.EventsCollection.CountDocuments(ctx, filter)
				e.Logger.Infof("Expected to delete %d events with parent %s", count, id)
			}

			// Execute deletion
			result, err := e.Database.EventsCollection.DeleteMany(ctx, filter)
			if err != nil {
				e.Logger.Error("failed to delete events", err.Error())
				http.Error(rw, `{ "msg": "failed to delete events" }`, http.StatusInternalServerError)
				return
			}

			// Delete parent event's participants
			err = e.DeleteParticipants(ctx, bson.M{"event_id": oid})
			if err != nil {
				e.Logger.Error("Failed to remove event's participants. Error: ", err.Error())
			}

			// Log deletion result
			e.Logger.Infof("Actually deleted %d documents", result.DeletedCount)

			if result.DeletedCount == 0 {
				e.Logger.Warn("No documents were deleted with filter:", filter)
			}

		} else {
			// Delete single instance
			filter := bson.M{"_id": oid}
			result, err := e.Database.EventsCollection.DeleteOne(ctx, filter)
			if err != nil {
				e.Logger.Error("failed to delete event", err.Error())
				http.Error(rw, `{ "msg": "failed to delete event" }`, http.StatusInternalServerError)
				return
			}

			e.Logger.Infof("Deleted %d document", result.DeletedCount)

			// Delete event participants
			err = e.DeleteParticipants(ctx, bson.M{"event_id": oid})
			if err != nil {
				e.Logger.Error("Failed to remove event's participants. Error: ", err.Error())
			}

			// Track deletion in parent if this is part of a series
			if event.RecurrenceConfig != nil && event.RecurrenceConfig.ParentEventID != nil {
				e.Logger.Info("Updating parent event with deleted instance")

				parentFilter := bson.M{"_id": event.RecurrenceConfig.ParentEventID}
				update := bson.M{
					"$addToSet": bson.M{
						"deleted_instances": oid,
					},
				}

				updateResult, err := e.Database.EventsCollection.UpdateOne(ctx, parentFilter, update)
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

func (e *Service) AddParticipant() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		uuid := r.Header.Get("UUID")
		token := r.Header.Get("Authorization")

		// Grab event id from path
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(rw, `{ "msg": "bad event id" }`, http.StatusBadRequest)
			return
		}

		// Decode request
		var req models.ParticipantDao
		oid, _ := primitive.ObjectIDFromHex(id)
		timestamp := primitive.NewDateTimeFromTime(time.Now())
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(rw, `{"msg":"failed to decode request"}`, http.StatusBadRequest)
			return
		}

		// Validations
		if req.EventID == nil {
			req.EventID = &oid
		}
		if req.Status == nil {
			defaultRSVP := models.RSVPYes
			req.Status = &defaultRSVP
		}

		// Find event in database
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
		participants, err := e.FindParticipants(context.TODO(), bson.M{"event_id": oid}, nil)
		if err != nil {
			http.Error(rw, `{"msg":"failed to find event's participants"}`, http.StatusInternalServerError)
			return
		}
		for i := range participants {
			if *participants[i].UserID == uuid {
				rw.WriteHeader(http.StatusOK)
				rw.Write(fmt.Appendf(nil, `{ "id": "%s" }`, participants[i].ID))
				return
			}
		}

		// New participant object
		participant := &models.ParticipantDao{
			UserID:    &uuid,
			EventID:   req.EventID,
			Status:    req.Status,
			CreatedAt: &timestamp,
		}

		// If event is full add the user to the wait-list
		if event.ParticipantsConfig != nil && event.ParticipantsConfig.MaxParticipants != nil {
			if *event.ParticipantsConfig.MaxParticipants != 0 {
				if len(participants) >= int(*event.ParticipantsConfig.MaxParticipants) {

					// Add Participant to the participants database
					waitStatus := models.RSVPWaitlist
					participant.Status = &waitStatus
					pid, err := e.InsertParticipant(context.TODO(), participant)
					if err != nil {
						e.Logger.Error("Failed to add participant to waitlist", err.Error())
						http.Error(rw, `{ "msg": "failed to add participant to waitlist" }`, http.StatusInternalServerError)
						return
					}

					// Add participant to notifications
					e.Notification.ModifyTopic(r.Header.Get("Authorization"), id, models.NotificationTopicUpdateRequest{
						Action: "subscribe",
						Users:  []string{uuid},
					})

					rw.WriteHeader(http.StatusOK)
					rw.Write(fmt.Appendf(nil, `{"id": "%s"}`, pid.Hex()))
					return
				}
			}
		}

		// Insert participant to event participants database
		pid, err := e.InsertParticipant(context.TODO(), participant)
		if err != nil {
			e.Logger.Error("Failed to add participant to event", err.Error())
			http.Error(rw, `{ "msg": "failed to add participant to event" }`, http.StatusInternalServerError)
			return
		}

		// Add participant to notifications
		e.Notification.ModifyTopic(token, id, models.NotificationTopicUpdateRequest{
			Action: "subscribe",
			Users:  []string{uuid},
		})

		// Notify event participants
		status := "yes"
		if *req.Status == 0 {
			status = "maybe"
		}
		topicName := oid.Hex()
		note := generateNewParticipantNotification(oid.Hex(), *event.Title, status)
		e.Notification.SendNotification(token, models.NotificationPushRequest{
			Topic:        &topicName,
			Notification: note,
		})

		rw.WriteHeader(http.StatusOK)
		rw.Write(fmt.Appendf(nil, `{"id": "%s"}`, pid.Hex()))
	}
}

func (e *Service) RemoveParticipant() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()

		uuid := r.Header.Get("UUID")
		token := r.Header.Get("Authorization")

		// grab ids from path
		vars := mux.Vars(r)
		eventID := vars["id"]
		oid, err := primitive.ObjectIDFromHex(eventID)
		if err != nil {
			e.Logger.Error("Failed to encode event ID to Object ID", err.Error())
			http.Error(rw, `{"msg": "invalid event id"}`, http.StatusBadRequest)
			return
		}

		// First, fetch the event
		event, err := e.FindEvent(ctx, bson.M{"_id": oid})
		if err != nil {
			e.Logger.Error("failed to fetch event: ", err.Error())
			http.Error(rw, `{ "msg": "failed to fetch event" }`, http.StatusInternalServerError)
			return
		}

		// If we have a participant ID this means that they are being removed
		if participantID, exists := vars["participantID"]; exists && participantID != "" {
			participantOID, _ := primitive.ObjectIDFromHex(participantID)
			participant, err := e.FindParticipant(ctx, bson.M{"_id": participantOID})
			if err != nil {
				http.Error(rw, `{"msg": "failed to find participant"}`, http.StatusNotFound)
				return
			}

			// Remove the participant from the database
			err = e.DeleteParticipant(ctx, bson.M{"_id": participantOID, "event_id": oid})
			if err != nil {
				e.Logger.Errorf("Failed to delete event participant. Error: %s", err.Error())
				http.Error(rw, `{"msg": "failed to remove participant"}`, http.StatusInternalServerError)
				return
			}

			// Notification model
			notif := models.PushNotification{
				Title:    *event.Title,
				Body:     "You've been kicked from the participants list!",
				Type:     "push",
				Category: "events",
				Data: map[string]any{
					"type": "event_update",
					"id":   eventID,
				},
			}

			// Notify the user that they have been removed
			e.Notification.SendNotification(token, models.NotificationPushRequest{
				Users:        &[]string{*participant.UserID},
				Notification: notif,
			})

			// Unsubscribe user from topic
			e.Notification.ModifyTopic(token, eventID, models.NotificationTopicUpdateRequest{
				Action: "unsubscribe",
				Users:  []string{*participant.UserID},
			})
		} else { // User removing themselves from the list

			// Remove participant
			err = e.DeleteParticipant(ctx, bson.M{"user_id": uuid, "event_id": oid})
			if err != nil {
				e.Logger.Errorf("Failed to delete event participant. Error: %s", err.Error())
			}

			// Unsubscribe user from topic
			e.Notification.ModifyTopic(token, eventID, models.NotificationTopicUpdateRequest{
				Action: "unsubscribe",
				Users:  []string{uuid},
			})
		}

		// Check waitlist and see if we need to promote another participant
		opts := options.FindOptions{Sort: bson.M{"created_at": 1}}
		waitlist, err := e.FindParticipants(ctx, bson.M{"event_id": oid, "status": models.RSVPWaitlist}, &opts)
		if err != nil {
			e.Logger.Error("Failed to check waitlist. Error: ", err.Error())
		}
		if len(waitlist) > 0 {
			participant := waitlist[0]
			err := e.UpdateParticipant(ctx, bson.M{"_id": participant.ID}, bson.M{"status": models.RSVPYes})
			if err != nil {
				e.Logger.Error("Failed to promote participant from waitlist. Error: ", err.Error())
			}

			// Notification model
			notif := models.PushNotification{
				Title:    *event.Title,
				Body:     "You've been promoted from the waitlist!",
				Type:     "push",
				Category: "events",
				Data: map[string]any{
					"type": "event_update",
					"id":   eventID,
				},
			}

			// Notify the user that they have been promoted
			e.Notification.SendNotification(token, models.NotificationPushRequest{
				Users:        &[]string{*participant.UserID},
				Notification: notif,
			})
		}

		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{"msg": "OK"}`))
	}
}

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

func (s *Service) AddComment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		uuid := r.Header.Get("UUID")

		// Grab event id from path
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(w, `{"msg": "bad event id"}`, http.StatusBadRequest)
			return
		}
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			s.Logger.Error("Failed to convert id to ObjectID. Error: ", err.Error())
			http.Error(w, `{"msg": "failed to encode id"}`, http.StatusBadRequest)
			return
		}

		// Decode request
		var req models.EventCommentDao
		err = json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			s.Logger.Error("Failed to decode request. Error: ", err.Error())
			http.Error(w, `{"msg":"failed to decode request"}`, http.StatusBadRequest)
			return
		}

		// Add comment to the database
		timestamp := primitive.NewDateTimeFromTime(time.Now())
		req.UserID = &uuid
		req.EventID = oid
		req.CreatedAt = &timestamp
		cid, err := s.InsertComment(ctx, &req)
		if err != nil {
			s.Logger.Error("Failed to insert comment into the database. Error: ", err.Error())
			http.Error(w, `{"msg": "failed to create comment"}`, http.StatusInternalServerError)
		}

		// Notify users of a new comment
		event, err := s.FindEvent(ctx, bson.M{"_id": oid})
		notif := models.PushNotification{
			Title:    *event.Title,
			Body:     "A new comment was posted!",
			Type:     "push",
			Category: "events",
			Data: map[string]any{
				"type": "event_update",
				"id":   id,
			},
		}
		s.Notification.SendNotification(r.Header.Get("Authorization"), models.NotificationPushRequest{
			Notification: notif,
			Users:        &[]string{uuid},
		})

		w.WriteHeader(http.StatusCreated)
		w.Write(fmt.Appendf(nil, `{"id": "%s"}`, cid.Hex()))
	}
}

func (s *Service) RemoveComment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		// Grab comment id from path
		vars := mux.Vars(r)
		id := vars["commentID"]
		if len(id) < 24 {
			http.Error(w, `{"msg": "bad comment id"}`, http.StatusBadRequest)
			return
		}
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			s.Logger.Error("Failed to convert id to ObjectID. Error: ", err.Error())
			http.Error(w, `{"msg": "failed to encode id"}`, http.StatusBadRequest)
			return
		}

		// Delete comment from db
		err = s.DeleteComment(ctx, bson.M{"_id": oid})
		if err != nil {
			s.Logger.Error("Failed to delete comment. Error: ", err.Error())
			http.Error(w, `{"msg": "failed to remove comment"}`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"msg": "OK"}`))
	}
}

func (s *Service) Location() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse and validate query parameters
		params, err := parseLocationQueryParams(r)
		if err != nil {
			s.Logger.Info("Invalid query parameters: ", err.Error())
			http.Error(w, fmt.Sprintf(`{"msg":"%s"}`, err.Error()), http.StatusBadRequest)
			return
		}

		// Create GeoJSON location object
		location := models.GeoJSON{
			Type:        "Point",
			Coordinates: []float64{params.Longitude, params.Latitude},
		}

		// Get nearby venues based on location, sports, and radius
		venues, venueIDs, err := s.FindNearbyVenues(r.Context(), location, params.Radius)
		if err != nil {
			s.Logger.Error("Failed to find venues: ", err.Error())
			http.Error(w, `{"msg":"Failed to get venue data"}`, http.StatusInternalServerError)
			return
		}

		// Get user data if authenticated
		userID := r.Header.Get("UUID")
		var userData *models.UserData
		var clubs, orgs []primitive.ObjectID

		if userID != "" {
			userData, err = aggregations.AggregateUser(&userID, s.Database)
			if err != nil {
				s.Logger.Warning("Failed to get user data, proceeding as unauthenticated: ", err.Error())
				// Continue with unauthenticated user rather than failing
				userID = "" // Clear userID to ensure we're using unauthenticated mode
			} else if userData != nil {
				// Extract clubs and organizations from user data
				if userData.Clubs != nil {
					clubs = *userData.Clubs
				}
				if userData.Organizations != nil {
					orgs = *userData.Organizations
				}
			}
		}

		// Convert optional params to pointer types for AggregateEvents
		userIDPtr := new(string)
		*userIDPtr = userID

		sportsList := params.Sports
		sportsPtr := &sportsList

		clubsPtr := &clubs
		orgsPtr := &orgs
		venueIDsPtr := &venueIDs

		// Find events
		events, err := aggregations.AggregateEvents(
			userIDPtr,
			sportsPtr,
			&location,
			venueIDsPtr,
			clubsPtr,
			orgsPtr,
			params.Radius,
			params.Limit,
			params.Skip,
			s.Database,
		)

		if err != nil {
			s.Logger.Error("Failed to find events: ", err.Error())
			http.Error(w, `{"msg":"Failed to find events"}`, http.StatusInternalServerError)
			return
		}

		// Build response
		resp := models.LocationResponse{
			Venues: venues,
			Events: events,
		}

		// Send response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			s.Logger.Error("Failed to encode response: ", err.Error())
		}
	}
}
