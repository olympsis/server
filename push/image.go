package push

import "strings"

// imageBaseURL is the public Cloud Storage prefix for Olympsis media. The two
// transports want OPPOSITE forms of the image URL:
//   - iOS: the RELATIVE tail (e.g. "event-media/<uuid>.jpg"). The Notification
//     Service Extension prepends this base itself.
//   - FCM: the ABSOLUTE URL.
//
// The event's stored MediaURL isn't guaranteed to be one or the other, so the
// helpers below normalize in both directions defensively.
const imageBaseURL = "https://storage.googleapis.com/olympsis-"

// relativeImagePath returns the storage-relative tail for the iOS payload. If the
// stored value is already absolute it strips the base prefix; an already-relative
// value is returned unchanged.
func relativeImagePath(stored string) string {
	return strings.TrimPrefix(stored, imageBaseURL)
}

// absoluteImageURL returns a fully-qualified URL for the FCM payload, prepending
// the base prefix when the stored value is relative. Empty in, empty out.
func absoluteImageURL(stored string) string {
	if stored == "" {
		return ""
	}
	if strings.HasPrefix(stored, "http://") || strings.HasPrefix(stored, "https://") {
		return stored
	}
	return imageBaseURL + stored
}
