package service

import (
	"context"
	"olympsis-server/database"
	"olympsis-server/notifications"
	"olympsis-server/redis"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type EventPollingService struct {
	db     *database.Database
	logger *logrus.Logger
	cache  *redis.RedisDatabase
	sender *notifications.NotificationProcess
}

// Stripped down event object to reduce memory footprint
type StrippedEvent struct {
	ID       string             `bson:"_id"`
	StopTime primitive.DateTime `bson:"stop_time"`
}

func NewEventPollingService(d *database.Database, l *logrus.Logger, c *redis.RedisDatabase, s *notifications.NotificationProcess) *EventPollingService {
	return &EventPollingService{
		db:     d,
		cache:  c,
		sender: s,
		logger: l,
	}
}

func (p *EventPollingService) Start(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.processUpcomingEvents()
		}
	}
}

func (p *EventPollingService) getEvents(start time.Time, end time.Time) []StrippedEvent {
	projection := bson.M{"_id": 1, "stop_time": 1}
	options := options.Find().SetProjection(projection)
	filter := bson.M{
		"start_time": bson.M{
			"$gte": primitive.NewDateTimeFromTime(start),
			"$let": primitive.NewDateTimeFromTime(end),
		},
	}
	cursor, err := p.db.EventsCollection.Find(context.Background(), filter, options)
	if err != nil {
		p.logger.Errorf("Error fetching events: %v", err)
		return []StrippedEvent{}
	}

	// Decode events
	var events []StrippedEvent
	for cursor.Next(context.TODO()) {
		var event StrippedEvent
		err := cursor.Decode(&event)
		if err != nil {
			p.logger.Errorf("Failed to decode event ID. Error: %s", err.Error())
			continue
		}
		events = append(events, event)
	}

	return events
}

func (p *EventPollingService) processUpcomingEvents() {
	p.logger.Info("Starting Event Polling Reminder Processing...")

	start := time.Now().Add(25 * time.Minute)
	end := time.Now().Add(35 * time.Minute)

	// Fetch events starting in the next 30 mins or so...
	events := p.getEvents(start, end)

	// Group events by stop time for efficient queue processing
	eventsByStopTime := make(map[time.Time][]string)
	for _, event := range events {
		sent, err := p.cache.IsNotificationSent(event.ID)
		if err != nil {
			p.logger.Errorf("Error checking cache for event %s: %v", event.ID, err)
			continue
		}

		if !sent {
			eventsByStopTime[event.StopTime.Time()] = append(eventsByStopTime[event.StopTime.Time()], event.ID)
		}
	}

	// Process each group with the same TTL
	for stopTime, eventIDs := range eventsByStopTime {
		queue := NewEventNotificationQueue(p.logger)
		for _, eventID := range eventIDs {
			queue.Add(eventID)
		}
		queue.ProcessWithRetry(p.sender, p.cache, stopTime)
	}

	p.logger.Info("Stopping Event Polling Reminder Processing...")
}
