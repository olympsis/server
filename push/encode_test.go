package push

import (
	"encoding/json"
	"testing"

	"github.com/olympsis/models"
)

// notes builds one of each event note for the golden tests. The image path is
// stored absolute on purpose so we exercise the iOS strip / FCM keep behavior.
func notes(t *testing.T) map[string]EventNote {
	t.Helper()

	const absImage = "https://storage.googleapis.com/olympsis-event-media/abc.jpg"

	reminder, err := NewReminder("evt1", "Pickup Soccer", absImage, 30)
	if err != nil {
		t.Fatalf("NewReminder: %v", err)
	}
	participant, err := NewParticipant("evt1", "Pickup Soccer", absImage, "part1")
	if err != nil {
		t.Fatalf("NewParticipant: %v", err)
	}
	comment, err := NewComment("evt1", "Pickup Soccer", absImage, "cmt1")
	if err != nil {
		t.Fatalf("NewComment: %v", err)
	}
	return map[string]EventNote{
		"reminder":    reminder,
		"participant": participant,
		"comment":     comment,
	}
}

// marshalAPNS renders the APNs payload to the generic JSON shape that actually
// goes over the wire, so we can assert on top-level siblings of "aps".
func marshalAPNS(t *testing.T, n EventNote) map[string]any {
	t.Helper()
	raw, err := json.Marshal(buildAPNSPayload(n))
	if err != nil {
		t.Fatalf("marshal apns: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal apns: %v", err)
	}
	return m
}

func TestAPNSPayloadShape(t *testing.T) {
	all := notes(t)
	for name, n := range all {
		t.Run(name, func(t *testing.T) {
			m := marshalAPNS(t, n)

			// aps must exist with mutable-content == 1 and a non-empty alert title,
			// and must NOT carry an alert body.
			aps, ok := m["aps"].(map[string]any)
			if !ok {
				t.Fatalf("aps missing or wrong type: %#v", m["aps"])
			}
			if mc, _ := aps["mutable-content"].(float64); mc != 1 {
				t.Errorf("aps.mutable-content = %v, want 1", aps["mutable-content"])
			}
			alert, ok := aps["alert"].(map[string]any)
			if !ok {
				t.Fatalf("aps.alert missing or wrong type: %#v", aps["alert"])
			}
			if title, _ := alert["title"].(string); title == "" {
				t.Errorf("aps.alert.title is empty")
			}
			if _, exists := alert["body"]; exists {
				t.Errorf("aps.alert.body must not be set, got %v", alert["body"])
			}

			// Custom fields must be TOP-LEVEL siblings of aps (not nested in it).
			for _, key := range []string{"type", "event_id", "event_image_url", "loc_key", "loc_args"} {
				if _, exists := m[key]; !exists {
					t.Errorf("top-level field %q missing", key)
				}
				if _, nested := aps[key]; nested {
					t.Errorf("field %q must be a sibling of aps, found nested under aps", key)
				}
			}

			// loc_key must match the catalog mapping verbatim.
			if got, want := m["loc_key"], n.LocKey(); got != want {
				t.Errorf("loc_key = %v, want %v", got, want)
			}

			// event_image_url must be the RELATIVE tail for iOS.
			if got := m["event_image_url"]; got != "event-media/abc.jpg" {
				t.Errorf("event_image_url = %v, want relative tail", got)
			}
		})
	}
}

func TestAPNSLocArgs(t *testing.T) {
	all := notes(t)

	// Reminder carries ["30"] — a string, never a number.
	reminder := marshalAPNS(t, all["reminder"])
	args, ok := reminder["loc_args"].([]any)
	if !ok || len(args) != 1 {
		t.Fatalf("reminder loc_args = %#v, want 1 element", reminder["loc_args"])
	}
	if s, isString := args[0].(string); !isString || s != "30" {
		t.Errorf("reminder loc_args[0] = %#v, want string \"30\"", args[0])
	}

	// Participant/comment carry an empty array — [] (present), never null.
	for _, name := range []string{"participant", "comment"} {
		m := marshalAPNS(t, all[name])
		raw, exists := m["loc_args"]
		if !exists {
			t.Errorf("%s: loc_args missing (must be [] not absent)", name)
			continue
		}
		if raw == nil {
			t.Errorf("%s: loc_args is null, want []", name)
			continue
		}
		if arr, ok := raw.([]any); !ok || len(arr) != 0 {
			t.Errorf("%s: loc_args = %#v, want empty array", name, raw)
		}
	}
}

func TestAPNSRoutingFields(t *testing.T) {
	all := notes(t)

	if m := marshalAPNS(t, all["participant"]); m["participant_id"] != "part1" {
		t.Errorf("participant_id = %v, want part1", m["participant_id"])
	}
	if m := marshalAPNS(t, all["participant"]); m["comment_id"] != nil {
		t.Errorf("participant note must not carry comment_id, got %v", m["comment_id"])
	}
	if m := marshalAPNS(t, all["comment"]); m["comment_id"] != "cmt1" {
		t.Errorf("comment_id = %v, want cmt1", m["comment_id"])
	}
	if m := marshalAPNS(t, all["comment"]); m["participant_id"] != nil {
		t.Errorf("comment note must not carry participant_id, got %v", m["participant_id"])
	}
}

func TestFCMMessageShape(t *testing.T) {
	all := notes(t)
	for name, n := range all {
		t.Run(name, func(t *testing.T) {
			msg := buildFCMMessage(n, "tok1")

			if msg.Token != "tok1" {
				t.Errorf("token = %q, want tok1", msg.Token)
			}
			if msg.Android == nil || msg.Android.Notification == nil {
				t.Fatalf("android notification missing")
			}
			an := msg.Android.Notification

			if an.Title == "" {
				t.Errorf("android title is empty")
			}
			if an.BodyLocKey != n.LocKey() {
				t.Errorf("body_loc_key = %q, want %q", an.BodyLocKey, n.LocKey())
			}
			// image must be ABSOLUTE for FCM.
			if an.ImageURL != "https://storage.googleapis.com/olympsis-event-media/abc.jpg" {
				t.Errorf("image = %q, want absolute URL", an.ImageURL)
			}

			// data values are all strings (map type enforces it) and carry the routing fields.
			if msg.Data["type"] != string(n.Type) {
				t.Errorf("data.type = %q, want %q", msg.Data["type"], string(n.Type))
			}
			if msg.Data["event_id"] != "evt1" {
				t.Errorf("data.event_id = %q, want evt1", msg.Data["event_id"])
			}
		})
	}
}

func TestFCMBodyLocArgs(t *testing.T) {
	all := notes(t)

	// Reminder sets body_loc_args ["30"]; the SDK requires body_loc_key alongside,
	// which we always set.
	if an := buildFCMMessage(all["reminder"], "t").Android.Notification; len(an.BodyLocArgs) != 1 || an.BodyLocArgs[0] != "30" {
		t.Errorf("reminder body_loc_args = %#v, want [\"30\"]", an.BodyLocArgs)
	}

	// No-arg notes omit body_loc_args entirely (the SDK rejects empty args).
	if an := buildFCMMessage(all["comment"], "t").Android.Notification; len(an.BodyLocArgs) != 0 {
		t.Errorf("comment body_loc_args = %#v, want none", an.BodyLocArgs)
	}
}

// TestLocKeysMatchContract guards the type->loc_key map against drift.
func TestLocKeysMatchContract(t *testing.T) {
	want := map[models.NotificationType]string{
		models.EventReminderType:          "event-starting-soon",
		models.EventParticipantUpdateType: "event-new-participant",
		models.NewEventCommentType:        "event-new-comment",
	}
	for typ, key := range want {
		if locKeys[typ] != key {
			t.Errorf("locKeys[%q] = %q, want %q", typ, locKeys[typ], key)
		}
	}
}

// TestTypeRoutingStrings locks the exact `type` strings the iOS NSE and Android
// handler switch on. If the models dependency hasn't been bumped to carry
// event_comment, this fails — catching the missing bump before deploy.
func TestTypeRoutingStrings(t *testing.T) {
	want := map[models.NotificationType]string{
		models.EventReminderType:          "event_reminder",
		models.EventParticipantUpdateType: "event_participant_update",
		models.NewEventCommentType:        "event_comment",
	}
	for typ, str := range want {
		if string(typ) != str {
			t.Errorf("type %v = %q, want %q (is the models dependency bumped?)", typ, string(typ), str)
		}
	}
}
