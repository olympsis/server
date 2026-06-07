package service

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// AddParticipantViaBot records an RSVP captured by the Telegram/Discord bot father on
// behalf of a resolved Olympsis user. It is internal-only (called by the bots service)
// and differs from AddParticipant in that the user id comes from the request body rather
// than an authenticated header, and a decline removes the participant.
//
// POST /v1/events/{id}/participants/bot
func (e *Service) AddParticipantViaBot() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		id := mux.Vars(r)["id"]
		oid, err := bson.ObjectIDFromHex(id)
		if err != nil {
			http.Error(rw, `{"msg": "bad event id"}`, http.StatusBadRequest)
			return
		}

		var req models.BotParticipantRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(rw, `{"msg": "failed to decode request"}`, http.StatusBadRequest)
			return
		}
		if req.UserID == "" {
			http.Error(rw, `{"msg": "missing user id"}`, http.StatusBadRequest)
			return
		}

		// "Not coming" => remove any existing RSVP for this user.
		if req.Decline {
			if err := e.DeleteParticipant(ctx, bson.M{"user_id": req.UserID, "event_id": oid}); err != nil {
				e.Logger.Errorf("Failed to remove bot participant. Event: %s - Error: %s", id, err.Error())
				http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
				return
			}
			if err := e.Notification.RemoveUsersFromTopic(id, []string{req.UserID}); err != nil {
				e.Logger.Errorf("Failed to remove user from notification topic. Event: %s - Error: %s", id, err.Error())
			}
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte(`{"msg": "OK"}`))
			return
		}

		// Otherwise upsert the RSVP. If the user already has one, just update the status.
		existing, err := e.FindParticipants(ctx, bson.M{"event_id": oid, "user_id": req.UserID}, nil)
		if err != nil {
			e.Logger.Errorf("Failed to find participant. Event: %s - Error: %s", id, err.Error())
			http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			return
		}

		status := req.Status
		if len(existing) > 0 {
			if err := e.UpdateParticipant(ctx, bson.M{"_id": existing[0].ID}, bson.M{"$set": bson.M{"status": status}}); err != nil {
				e.Logger.Errorf("Failed to update participant. Event: %s - Error: %s", id, err.Error())
				http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
				return
			}
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte(`{"msg": "OK"}`))
			return
		}

		timestamp := bson.NewDateTimeFromTime(time.Now())
		participant := &models.ParticipantDao{
			UserID:    &req.UserID,
			EventID:   &oid,
			Status:    &status,
			CreatedAt: &timestamp,
		}
		if _, err := e.InsertParticipant(ctx, participant); err != nil {
			e.Logger.Errorf("Failed to insert bot participant. Event: %s - Error: %s", id, err.Error())
			http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			return
		}
		if err := e.Notification.AddUsersToTopic(id, []string{req.UserID}); err != nil {
			e.Logger.Errorf("Failed to add user to notification topic. Event: %s - Error: %s", id, err.Error())
		}

		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{"msg": "OK"}`))
	}
}
