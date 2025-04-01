package service

import (
	"context"
	"fmt"
	"olympsis-server/utils"
	"time"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Helper function to generate the document containing the changes for an event dao
func buildUpdateChanges(req *models.EventDao) bson.M {
	changes := bson.M{}

	// Basic event details
	if req.PosterID != nil {
		changes["poster_id"] = req.PosterID
	}
	if req.Organizers != nil {
		changes["organizers"] = req.Organizers
	}
	if req.Venues != nil {
		changes["venues"] = req.Venues
	}

	// Media details
	if req.MediaURL != "" {
		changes["media_url"] = req.MediaURL
	}
	if req.MediaType != "" {
		changes["media_type"] = req.MediaType
	}

	// Content details
	if req.Title != nil {
		changes["title"] = req.Title
	}
	if req.Body != nil {
		changes["body"] = req.Body
	}
	if req.Sports != nil {
		changes["sports"] = req.Sports
	}

	// Format configuration
	if req.FormatConfig != nil {
		changes["format_config"] = req.FormatConfig
	}

	// Time details
	if req.StartTime != nil {
		changes["start_time"] = req.StartTime
	}
	if req.StopTime != nil {
		changes["stop_time"] = req.StopTime
	}

	// Participants configuration
	if req.ParticipantsConfig != nil {
		changes["participants_config"] = req.ParticipantsConfig
	}

	// Teams configuration
	if req.TeamsConfig != nil {
		changes["teams_config"] = req.TeamsConfig
	}

	// Visibility and access details
	if req.Visibility != nil {
		changes["visibility"] = req.Visibility
	}
	if req.ExternalLink != nil {
		changes["external_link"] = req.ExternalLink
	}
	if req.IsSensitive != nil {
		changes["is_sensitive"] = req.IsSensitive
	}

	// Status timestamps
	if req.UpdatedAt != nil {
		changes["updated_at"] = req.UpdatedAt
	}
	if req.CancelledAt != nil {
		changes["cancelled_at"] = req.CancelledAt
	}

	// Recurrence configuration
	if req.RecurrenceConfig != nil {
		changes["recurrence_config"] = req.RecurrenceConfig
	}

	return bson.M{"$set": changes}
}

// Helper function to generate updates for recurring events
func buildRecurringUpdateFilter(id primitive.ObjectID, event *models.EventDao, currentTime primitive.DateTime) bson.M {
	if event.RecurrenceConfig.ParentEventID != nil {
		// This is a child event, update all future siblings
		return bson.M{
			"$or": []bson.M{
				{
					"_id":        event.RecurrenceConfig.ParentEventID,
					"start_time": bson.M{"$gte": currentTime},
				},
				{
					"parent_event_id": event.RecurrenceConfig.ParentEventID,
					"start_time":      bson.M{"$gte": currentTime},
				},
			},
		}
	}

	// This is a parent event
	return bson.M{
		"$or": []bson.M{
			{
				"_id":        id,
				"start_time": bson.M{"$gte": currentTime},
			},
			{
				"parent_event_id": id,
				"start_time":      bson.M{"$gte": currentTime},
			},
		},
	}
}

// Helper function to generate recurring event instances
func GenerateEventInstancesBatched(parentID primitive.ObjectID, baseEvent *models.EventDao, recurrence *models.RecurrenceOptions) []models.EventDao {
	var instances []models.EventDao

	// Get start time as Go time.Time
	if baseEvent.StartTime == nil || baseEvent.StopTime == nil {
		return instances // Cannot proceed without start/stop times
	}

	startTime := baseEvent.StartTime.Time()
	endTime := baseEvent.StopTime.Time()
	eventDuration := endTime.Sub(startTime)

	// Get recurrence end time
	recurrenceEndTime := recurrence.EndTime.Time()

	// Calculate next occurrence based on pattern and interval
	var nextTime time.Time
	switch recurrence.Pattern {
	case "DAILY":
		nextTime = startTime.AddDate(0, 0, recurrence.Interval)
	case "WEEKLY":
		nextTime = startTime.AddDate(0, 0, 7*recurrence.Interval)
	case "MONTHLY":
		nextTime = startTime.AddDate(0, recurrence.Interval, 0)
	default:
		return instances // Invalid pattern
	}

	// Add safety limit to prevent infinite loops
	maxInstances := 365 // Maximum one year of events
	instanceCount := 0

	// Create instances until we reach the end time or hit the safety limit
	for nextTime.Before(recurrenceEndTime) && instanceCount < maxInstances {
		// Create a new instance by copying the base event
		instance := *baseEvent // Copy all fields from parent

		// Set new times for this instance
		instanceStartTime := primitive.NewDateTimeFromTime(nextTime)
		instanceStopTime := primitive.NewDateTimeFromTime(nextTime.Add(eventDuration))
		instance.StartTime = &instanceStartTime
		instance.StopTime = &instanceStopTime

		// Set up recurrence config pointing to parent event
		instance.RecurrenceConfig = &models.EventRecurrenceConfig{
			RecurrenceRule: &recurrence.Pattern,
			ParentEventID:  &parentID,
		}

		// Add instance to the list
		instances = append(instances, instance)
		instanceCount++

		// Calculate the next occurrence
		switch recurrence.Pattern {
		case "DAILY":
			nextTime = nextTime.AddDate(0, 0, recurrence.Interval)
		case "WEEKLY":
			nextTime = nextTime.AddDate(0, 0, 7*recurrence.Interval)
		case "MONTHLY":
			nextTime = nextTime.AddDate(0, recurrence.Interval, 0)
		}
	}

	return instances
}

// Helper function to send notifications to an event's organizers
func notifyOrganizers(organizers []models.Organizer, note *models.PushNotification, token string, service *utils.NotificationInterface) {
	for _, v := range organizers {
		ID := v.ID.Hex()
		err := service.SendNotification(token, models.NotificationPushRequest{
			Topic:        &ID,
			Notification: *note,
		})

		if err != nil {
			service.Logger.Errorf("Failed to send notification. Error: %s", err.Error())
		}
	}
}

func GenerateNewEventNotification(id string, title string) models.PushNotification {
	return models.PushNotification{
		Title:    "New Event Created!",
		Body:     title,
		Type:     "push",
		Category: "events",
		Data: map[string]interface{}{
			"type": "new_event",
			"id":   id,
		},
	}
}

func generateNewParticipantNotification(id string, title string, status string) models.PushNotification {
	return models.PushNotification{
		Title:    title,
		Body:     fmt.Sprintf("New Participant RSVP'ed %s", status),
		Type:     "push",
		Category: "events",
		Data: map[string]interface{}{
			"type": "event_update",
			"id":   id,
		},
	}
}

// Find nearby venues based on location, sports, and radius
func (s *Service) FindNearbyVenues(ctx context.Context, location models.GeoJSON, radius float64) (*[]models.Venue, []primitive.ObjectID, error) {
	// Convert radius from miles to meters (1 mile = 1609.34 meters)
	radiusInMeters := radius * 1609.34

	// Create filter for geospatial query
	filter := bson.M{
		"location": bson.M{
			"$near": bson.M{
				"$geometry":    location,
				"$maxDistance": radiusInMeters,
			},
		},
	}

	// Rest of the function remains the same
	cursor, err := s.Database.VenuesCollection.Find(ctx, filter)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Return empty results rather than error
			return &[]models.Venue{}, []primitive.ObjectID{}, nil
		}
		return nil, nil, err
	}
	defer cursor.Close(ctx)

	// Process results
	venues := []models.Venue{}
	venueIDs := []primitive.ObjectID{}

	for cursor.Next(ctx) {
		var venue models.Venue
		if err := cursor.Decode(&venue); err != nil {
			s.Logger.Warning("Failed to decode venue: ", err.Error())
			continue
		}
		venues = append(venues, venue)
		venueIDs = append(venueIDs, venue.ID)
	}

	if err := cursor.Err(); err != nil {
		return nil, nil, err
	}

	return &venues, venueIDs, nil
}
