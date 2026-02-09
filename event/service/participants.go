package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func (e *Service) AddParticipant() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		uuid := r.Header.Get("userID")

		// Grab event id from path
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(rw, `{ "msg": "bad event id" }`, http.StatusBadRequest)
			return
		}

		// Decode request
		var req models.ParticipantDao
		oid, _ := bson.ObjectIDFromHex(id)
		timestamp := bson.NewDateTimeFromTime(time.Now())
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
			UserID:      &uuid,
			EventID:     req.EventID,
			Status:      req.Status,
			IsAnonymous: req.IsAnonymous,
			CreatedAt:   &timestamp,
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
					err = e.Notification.AddUsersToTopic(id, []string{uuid})
					if err != nil {
						e.Logger.Errorf("Failed to add user to event notifications topic. Event ID: %s - Error: %s", id, err.Error())
					}

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
		if err = e.Notification.AddUsersToTopic(id, []string{uuid}); err != nil {
			e.Logger.Errorf("Failed to add user to notifications topic. EventID: %s - Error: %s", id, err.Error())
		}

		rw.WriteHeader(http.StatusOK)
		rw.Write(fmt.Appendf(nil, `{"id": "%s"}`, pid.Hex()))
	}
}

func (e *Service) RemoveParticipant() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
		defer cancel()

		uuid := r.Header.Get("userID")

		// grab ids from path
		vars := mux.Vars(r)
		eventID := vars["id"]
		oid, err := bson.ObjectIDFromHex(eventID)
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
			participantOID, _ := bson.ObjectIDFromHex(participantID)
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

			// Remove user from the topic
			if err = e.Notification.RemoveUsersFromTopic(eventID, []string{*participant.UserID}); err != nil {
				e.Logger.Errorf("Failed to remove user from notification topic. Event: %s - Error: %s", eventID, err.Error())
			}

			// Notify the user that they have been removed
			if err = e.Notification.ParticipantKick(event, participant); err != nil {
				e.Logger.Errorf("Failed to notify user. Event ID: %s - Error: %s", eventID, err.Error())
			}
		} else { // User removing themselves from the list

			// Remove participant
			err = e.DeleteParticipant(ctx, bson.M{"user_id": uuid, "event_id": oid})
			if err != nil {
				e.Logger.Errorf("Failed to delete event participant. Error: %s", err.Error())
				http.Error(rw, `{"msg": "failed to remove participant"}`, http.StatusInternalServerError)
			}

			// Remove user from the topic
			if err = e.Notification.RemoveUsersFromTopic(eventID, []string{uuid}); err != nil {
				e.Logger.Errorf("Failed to remove user from notification topic. Event: %s - Error: %s", eventID, err.Error())
			}
		}

		// Check waitlist and see if we need to promote another participant
		opts := options.Find().SetSort(bson.M{"created_at": 1})
		waitlist, err := e.FindParticipants(ctx, bson.M{"event_id": oid, "status": models.RSVPWaitlist}, opts)
		if err != nil {
			e.Logger.Error("Failed to check waitlist. Error: ", err.Error())
		}
		if len(waitlist) > 0 {
			participant := waitlist[0]
			err := e.UpdateParticipant(ctx, bson.M{"_id": participant.ID}, bson.M{"status": models.RSVPYes})
			if err != nil {
				e.Logger.Error("Failed to promote participant from waitlist. Error: ", err.Error())
			} else {
				// Notify the user that they have been promoted
				if err = e.Notification.WaitlistPromotion(event, &participant); err != nil {
					e.Logger.Errorf("Failed to notify user. Event: %s - Error: %s", eventID, err.Error())
				}
			}
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

		request := models.NotificationPushRequest{
			Topic:        &id,
			Notification: req,
		}
		if err = e.Notification.AddNoteToCarousel(2, &request); err != nil {
			e.Logger.Errorf("Failed to send notification. Event ID: %s - Error: %s", id, err.Error())
			http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{ "msg": "OK" }`))
	}
}

/*****************
DATABASE FUNCTIONS
******************/

// Insert new participant into database
func (s *Service) InsertParticipant(ctx context.Context, participant *models.ParticipantDao) (bson.ObjectID, error) {
	resp, err := s.Database.EventParticipantsCollection.InsertOne(ctx, participant)
	if err != nil {
		return bson.NilObjectID, err
	}
	id := resp.InsertedID.(bson.ObjectID)
	return id, nil
}

// Insert multiple participants into database
func (s *Service) InsertParticipants(ctx context.Context, participants []interface{}) ([]bson.ObjectID, error) {
	resp, err := s.Database.EventParticipantsCollection.InsertMany(ctx, participants)
	if err != nil {
		return nil, err
	}

	// Convert inserted IDs to ObjectIDs
	ids := make([]bson.ObjectID, len(resp.InsertedIDs))
	for i, id := range resp.InsertedIDs {
		ids[i] = id.(bson.ObjectID)
	}

	return ids, nil
}

// Get participant from database
func (s *Service) FindParticipant(ctx context.Context, filter bson.M) (*models.ParticipantDao, error) {
	var participant models.ParticipantDao
	err := s.Database.EventParticipantsCollection.FindOne(ctx, filter).Decode(&participant)
	if err != nil {
		return nil, err
	}
	return &participant, nil
}

// Get participants from database
func (s *Service) FindParticipants(ctx context.Context, filter bson.M, opts *options.FindOptionsBuilder) ([]models.ParticipantDao, error) {
	var participants []models.ParticipantDao
	cursor, err := s.Database.EventParticipantsCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &participants); err != nil {
		return nil, err
	}
	return participants, nil
}

// Update participant in database
func (s *Service) UpdateParticipant(ctx context.Context, filter bson.M, update bson.M) error {
	_, err := s.Database.EventParticipantsCollection.UpdateOne(ctx, filter, update)
	return err
}

// Update multiple participants in database
func (s *Service) UpdateParticipants(ctx context.Context, filter bson.M, update bson.M) error {
	_, err := s.Database.EventParticipantsCollection.UpdateMany(ctx, filter, update)
	return err
}

// Delete participant from database
func (s *Service) DeleteParticipant(ctx context.Context, filter bson.M) error {
	_, err := s.Database.EventParticipantsCollection.DeleteOne(ctx, filter)
	return err
}

// Delete multiple participants from database
func (s *Service) DeleteParticipants(ctx context.Context, filter bson.M) error {
	_, err := s.Database.EventParticipantsCollection.DeleteMany(ctx, filter)
	return err
}
