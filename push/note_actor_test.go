package push

import "testing"

// The audit record feeds the in-app inbox, so actor_id has to survive into it —
// that's what lets a client render "<name> commented on your event".
func TestAuditDataCarriesActor(t *testing.T) {
	note, err := NewComment("event-1", "Sunday Run", "", "comment-1", "actor-1")
	if err != nil {
		t.Fatalf("NewComment: %v", err)
	}
	if note.ActorID != "actor-1" {
		t.Errorf("ActorID = %q, want actor-1", note.ActorID)
	}
	if got := note.auditData()["actor_id"]; got != "actor-1" {
		t.Errorf("audit actor_id = %v, want actor-1", got)
	}

	participant, err := NewParticipant("event-1", "Sunday Run", "", "p-1", "actor-2")
	if err != nil {
		t.Fatalf("NewParticipant: %v", err)
	}
	if got := participant.auditData()["actor_id"]; got != "actor-2" {
		t.Errorf("audit actor_id = %v, want actor-2", got)
	}
}

// A reminder is system-triggered: there is no actor, and the key must be absent
// rather than present-and-empty.
func TestAuditDataOmitsActorForReminder(t *testing.T) {
	note, err := NewReminder("event-1", "Sunday Run", "", 30)
	if err != nil {
		t.Fatalf("NewReminder: %v", err)
	}
	if _, ok := note.auditData()["actor_id"]; ok {
		t.Error("actor_id must be omitted when there is no actor")
	}
}
