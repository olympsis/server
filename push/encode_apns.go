package push

import (
	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/payload"
)

// apnsTopic is the iOS app bundle id, which APNs uses as the notification topic.
const apnsTopic = "com.olympsis.client"

// buildAPNSPayload constructs the APNs payload for an event note in the strict
// shape the iOS Notification Service Extension enforces:
//
//   - Custom fields are TOP-LEVEL siblings of "aps" (payload.Custom does exactly
//     this — it does NOT nest them under aps).
//   - aps.alert.title is the literal event/group name and must be non-empty
//     (the constructors guarantee this).
//   - aps.mutable-content = 1 so the service extension runs.
//   - event_image_url is the RELATIVE storage path; the extension prepends the base.
//   - loc_args is a JSON array of strings, "[]" (never null) when empty.
//   - There is deliberately NO aps.alert.body and no legacy rich metadata.
func buildAPNSPayload(n EventNote) *payload.Payload {
	locArgs := n.LocArgs
	if locArgs == nil {
		locArgs = []string{} // marshal as [], never null
	}

	p := payload.NewPayload().
		AlertTitle(n.Title).
		MutableContent().
		Sound("default").
		Custom("type", string(n.Type)).
		Custom("event_id", n.EventID).
		Custom("event_image_url", relativeImagePath(n.ImagePath)).
		Custom("loc_key", n.LocKey()).
		Custom("loc_args", locArgs)

	if n.ParticipantID != "" {
		p.Custom("participant_id", n.ParticipantID)
	}
	if n.CommentID != "" {
		p.Custom("comment_id", n.CommentID)
	}
	return p
}

// buildAPNSNotification wraps the payload with the routing headers APNs needs.
// PushTypeAlert is required for an alert that runs a service extension; the topic
// is the app bundle id; PriorityHigh (10) delivers immediately for these
// user-facing notes.
func buildAPNSNotification(n EventNote, token string) *apns2.Notification {
	return &apns2.Notification{
		DeviceToken: token,
		Topic:       apnsTopic,
		PushType:    apns2.PushTypeAlert,
		Priority:    apns2.PriorityHigh,
		Payload:     buildAPNSPayload(n),
	}
}
