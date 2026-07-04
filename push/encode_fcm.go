package push

import "firebase.google.com/go/messaging"

// buildFCMMessage constructs the FCM v1 message for an event note:
//
//   - android.notification carries the title plus body_loc_key / body_loc_args,
//     which the device resolves on-device against strings.xml.
//   - image is the ABSOLUTE URL (opposite of iOS, which wants the relative tail).
//   - Routing fields live in data, whose values must all be strings — the
//     map[string]string type enforces that at compile time.
//
// Note: the Admin SDK rejects body_loc_args without a body_loc_key
// (messaging_utils.go), so we only set args when we also have a key — which we
// always do here.
func buildFCMMessage(n EventNote, token string) *messaging.Message {
	notif := &messaging.AndroidNotification{
		Title:      n.Title,
		BodyLocKey: n.LocKey(),
		ImageURL:   absoluteImageURL(n.ImagePath), // JSON "image"; FCM requires an absolute URL
	}
	if len(n.LocArgs) > 0 {
		notif.BodyLocArgs = n.LocArgs
	}

	data := map[string]string{
		"type":     string(n.Type),
		"event_id": n.EventID,
	}
	if n.ParticipantID != "" {
		data["participant_id"] = n.ParticipantID
	}
	if n.CommentID != "" {
		data["comment_id"] = n.CommentID
	}

	return &messaging.Message{
		Token:   token,
		Android: &messaging.AndroidConfig{Notification: notif},
		Data:    data,
	}
}
