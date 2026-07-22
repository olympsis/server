package service

import (
	"context"
	"testing"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// fakeTeamStore is an in-memory teamMemberStore for testing addTeamMemberViaInvite
// (the core the gRPC AddTeamMember RPC delegates to) without a database.
type fakeTeamStore struct {
	team     *models.TeamDao
	event    *models.EventDao
	added    bool // what AddTeamMemberAtomic reports
	addCalls int  // how many times the atomic add was attempted
}

func (f *fakeTeamStore) FindTeamByID(_ context.Context, _ bson.ObjectID) (*models.TeamDao, error) {
	return f.team, nil
}

func (f *fakeTeamStore) FindEventByID(_ context.Context, _ bson.ObjectID) (*models.EventDao, error) {
	return f.event, nil
}

func (f *fakeTeamStore) AddTeamMemberAtomic(_ context.Context, _ bson.ObjectID, _ string, _ *int32) (bool, error) {
	f.addCalls++
	return f.added, nil
}

func teamWithEvent(eventID bson.ObjectID, members ...models.TeamMemberDao) *models.TeamDao {
	m := members
	return &models.TeamDao{EventID: &eventID, Members: &m}
}

func eventWithMaxTeamSize(max int32) *models.EventDao {
	return &models.EventDao{TeamsConfig: &models.TeamsConfig{MaxTeamSize: &max}}
}

func TestAddTeamMemberViaInvite(t *testing.T) {
	eventID := bson.NewObjectID()
	teamID := bson.NewObjectID()

	t.Run("adds a new member when there is room", func(t *testing.T) {
		store := &fakeTeamStore{
			team:  teamWithEvent(eventID, member("owner", models.OwnerMember)),
			event: eventWithMaxTeamSize(4),
			added: true,
		}
		added, reason, err := addTeamMemberViaInvite(context.Background(), store, teamID, "newbie")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !added || reason != "" {
			t.Fatalf("got added=%v reason=%q, want true/\"\"", added, reason)
		}
		if store.addCalls != 1 {
			t.Errorf("expected exactly one atomic add, got %d", store.addCalls)
		}
	})

	t.Run("deleted team is a harmless no-op", func(t *testing.T) {
		store := &fakeTeamStore{team: nil}
		added, reason, err := addTeamMemberViaInvite(context.Background(), store, teamID, "newbie")
		if err != nil || added || reason != reasonTeamNotFound {
			t.Fatalf("got added=%v reason=%q err=%v, want false/team_not_found/nil", added, reason, err)
		}
		if store.addCalls != 0 {
			t.Errorf("must not attempt an add for a missing team, got %d", store.addCalls)
		}
	})

	t.Run("already a member is idempotent", func(t *testing.T) {
		store := &fakeTeamStore{
			team:  teamWithEvent(eventID, member("owner", models.OwnerMember), member("newbie", models.MemberMember)),
			event: eventWithMaxTeamSize(4),
		}
		added, reason, err := addTeamMemberViaInvite(context.Background(), store, teamID, "newbie")
		if err != nil || added || reason != reasonAlreadyMember {
			t.Fatalf("got added=%v reason=%q err=%v, want false/already_member/nil", added, reason, err)
		}
		if store.addCalls != 0 {
			t.Errorf("must not attempt an add for an existing member, got %d", store.addCalls)
		}
	})

	t.Run("full team is rejected", func(t *testing.T) {
		store := &fakeTeamStore{
			team:  teamWithEvent(eventID, member("owner", models.OwnerMember), member("m2", models.MemberMember)),
			event: eventWithMaxTeamSize(2), // already at 2 members
		}
		added, reason, err := addTeamMemberViaInvite(context.Background(), store, teamID, "newbie")
		if err != nil || added || reason != reasonTeamFull {
			t.Fatalf("got added=%v reason=%q err=%v, want false/team_full/nil", added, reason, err)
		}
		if store.addCalls != 0 {
			t.Errorf("must not attempt an add for a full team, got %d", store.addCalls)
		}
	})

	t.Run("lost race after pre-checks reports conflict", func(t *testing.T) {
		store := &fakeTeamStore{
			team:  teamWithEvent(eventID, member("owner", models.OwnerMember)),
			event: eventWithMaxTeamSize(4),
			added: false, // atomic add matched nothing (someone else won the seat)
		}
		added, reason, err := addTeamMemberViaInvite(context.Background(), store, teamID, "newbie")
		if err != nil || added || reason != reasonConflict {
			t.Fatalf("got added=%v reason=%q err=%v, want false/conflict/nil", added, reason, err)
		}
		if store.addCalls != 1 {
			t.Errorf("expected one atomic add attempt, got %d", store.addCalls)
		}
	})
}
