package notifications

import (
	"time"

	"github.com/sirupsen/logrus"
)

type NotificationService struct {
	logger *logrus.Logger
}

func NewNotificationService(l *logrus.Logger) *NotificationService {
	return &NotificationService{
		logger: l,
	}
}

func (n *NotificationService) SendNewEventNotification(eventID string) error {
	return nil
}

// Event starts soon
func (n *NotificationService) SendEventReminderNotification(eventID string, time time.Duration) error {
	return nil
}
