package service

import (
	"olympsis-server/notifications"
	"olympsis-server/redis"
	"time"

	"github.com/sirupsen/logrus"
)

type NotificationReminderQueue struct {
	items      []string
	retries    map[string]int
	maxRetries int
	logger     *logrus.Logger
}

func NewEventNotificationQueue(l *logrus.Logger) *NotificationReminderQueue {
	return &NotificationReminderQueue{
		items:      make([]string, 0),
		retries:    make(map[string]int),
		maxRetries: 3,
		logger:     l,
	}
}

func (q *NotificationReminderQueue) Add(eventID string) {
	q.items = append(q.items, eventID)
}

func (q *NotificationReminderQueue) ProcessWithRetry(sender *notifications.NotificationService, cache *redis.RedisDatabase, eventStopTime time.Time) {
	var failures []string

	for _, eventID := range q.items {
		if err := sender.SendEventReminderNotification(eventID, time.Duration(30)*time.Minute); err != nil {
			q.retries[eventID]++
			if q.retries[eventID] < q.maxRetries {
				failures = append(failures, eventID)
			} else {
				q.logger.Errorf("Failed to send notification for event: %s after %d retries: %v",
					eventID, q.maxRetries, err)
			}
			continue
		}

		// Calculate TTL from now until event stop time
		ttl := time.Until(eventStopTime)
		if ttl > 0 {
			cache.MarkNotificationSent(eventID, ttl)
		}
		delete(q.retries, eventID)
	}

	q.items = failures
}
