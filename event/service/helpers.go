package service

import (
	"context"
	"time"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
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
	if req.MediaURL != nil && *req.MediaURL != "" {
		changes["media_url"] = req.MediaURL
	}
	if req.MediaType != nil && *req.MediaType != "" {
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
	if req.ExternalLinks != nil {
		changes["external_links"] = req.ExternalLinks
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

// extractAuditChanges takes the output of buildUpdateChanges and the current event state,
// and returns the fields changed, old values, and new values for use in an EventAuditLog.
// The changes param is the bson.M{"$set": bson.M{...}} returned by buildUpdateChanges.
func extractAuditChanges(changes bson.M, currentEvent *models.EventDao) ([]string, map[string]any, map[string]any) {
	setMap, ok := changes["$set"].(bson.M)
	if !ok {
		return nil, nil, nil
	}

	fieldsChanged := make([]string, 0, len(setMap))
	oldValues := make(map[string]any, len(setMap))
	newValues := make(map[string]any, len(setMap))

	// Build a lookup of bson field names to current values on the existing event
	currentFields := eventFieldValues(currentEvent)

	for field, newVal := range setMap {
		fieldsChanged = append(fieldsChanged, field)
		newValues[field] = newVal
		if oldVal, exists := currentFields[field]; exists {
			oldValues[field] = oldVal
		}
	}

	return fieldsChanged, oldValues, newValues
}

// eventFieldValues maps an EventDao's bson field names to their current values.
// Only non-nil fields are included so callers can distinguish "unset" from "set".
func eventFieldValues(e *models.EventDao) map[string]any {
	m := make(map[string]any)
	if e == nil {
		return m
	}
	if e.PosterID != nil {
		m["poster_id"] = *e.PosterID
	}
	if e.Organizers != nil {
		m["organizers"] = *e.Organizers
	}
	if e.Venues != nil {
		m["venues"] = *e.Venues
	}
	if e.MediaURL != nil {
		m["media_url"] = *e.MediaURL
	}
	if e.MediaType != nil {
		m["media_type"] = *e.MediaType
	}
	if e.Title != nil {
		m["title"] = *e.Title
	}
	if e.Body != nil {
		m["body"] = *e.Body
	}
	if e.Sports != nil {
		m["sports"] = *e.Sports
	}
	if e.FormatConfig != nil {
		m["format_config"] = e.FormatConfig
	}
	if e.StartTime != nil {
		m["start_time"] = e.StartTime
	}
	if e.StopTime != nil {
		m["stop_time"] = e.StopTime
	}
	if e.ParticipantsConfig != nil {
		m["participants_config"] = e.ParticipantsConfig
	}
	if e.TeamsConfig != nil {
		m["teams_config"] = e.TeamsConfig
	}
	if e.Visibility != nil {
		m["visibility"] = *e.Visibility
	}
	if e.ExternalLinks != nil {
		m["external_links"] = *e.ExternalLinks
	}
	if e.IsSensitive != nil {
		m["is_sensitive"] = *e.IsSensitive
	}
	if e.UpdatedAt != nil {
		m["updated_at"] = e.UpdatedAt
	}
	if e.CancelledAt != nil {
		m["cancelled_at"] = e.CancelledAt
	}
	if e.RecurrenceConfig != nil {
		m["recurrence_config"] = e.RecurrenceConfig
	}
	return m
}

// Helper function to generate updates for recurring events
func buildRecurringUpdateFilter(id bson.ObjectID, event *models.EventDao, currentTime bson.DateTime) bson.M {
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
func GenerateEventInstancesBatched(parentID bson.ObjectID, baseEvent *models.EventDao, recurrence *models.RecurrenceOptions) []models.EventDao {
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
		instanceStartTime := bson.NewDateTimeFromTime(nextTime)
		instanceStopTime := bson.NewDateTimeFromTime(nextTime.Add(eventDuration))
		instance.StartTime = &instanceStartTime
		instance.StopTime = &instanceStopTime

		// Set up recurrence config pointing to parent event
		pattern := string(recurrence.Pattern)
		instance.RecurrenceConfig = &models.EventRecurrenceConfig{
			RecurrenceRule: &pattern,
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

// Find nearby venues based on location, sports, and radius
func (s *Service) FindNearbyVenues(ctx context.Context, location models.GeoJSON, radius float64) (*[]models.Venue, []bson.ObjectID, error) {
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
			return &[]models.Venue{}, []bson.ObjectID{}, nil
		}
		return nil, nil, err
	}
	defer cursor.Close(ctx)

	// Process results
	venues := []models.Venue{}
	venueIDs := []bson.ObjectID{}

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
