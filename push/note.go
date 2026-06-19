// Package push delivers the event push notifications (reminder, new participant,
// new comment) to iOS via direct APNs and to Android via FCM v1.
//
// The design keeps three concerns separate so each can change without touching
// the others:
//
//   - EventNote (this file) is a transport-neutral description of a notification.
//     It says nothing about wire format.
//   - The encoders (encode_apns.go, encode_fcm.go) translate an EventNote into
//     each platform's payload — this is the ONLY place per-transport shape lives.
//   - The dispatcher + senders (dispatch.go, sender.go) resolve recipients,
//     persist the audit records, and put bytes on the wire.
//
// Localization is done ON-DEVICE for both platforms: the server only ships a
// loc_key + loc_args; it never builds a localized body string.
package push

import (
	"fmt"

	"github.com/olympsis/models"
)

// locKeys is the single source of truth mapping a notification type to the
// on-device localization key. These MUST match the iOS catalog
// (Notifications.xcstrings) and the Android strings.xml VERBATIM — if a key
// drifts, both platforms silently fall back to a title-only banner.
var locKeys = map[models.NotificationType]string{
	models.EventReminderType:          "event-starting-soon", // plural, formats %lld minutes
	models.EventParticipantUpdateType: "event-new-participant",
	models.NewEventCommentType:        "event-new-comment",
}

// EventNote is a transport-neutral description of an event push notification.
// The encoders absorb every per-platform difference; nothing here is wire-shaped.
type EventNote struct {
	Type      models.NotificationType
	EventID   string
	Title     string   // literal event/group name; MUST be non-empty (iOS NSE ignores edits if title is empty)
	ImagePath string   // image as stored on the event; may be absolute or relative (helpers normalize per-platform)
	LocArgs   []string // always strings, e.g. ["30"]; never numbers, and [] (not nil) when there are no args

	ParticipantID string // participant note only — routes the tap to the Participants screen
	CommentID     string // comment note only — routes the tap to the specific comment
}

// LocKey returns the on-device localization key for this note's type.
func (n EventNote) LocKey() string { return locKeys[n.Type] }

// ---- Per-type constructors: required fields + validation live here ----

// NewReminder builds an "event starts in N minutes" note. minutes is rendered as
// a string loc arg so the device applies its own plural rule against %lld.
func NewReminder(eventID, title, imagePath string, minutes int) (EventNote, error) {
	if title == "" {
		return EventNote{}, fmt.Errorf("reminder: title must be non-empty")
	}
	return EventNote{
		Type:      models.EventReminderType,
		EventID:   eventID,
		Title:     title,
		ImagePath: imagePath,
		LocArgs:   []string{fmt.Sprintf("%d", minutes)},
	}, nil
}

// NewParticipant builds a "new participant joined" note. participantID routes the
// notification tap; it must be present.
func NewParticipant(eventID, title, imagePath, participantID string) (EventNote, error) {
	if title == "" || participantID == "" {
		return EventNote{}, fmt.Errorf("participant: title and participantID required")
	}
	return EventNote{
		Type:          models.EventParticipantUpdateType,
		EventID:       eventID,
		Title:         title,
		ImagePath:     imagePath,
		ParticipantID: participantID,
		LocArgs:       []string{},
	}, nil
}

// NewComment builds a "new comment on event" note. commentID routes the tap to
// the specific comment; it must be present.
func NewComment(eventID, title, imagePath, commentID string) (EventNote, error) {
	if title == "" || commentID == "" {
		return EventNote{}, fmt.Errorf("comment: title and commentID required")
	}
	return EventNote{
		Type:      models.NewEventCommentType,
		EventID:   eventID,
		Title:     title,
		ImagePath: imagePath,
		CommentID: commentID,
		LocArgs:   []string{},
	}, nil
}

// auditData flattens the note's routing fields into the map stored on the
// PushNotification audit record (and surfaced to the in-app inbox). The body is
// intentionally empty server-side — clients localize from loc_key/loc_args.
func (n EventNote) auditData() map[string]any {
	data := map[string]any{
		"type":     string(n.Type),
		"event_id": n.EventID,
		"loc_key":  n.LocKey(),
		"loc_args": n.LocArgs,
	}
	if n.ParticipantID != "" {
		data["participant_id"] = n.ParticipantID
	}
	if n.CommentID != "" {
		data["comment_id"] = n.CommentID
	}
	return data
}
