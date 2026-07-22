package service

import (
	"context"
	"errors"
	"time"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Reasons returned to the caller (invite-service, over gRPC) when a member was
// NOT added. All of these are benign, idempotent outcomes — not errors.
const (
	reasonAlreadyMember = "already_member"
	reasonTeamFull      = "team_full"
	reasonTeamNotFound  = "team_not_found"
	reasonConflict      = "conflict" // lost a race after passing the pre-checks
)

// teamMemberStore is the slice of persistence that addTeamMemberViaInvite needs.
// *Service satisfies it; tests supply an in-memory fake. FindTeamByID /
// FindEventByID return (nil, nil) when the document is absent so a stale invite
// to a deleted team is a harmless no-op rather than an error.
type teamMemberStore interface {
	FindTeamByID(ctx context.Context, teamID bson.ObjectID) (*models.TeamDao, error)
	FindEventByID(ctx context.Context, eventID bson.ObjectID) (*models.EventDao, error)
	AddTeamMemberAtomic(ctx context.Context, teamID bson.ObjectID, userID string, maxTeamSize *int32) (bool, error)
}

// maxTeamSizeFor returns the event's per-team size cap, or nil (unlimited).
func maxTeamSizeFor(event *models.EventDao) *int32 {
	if event != nil && event.TeamsConfig != nil {
		return event.TeamsConfig.MaxTeamSize
	}
	return nil
}

// addTeamMemberViaInvite is the testable core behind the gRPC AddTeamMember RPC.
// It loads the team, derives the capacity cap from the event, and performs an
// atomic capacity-guarded add. It returns (added, reason, err):
//   - added == true, reason == ""            → the member was newly added
//   - added == false, reason != "", err==nil → benign no-op (see the reason consts)
//   - err != nil                             → transient failure; the caller should retry
func addTeamMemberViaInvite(ctx context.Context, store teamMemberStore, teamID bson.ObjectID, userID string) (bool, string, error) {
	team, err := store.FindTeamByID(ctx, teamID)
	if err != nil {
		return false, "", err
	}
	if team == nil {
		// Team was deleted after the invite was sent — nothing to join.
		return false, reasonTeamNotFound, nil
	}

	var maxTeamSize *int32
	if team.EventID != nil {
		event, err := store.FindEventByID(ctx, *team.EventID)
		if err != nil {
			return false, "", err
		}
		maxTeamSize = maxTeamSizeFor(event)
	}

	// Resolve a precise reason in-memory before writing. The atomic add below is
	// still the real guard; this only makes the response informative.
	switch err := canAddTeamMember(teamMembers(team), userID, maxTeamSize); {
	case errors.Is(err, errAlreadyMember):
		return false, reasonAlreadyMember, nil
	case errors.Is(err, errTeamFull):
		return false, reasonTeamFull, nil
	}

	added, err := store.AddTeamMemberAtomic(ctx, teamID, userID, maxTeamSize)
	if err != nil {
		return false, "", err
	}
	if !added {
		// Passed the pre-checks but the atomic update matched nothing — another
		// request added them or filled the last seat first. Benign.
		return false, reasonConflict, nil
	}
	return true, "", nil
}

// AddTeamMemberViaInvite is the entry point the gRPC EventTeamService calls when
// a user accepts a TEAM invite. Team membership IS the RSVP, so this is what
// turns an accepted invite into an attendee.
func (s *Service) AddTeamMemberViaInvite(ctx context.Context, teamID bson.ObjectID, userID string) (bool, string, error) {
	added, reason, err := addTeamMemberViaInvite(ctx, s, teamID, userID)
	if err != nil {
		return false, "", err
	}
	if added {
		// Best-effort: put the new member on the event's notification topic so
		// they receive event reminders, mirroring AddParticipant.
		if team, terr := s.FindTeamByID(ctx, teamID); terr == nil && team != nil && team.EventID != nil {
			if aerr := s.Notification.AddUsersToTopic(team.EventID.Hex(), []string{userID}); aerr != nil {
				s.Logger.Errorf("Failed to add team member to event topic. Team: %s - Error: %s", teamID.Hex(), aerr.Error())
			}
		}
	}
	return added, reason, nil
}

/*****************
DATABASE FUNCTIONS
******************/

// FindTeamByID looks up a team by its id, translating "not found" to (nil, nil)
// so callers can treat a deleted team as a no-op.
func (s *Service) FindTeamByID(ctx context.Context, teamID bson.ObjectID) (*models.TeamDao, error) {
	team, err := s.FindTeam(ctx, bson.M{"_id": teamID})
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return team, err
}

// FindEventByID looks up an event by its id, translating "not found" to
// (nil, nil).
func (s *Service) FindEventByID(ctx context.Context, eventID bson.ObjectID) (*models.EventDao, error) {
	event, err := s.FindEvent(ctx, bson.M{"_id": eventID})
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	return event, err
}

// AddTeamMemberAtomic appends userID to the team as a MEMBER in a single atomic
// update, so concurrent joins can't race past MaxTeamSize. The filter refuses to
// match when the user is already a member (idempotency) or the roster is full;
// maxTeamSize == nil or <= 0 means unlimited. Returns whether a member was added.
func (s *Service) AddTeamMemberAtomic(ctx context.Context, teamID bson.ObjectID, userID string, maxTeamSize *int32) (bool, error) {
	role := models.MemberMember
	isAnonymous := false
	now := bson.NewDateTimeFromTime(time.Now())
	member := models.TeamMemberDao{
		UserID:      &userID,
		Role:        &role,
		IsAnonymous: &isAnonymous,
		JoinedAt:    &now,
	}

	filter := bson.M{
		"_id":             teamID,
		"members.user_id": bson.M{"$ne": userID}, // not already a member
	}
	if maxTeamSize != nil && *maxTeamSize > 0 {
		filter["$expr"] = bson.M{"$lt": bson.A{bson.M{"$size": "$members"}, *maxTeamSize}}
	}

	res, err := s.Database.EventTeamsCollection.UpdateOne(ctx, filter, bson.M{"$push": bson.M{"members": member}})
	if err != nil {
		return false, err
	}
	return res.ModifiedCount > 0, nil
}
