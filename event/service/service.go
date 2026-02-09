package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"olympsis-server/aggregations"
	"olympsis-server/server"
	"olympsis-server/utils"
	"time"

	"github.com/gorilla/mux"
	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/v2/bson"
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

		userID := r.Header.Get("userID")
		isRecurring := req.Recurrence != nil
		timestamp := bson.NewDateTimeFromTime(time.Now())

		// Create base event
		event := req.Event
		event.PosterID = &userID
		event.CreatedAt = &timestamp

		// Single event creation & returns early
		if !isRecurring {
			id, err := s.InsertEvent(ctx, &event)
			if err != nil {
				s.Logger.Error("Failed to insert event: ", err.Error())
				http.Error(rw, `{ "msg": "failed to insert event" }`, http.StatusInternalServerError)
				return
			}

			// Create notification topic
			err = s.Notification.CreateTopic(id.Hex(), []string{userID})
			if err != nil {
				s.Logger.Errorf("Failed to create new event topic. Event ID: %s - Error: %s", id, err.Error())
			}

			// Add the host as the participant
			if req.IncludeHost != nil && *req.IncludeHost {
				rsvp := models.RSVPYes
				participant := models.ParticipantDao{
					UserID:    &userID,
					Status:    &rsvp,
					EventID:   id,
					CreatedAt: &timestamp,
				}
				_, err := s.InsertParticipant(ctx, &participant)
				if err != nil {
					s.Logger.Error("Failed to add host as participant. Error: ", err.Error())
				}
			}

			// Create log
			log := models.EventAuditLog{
				EventID:   *id,
				UserID:    userID,
				Action:    "create",
				Timestamp: timestamp,
			}
			err = s.createEventLog(&log)
			if err != nil {
				s.Logger.Errorf("Failed to create event log. Error: %s", err.Error())
			}

			rw.WriteHeader(http.StatusCreated)
			rw.Header().Set("Content-Type", "application/json")
			rw.Write(fmt.Appendf(nil, `{ "id": "%s" }`, id.Hex()))
			return
		}

		// Handle recurring event creation
		pattern := string(req.Recurrence.Pattern)
		recurrenceConfig := models.EventRecurrenceConfig{
			RecurrenceRule: &pattern,
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

		// Create log
		log := models.EventAuditLog{
			EventID:   *parentID,
			UserID:    userID,
			Action:    "create_many",
			Timestamp: timestamp,
		}
		err = s.createEventLog(&log)
		if err != nil {
			s.Logger.Errorf("Failed to create event log. Error: %s", err.Error())
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

		oid, _ := bson.ObjectIDFromHex(id)

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
		userID := r.Header.Get("userID")
		var user *models.UserData
		var clubs, orgs []bson.ObjectID
		var sportsList []string

		// Initialize empty slices to avoid nil references
		venues := []bson.ObjectID{}

		// If location is provided, get nearby venues
		if queryParams.Location != nil {
			_, venueIDs, err := s.FindNearbyVenues(r.Context(), *queryParams.Location, queryParams.Radius)
			if err != nil {
				s.Logger.Error("Failed to find venues: ", err.Error())
				// Continue with empty venues list instead of failing
				venues = []bson.ObjectID{}
			} else {
				venues = venueIDs
			}
		} else if len(queryParams.VenueIDs) > 0 {
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
			if len(queryParams.Sports) > 0 {
				sportsList = queryParams.Sports
			}

			// Use empty slices for clubs and orgs to avoid nil references
			clubs = []bson.ObjectID{}
			orgs = []bson.ObjectID{}
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
			TotalEvents: int32(len(*events)),
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

		oid, err := bson.ObjectIDFromHex(id)
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

		oid, _ := bson.ObjectIDFromHex(id)
		currentEvent, err := e.FindEvent(context.Background(), bson.M{"_id": oid})
		if err != nil {
			e.Logger.Error("failed to find event", err.Error())
			http.Error(rw, `{ "msg": "failed to find event" }`, http.StatusInternalServerError)
			return
		}

		changes := buildUpdateChanges(&req)
		currentTime := bson.NewDateTimeFromTime(time.Now())

		if updateAll && currentEvent.RecurrenceConfig != nil {
			// Update all future instances
			filter := buildRecurringUpdateFilter(oid, currentEvent, currentTime)
			err = e.UpdateEvents(context.Background(), filter, changes)
		} else {
			// Update single instance
			filter := bson.M{"_id": oid}
			err = e.UpdateEvent(context.Background(), filter, changes)
		}

		// Extract audit-friendly change data from the update
		fieldsChanged, oldValues, newValues := extractAuditChanges(changes, currentEvent)

		// Create log
		log := models.EventAuditLog{
			EventID:       oid,
			UserID:        r.Header.Get("userID"),
			Action:        "update",
			FieldsChanged: fieldsChanged,
			OldValues:     oldValues,
			NewValues:     newValues,
			Timestamp:     bson.NewDateTimeFromTime(time.Now()),
		}
		err = e.createEventLog(&log)
		if err != nil {
			e.Logger.Errorf("Failed to create event log. Error: %s", err.Error())
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
		oid, _ := bson.ObjectIDFromHex(id)

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

			// Execute archival
			update := bson.M{
				"$set": bson.M{
					"archived_at": bson.NewDateTimeFromTime(time.Now()),
				},
			}
			result, err := e.Database.EventsCollection.UpdateMany(ctx, filter, update)
			if err != nil {
				e.Logger.Errorf("Failed to delete events. Error: %s", err.Error())
				http.Error(rw, `{ "msg": "failed to delete events" }`, http.StatusInternalServerError)
				return
			}

			// Log deletion result
			e.Logger.Infof("Actually deleted %d documents", result.ModifiedCount)

			if result.ModifiedCount == 0 {
				e.Logger.Warnf("No documents were deleted with filter: %s", filter)
			}

		} else {
			// Delete single instance
			filter := bson.M{"_id": oid}
			result, err := e.Database.EventsCollection.UpdateOne(ctx, filter, bson.M{
				"$set": bson.M{
					"archived_at": bson.NewDateTimeFromTime(time.Now()),
				},
			})
			if err != nil {
				e.Logger.Error("failed to delete event", err.Error())
				http.Error(rw, `{ "msg": "failed to delete event" }`, http.StatusInternalServerError)
				return
			}

			e.Logger.Infof("Deleted %d document", result.ModifiedCount)

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
		err = e.Notification.RemoveTopic(id)
		if err != nil {
			e.Logger.Errorf("Failed to delete notification topic. Error: %s", err.Error())
		}

		// Create log
		log := models.EventAuditLog{
			EventID:   oid,
			UserID:    r.Header.Get("userID"),
			Action:    "delete",
			Timestamp: bson.NewDateTimeFromTime(time.Now()),
		}
		err = e.createEventLog(&log)
		if err != nil {
			e.Logger.Errorf("Failed to create event log. Error: %s", err.Error())
		}

		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{ "msg": "OK" }`))
	}
}

func (s *Service) Cancel() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		oid, err := utils.ValidateObjectID(id)
		if err != nil {
			s.Logger.Errorf("Failed to validate event ID. Error: %s", err.Error())
			http.Error(w, `{"msg": "bad id"}`, http.StatusBadRequest)
		}

		userID := r.Header.Get("userID")

		filter := bson.M{
			"_id": oid,
		}
		timestamp := bson.NewDateTimeFromTime(time.Now())
		update := bson.M{
			"$set": bson.M{
				"canceled_at": timestamp,
			},
		}
		if err = s.UpdateEvent(r.Context(), filter, update); err != nil {
			s.Logger.Errorf("Failed to update event. Event ID: %s - Error: %s", id, err.Error())
			http.Error(w, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			return
		}

		// Notify participants
		if err = s.Notification.CancelEvent(&oid, userID); err != nil {
			s.Logger.Errorf("Failed to notify event participants. Event ID: %s - Error: %s", id, err.Error())
		}

		// Disable notification topic
		if err = s.Notification.DisableTopic(id); err != nil {
			s.Logger.Errorf("Failed to disable notification topic. Topic ID: %s - Error: %s", id, err.Error())
		}

		// Create log
		log := models.EventAuditLog{
			EventID:   oid,
			UserID:    userID,
			Action:    "cancel",
			Timestamp: timestamp,
		}
		err = s.createEventLog(&log)
		if err != nil {
			s.Logger.Errorf("Failed to create event log. Error: %s", err.Error())
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
		userID := r.Header.Get("userID")
		var userData *models.UserData
		var clubs, orgs []bson.ObjectID

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
