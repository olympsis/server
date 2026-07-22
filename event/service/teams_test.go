package service

import (
	"errors"
	"testing"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// --- test builders ------------------------------------------------------------

func member(uid string, role models.MemberRole) models.TeamMemberDao {
	r := role
	return models.TeamMemberDao{UserID: &uid, Role: &r}
}

// legacyMember has no role, like a team created before the role field existed.
func legacyMember(uid string) models.TeamMemberDao {
	return models.TeamMemberDao{UserID: &uid}
}

func team(members ...models.TeamMemberDao) *models.TeamDao {
	m := members
	return &models.TeamDao{Members: &m}
}

func i32(v int32) *int32 { return &v }
func boolp(v bool) *bool { return &v }

// --- teamRSVPRequired ---------------------------------------------------------

func TestTeamRSVPRequired(t *testing.T) {
	cases := []struct {
		name  string
		event *models.EventDao
		want  bool
	}{
		{"nil event", nil, false},
		{"nil config", &models.EventDao{}, false},
		{"config, required nil", &models.EventDao{TeamsConfig: &models.TeamsConfig{}}, false},
		{"required false", &models.EventDao{TeamsConfig: &models.TeamsConfig{Required: boolp(false)}}, false},
		{"required true", &models.EventDao{TeamsConfig: &models.TeamsConfig{Required: boolp(true)}}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := teamRSVPRequired(tc.event); got != tc.want {
				t.Errorf("teamRSVPRequired = %v, want %v", got, tc.want)
			}
		})
	}
}

// --- ownership ----------------------------------------------------------------

func TestTeamOwnerRoleAware(t *testing.T) {
	tm := team(member("creator", models.OwnerMember), member("m2", models.MemberMember))
	owner, ok := teamOwnerID(tm)
	if !ok || owner != "creator" {
		t.Fatalf("teamOwnerID = %q,%v, want creator,true", owner, ok)
	}
	if !isTeamOwner(tm, "creator") {
		t.Error("creator should be owner")
	}
	if isTeamOwner(tm, "m2") {
		t.Error("m2 should not be owner")
	}
}

// TestTeamOwnerLegacyFallback: a team with no OWNER role (pre-role data) falls
// back to the first member, preserving the old creator-is-owner behaviour.
func TestTeamOwnerLegacyFallback(t *testing.T) {
	tm := team(legacyMember("creator"), legacyMember("m2"))
	owner, ok := teamOwnerID(tm)
	if !ok || owner != "creator" {
		t.Fatalf("legacy teamOwnerID = %q,%v, want creator,true", owner, ok)
	}
}

func TestCanLeaveTeam(t *testing.T) {
	tm := team(member("owner", models.OwnerMember), member("m2", models.MemberMember))
	if canLeaveTeam(tm, "owner") {
		t.Error("owner must not be able to leave freely")
	}
	if !canLeaveTeam(tm, "m2") {
		t.Error("a non-owner member should be able to leave")
	}
	if canLeaveTeam(tm, "stranger") {
		t.Error("a non-member cannot leave")
	}
}

// --- ownership transfer -------------------------------------------------------

func TestApplyOwnershipTransfer(t *testing.T) {
	members := []models.TeamMemberDao{
		member("owner", models.OwnerMember),
		member("m2", models.MemberMember),
	}
	updated, err := applyOwnershipTransfer(members, "owner", "m2")
	if err != nil {
		t.Fatalf("transfer: %v", err)
	}

	owners := 0
	roleOf := map[string]models.MemberRole{}
	for _, m := range updated {
		roleOf[*m.UserID] = *m.Role
		if *m.Role == models.OwnerMember {
			owners++
		}
	}
	if owners != 1 {
		t.Fatalf("expected exactly one owner after transfer, got %d", owners)
	}
	if roleOf["m2"] != models.OwnerMember {
		t.Errorf("m2 should be OWNER, got %s", roleOf["m2"])
	}
	if roleOf["owner"] != models.MemberMember {
		t.Errorf("old owner should be MEMBER, got %s", roleOf["owner"])
	}
}

func TestApplyOwnershipTransferToNonMember(t *testing.T) {
	members := []models.TeamMemberDao{member("owner", models.OwnerMember)}
	if _, err := applyOwnershipTransfer(members, "owner", "ghost"); !errors.Is(err, errNewOwnerNotFound) {
		t.Fatalf("expected errNewOwnerNotFound, got %v", err)
	}
}

// --- capacity / duplicates ----------------------------------------------------

func TestCanAddTeamMember(t *testing.T) {
	full := []models.TeamMemberDao{member("a", models.OwnerMember), member("b", models.MemberMember)}

	if err := canAddTeamMember(full, "c", i32(2)); !errors.Is(err, errTeamFull) {
		t.Errorf("expected errTeamFull at capacity, got %v", err)
	}
	if err := canAddTeamMember(full, "a", i32(5)); !errors.Is(err, errAlreadyMember) {
		t.Errorf("expected errAlreadyMember for a dup, got %v", err)
	}
	if err := canAddTeamMember(full, "c", nil); err != nil {
		t.Errorf("nil max = unlimited, want nil err, got %v", err)
	}
	if err := canAddTeamMember(full, "c", i32(0)); err != nil {
		t.Errorf("max 0 = unlimited, want nil err, got %v", err)
	}
	if err := canAddTeamMember(full, "c", i32(3)); err != nil {
		t.Errorf("under capacity, want nil err, got %v", err)
	}
}

func TestAddTeamMemberAppendsAsMember(t *testing.T) {
	members := []models.TeamMemberDao{member("owner", models.OwnerMember)}
	updated, err := addTeamMember(members, "newbie", i32(4), bson.DateTime(0))
	if err != nil {
		t.Fatalf("addTeamMember: %v", err)
	}
	if len(updated) != 2 {
		t.Fatalf("expected 2 members, got %d", len(updated))
	}
	added := updated[len(updated)-1]
	if added.UserID == nil || *added.UserID != "newbie" {
		t.Errorf("wrong appended user: %v", added.UserID)
	}
	if added.Role == nil || *added.Role != models.MemberMember {
		t.Errorf("appended member should have MEMBER role, got %v", added.Role)
	}
}
