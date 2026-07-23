package service

import (
	"context"
	"testing"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// fakeParticipantStore is an in-memory participantStore for testing
// addParticipantViaInvite (the core the gRPC AddParticipant RPC delegates to)
// without a database.
type fakeParticipantStore struct {
	event        *models.EventDao        // returned by FindEventByID (nil => deleted)
	participants []models.ParticipantDao // returned by FindParticipants
	insertErr    error                   // returned by InsertParticipant (e.g. a duplicate-key error)
	insertCalls  int                     // how many times InsertParticipant was attempted
	updateCalls  int                     // how many times UpdateParticipant was attempted
	updated      bson.M                  // the update doc passed to UpdateParticipant
}

func (f *fakeParticipantStore) FindEventByID(_ context.Context, _ bson.ObjectID) (*models.EventDao, error) {
	return f.event, nil
}

func (f *fakeParticipantStore) FindParticipants(_ context.Context, _ bson.M, _ *options.FindOptionsBuilder) ([]models.ParticipantDao, error) {
	return f.participants, nil
}

func (f *fakeParticipantStore) InsertParticipant(_ context.Context, _ *models.ParticipantDao) (bson.ObjectID, error) {
	f.insertCalls++
	if f.insertErr != nil {
		return bson.NilObjectID, f.insertErr
	}
	return bson.NewObjectID(), nil
}

func (f *fakeParticipantStore) UpdateParticipant(_ context.Context, _ bson.M, update bson.M) error {
	f.updateCalls++
	f.updated = update
	return nil
}

// participantOf builds a confirmed (YES) participant row owned by userID.
func participantOf(userID string) models.ParticipantDao {
	id := bson.NewObjectID()
	yes := models.RSVPYes
	return models.ParticipantDao{ID: &id, UserID: &userID, Status: &yes}
}

// participantWithStatus builds a participant row for userID with the given RSVP
// status (used to test promotion of a non-confirmed prior RSVP).
func participantWithStatus(userID string, status models.RSVPStatus) models.ParticipantDao {
	id := bson.NewObjectID()
	return models.ParticipantDao{ID: &id, UserID: &userID, Status: &status}
}

// eventRequiringTeamRSVP builds an event whose TeamsConfig marks it team-RSVP
// only, so individual participants are rejected.
func eventRequiringTeamRSVP() *models.EventDao {
	required := true
	return &models.EventDao{TeamsConfig: &models.TeamsConfig{Required: &required}}
}

// dupKeyErr is a Mongo duplicate-key error, as InsertParticipant would return
// when the unique {user_id, event_id} index is violated.
var dupKeyErr = mongo.WriteException{
	WriteErrors: mongo.WriteErrors{{Code: 11000, Message: "E11000 duplicate key error"}},
}

func TestAddParticipantViaInvite(t *testing.T) {
	eventID := bson.NewObjectID()

	t.Run("adds a new participant when none exists", func(t *testing.T) {
		store := &fakeParticipantStore{event: &models.EventDao{}}
		added, _, reason, err := addParticipantViaInvite(context.Background(), store, eventID, "newbie")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !added || reason != "" {
			t.Fatalf("got added=%v reason=%q, want true/\"\"", added, reason)
		}
		if store.insertCalls != 1 {
			t.Errorf("expected exactly one insert, got %d", store.insertCalls)
		}
	})

	t.Run("deleted event is a harmless no-op", func(t *testing.T) {
		store := &fakeParticipantStore{event: nil}
		added, _, reason, err := addParticipantViaInvite(context.Background(), store, eventID, "newbie")
		if err != nil || added || reason != reasonEventNotFound {
			t.Fatalf("got added=%v reason=%q err=%v, want false/event_not_found/nil", added, reason, err)
		}
		if store.insertCalls != 0 {
			t.Errorf("must not attempt an insert for a missing event, got %d", store.insertCalls)
		}
	})

	t.Run("team-RSVP-only event is rejected", func(t *testing.T) {
		store := &fakeParticipantStore{event: eventRequiringTeamRSVP()}
		added, _, reason, err := addParticipantViaInvite(context.Background(), store, eventID, "newbie")
		if err != nil || added || reason != reasonTeamRSVPRequired {
			t.Fatalf("got added=%v reason=%q err=%v, want false/team_rsvp_required/nil", added, reason, err)
		}
		if store.insertCalls != 0 {
			t.Errorf("must not attempt an insert for a team-RSVP event, got %d", store.insertCalls)
		}
	})

	t.Run("already a confirmed participant is idempotent", func(t *testing.T) {
		store := &fakeParticipantStore{
			event:        &models.EventDao{},
			participants: []models.ParticipantDao{participantOf("newbie")},
		}
		added, _, reason, err := addParticipantViaInvite(context.Background(), store, eventID, "newbie")
		if err != nil || added || reason != reasonAlreadyParticipant {
			t.Fatalf("got added=%v reason=%q err=%v, want false/already_participant/nil", added, reason, err)
		}
		if store.insertCalls != 0 || store.updateCalls != 0 {
			t.Errorf("a confirmed participant must not insert or update, got insert=%d update=%d", store.insertCalls, store.updateCalls)
		}
	})

	t.Run("waitlisted participant is promoted to confirmed", func(t *testing.T) {
		store := &fakeParticipantStore{
			event:        &models.EventDao{},
			participants: []models.ParticipantDao{participantWithStatus("newbie", models.RSVPWaitlist)},
		}
		added, _, reason, err := addParticipantViaInvite(context.Background(), store, eventID, "newbie")
		if err != nil || added || reason != reasonPromoted {
			t.Fatalf("got added=%v reason=%q err=%v, want false/promoted/nil", added, reason, err)
		}
		if store.updateCalls != 1 {
			t.Fatalf("expected exactly one status update, got %d", store.updateCalls)
		}
		if store.insertCalls != 0 {
			t.Errorf("promotion must not insert a new row, got %d insert(s)", store.insertCalls)
		}
		// The update must set the status to YES via $set.
		set, _ := store.updated["$set"].(bson.M)
		if set == nil || set["status"] != models.RSVPYes {
			t.Errorf("expected $set status=YES, got %v", store.updated)
		}
	})

	t.Run("maybe participant is promoted to confirmed", func(t *testing.T) {
		store := &fakeParticipantStore{
			event:        &models.EventDao{},
			participants: []models.ParticipantDao{participantWithStatus("newbie", models.RSVPMaybe)},
		}
		added, _, reason, err := addParticipantViaInvite(context.Background(), store, eventID, "newbie")
		if err != nil || added || reason != reasonPromoted {
			t.Fatalf("got added=%v reason=%q err=%v, want false/promoted/nil", added, reason, err)
		}
		if store.updateCalls != 1 {
			t.Fatalf("expected exactly one status update, got %d", store.updateCalls)
		}
	})

	t.Run("duplicate-key on insert reports already_participant", func(t *testing.T) {
		// A concurrent accept/join inserted first between our find and insert.
		store := &fakeParticipantStore{event: &models.EventDao{}, insertErr: dupKeyErr}
		added, _, reason, err := addParticipantViaInvite(context.Background(), store, eventID, "newbie")
		if err != nil || added || reason != reasonAlreadyParticipant {
			t.Fatalf("got added=%v reason=%q err=%v, want false/already_participant/nil", added, reason, err)
		}
		if store.insertCalls != 1 {
			t.Errorf("expected one insert attempt, got %d", store.insertCalls)
		}
	})
}
