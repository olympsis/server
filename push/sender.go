package push

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"firebase.google.com/go/messaging"
	"github.com/sideshow/apns2"
)

// sendResult reports the outcome of a single push so the dispatcher can log it
// and, when the token is permanently invalid, deactivate the device record.
type sendResult struct {
	sent      bool  // delivered to the platform gateway
	deadToken bool  // token is permanently invalid — stop sending to it
	err       error // non-nil on any non-success (transport error or rejection)
}

// sender puts an encoded note on the wire for one transport.
type sender interface {
	send(n EventNote, token string) sendResult
}

// apnsSender delivers via the existing direct-APNs HTTP/2 client.
type apnsSender struct{ client *apns2.Client }

func (s apnsSender) send(n EventNote, token string) sendResult {
	res, err := s.client.Push(buildAPNSNotification(n, token))
	if err != nil {
		// Transport/connection error — not the token's fault; let it retry later.
		return sendResult{err: err}
	}
	if res.Sent() {
		return sendResult{sent: true}
	}
	// 410 Gone, or an explicit bad-token reason, means the token is dead.
	dead := res.StatusCode == http.StatusGone ||
		res.Reason == apns2.ReasonUnregistered ||
		res.Reason == apns2.ReasonBadDeviceToken ||
		res.Reason == apns2.ReasonDeviceTokenNotForTopic
	return sendResult{
		deadToken: dead,
		err:       fmt.Errorf("apns not sent: %d %s %s", res.StatusCode, res.ApnsID, res.Reason),
	}
}

// fcmSender delivers via the Firebase Admin SDK, which handles FCM v1 + the
// service-account OAuth2 bearer token internally.
type fcmSender struct{ client *messaging.Client }

func (s fcmSender) send(n EventNote, token string) sendResult {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if _, err := s.client.Send(ctx, buildFCMMessage(n, token)); err != nil {
		// An unregistered token is permanently invalid (app uninstalled, token
		// rotated); deactivate it so we stop trying.
		if messaging.IsRegistrationTokenNotRegistered(err) {
			return sendResult{deadToken: true, err: err}
		}
		return sendResult{err: err}
	}
	return sendResult{sent: true}
}
