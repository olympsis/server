package service

import (
	"context"
	"time"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Reasons returned to the caller (invite-service, over gRPC) when a participant
// was NOT added. All of these are benign, idempotent outcomes — not errors.
const (
	reasonAlreadyParticipant = "already_participant"
	reasonEventNotFound      = "event_not_found"
	reasonTeamRSVPRequired   = "team_rsvp_required"
	reasonPromoted           = "promoted" // an existing non-confirmed RSVP was upgraded to YES
)

// participantStore is the slice of persistence that addParticipantViaInvite
// needs. *Service satisfies it; tests supply an in-memory fake. FindEventByID
// returns (nil, nil) when the event is absent so a stale invite to a deleted
// event is a harmless no-op rather than an error.
type participantStore interface {
	FindEventByID(ctx context.Context, eventID bson.ObjectID) (*models.EventDao, error)
	FindParticipants(ctx context.Context, filter bson.M, opts *options.FindOptionsBuilder) ([]models.ParticipantDao, error)
	InsertParticipant(ctx context.Context, participant *models.ParticipantDao) (bson.ObjectID, error)
	UpdateParticipant(ctx context.Context, filter bson.M, update bson.M) error
}

// addParticipantViaInvite is the testable core behind the gRPC AddParticipant
// RPC. It loads the event, rejects the team-RSVP-only case, checks for an
// existing RSVP, then inserts a confirmed (YES) participant. It returns
// (added, participantID, reason, err):
//   - added == true, reason == ""            → the participant was newly created
//   - added == false, reason != "", err==nil → benign no-op (see the reason consts)
//   - err != nil                             → transient failure; the caller should retry
//
// NOTE: invited users join as confirmed participants even when the event is at
// MaxParticipants — they intentionally bypass the waitlist that the public
// "join event" REST endpoint (participants.go AddParticipant) applies. This is a
// deliberate product decision for the invite path.
func addParticipantViaInvite(ctx context.Context, store participantStore, eventID bson.ObjectID, userID string) (bool, bson.ObjectID, string, error) {
	event, err := store.FindEventByID(ctx, eventID)
	if err != nil {
		return false, bson.NilObjectID, "", err
	}
	if event == nil {
		// Event was deleted after the invite was sent — nothing to join.
		return false, bson.NilObjectID, reasonEventNotFound, nil
	}
	if teamRSVPRequired(event) {
		// This event only accepts team RSVPs (membership = the RSVP); an
		// individual participant row is invalid here. Benign no-op — the invite
		// still gets marked accepted.
		return false, bson.NilObjectID, reasonTeamRSVPRequired, nil
	}

	// Resolve the existing-RSVP case in-memory before writing. The unique
	// {user_id, event_id} index is still the real idempotency guard below; this
	// also lets us upgrade a non-confirmed prior RSVP.
	participants, err := store.FindParticipants(ctx, bson.M{"event_id": eventID}, nil)
	if err != nil {
		return false, bson.NilObjectID, "", err
	}
	for i := range participants {
		if participants[i].UserID == nil || *participants[i].UserID != userID {
			continue
		}
		existing := participants[i]
		// Already confirmed — a true idempotent no-op.
		if existing.Status != nil && *existing.Status == models.RSVPYes {
			return false, bson.NilObjectID, reasonAlreadyParticipant, nil
		}
		// The user has a non-confirmed prior RSVP (waitlist / maybe / can't).
		// Accepting the invite is an affirmative RSVP and invited users bypass
		// the cap, so promote them to a confirmed participant. added stays false:
		// the host was already notified when the original row was created, so we
		// must not re-publish rsvp.created. (mirrors the waitlist-promotion update
		// in participants.go — a bare status doc is rejected, so $set is required.)
		if err := store.UpdateParticipant(ctx, bson.M{"_id": existing.ID}, bson.M{"$set": bson.M{"status": models.RSVPYes}}); err != nil {
			return false, bson.NilObjectID, "", err
		}
		return false, bson.NilObjectID, reasonPromoted, nil
	}

	rsvp := models.RSVPYes
	now := bson.NewDateTimeFromTime(time.Now())
	participant := &models.ParticipantDao{
		UserID:    &userID,
		EventID:   &eventID,
		Status:    &rsvp,
		CreatedAt: &now,
	}
	pid, err := store.InsertParticipant(ctx, participant)
	if err != nil {
		// The unique {user_id, event_id} index is the real race backstop: a
		// concurrent accept/join that inserted first surfaces here as a
		// duplicate-key error. Treat it as the benign already_participant no-op,
		// exactly as the team path treats its duplicate-key case — erroring would
		// make the invite-accept retry loop forever.
		if mongo.IsDuplicateKeyError(err) {
			return false, bson.NilObjectID, reasonAlreadyParticipant, nil
		}
		return false, bson.NilObjectID, "", err
	}
	return true, pid, "", nil
}

// AddParticipantViaInvite is the entry point the gRPC EventTeamService calls when
// a user accepts an EVENT invite. It creates the individual RSVP row, then
// (best-effort) subscribes the user to the event's notification topic and
// announces the RSVP on the bus so notif-service pushes the "new participant"
// notification to the event host — mirroring the public AddParticipant flow.
func (s *Service) AddParticipantViaInvite(ctx context.Context, eventID bson.ObjectID, userID string) (bool, string, error) {
	added, pid, reason, err := addParticipantViaInvite(ctx, s, eventID, userID)
	if err != nil {
		return false, "", err
	}
	if added {
		// Add the participant to the event's notification topic (drives reminders).
		if aerr := s.Notification.AddUsersToTopic(eventID.Hex(), []string{userID}); aerr != nil {
			s.Logger.Errorf("Failed to add participant to event topic. Event: %s - Error: %s", eventID.Hex(), aerr.Error())
		}
		// Announce the RSVP. notif-service consumes this and delivers the
		// "new participant" push to the host (the joiner is excluded).
		s.publishRSVPCreated(ctx, pid, userID, eventID.Hex(), models.RSVPYes)
	}
	return added, reason, nil
}
