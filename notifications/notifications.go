package notifications

import (
	"time"

	"github.com/sirupsen/logrus"
)

type NotificationProcess struct {
	logger logrus.Logger
}

func (n *NotificationProcess) SendNewEventNotification(eventID string) error {
	return nil
}

// Event starts soon
func (n *NotificationProcess) SendEventReminderNotification(eventID string, time time.Duration) error {
	return nil
}
