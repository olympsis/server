package service

import (
	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Helper function to generate the document containing the changes for an event dao
func buildUpdateChanges(req *models.EventDao) bson.M {
	changes := bson.M{}
	if req.Title != nil {
		changes["title"] = req.Title
	}
	if req.Body != nil {
		changes["body"] = req.Body
	}
	if req.ImageURL != nil {
		changes["image_url"] = req.ImageURL
	}
	if req.StartTime != nil {
		changes["start_time"] = req.StartTime
	}
	if req.StopTime != nil {
		changes["stop_time"] = req.StopTime
	}
	if req.MinParticipants != nil {
		changes["min_participants"] = req.MinParticipants
	}
	if req.MaxParticipants != nil {
		changes["max_participants"] = req.MaxParticipants
	}
	if req.Level != nil {
		changes["level"] = req.Level
	}
	if req.ExternalLink != nil {
		changes["external_link"] = req.ExternalLink
	}
	if req.Visibility != nil {
		changes["visibility"] = req.Visibility
	}

	return bson.M{"$set": changes}
}

// Helper function to generate updates for recurring events
func buildRecurringUpdateFilter(id primitive.ObjectID, event *models.EventDao, currentTime int64) bson.M {
	if event.ParentEventID != nil {
		// This is a child event, update all future siblings
		return bson.M{
			"$or": []bson.M{
				{
					"_id":        event.ParentEventID,
					"start_time": bson.M{"$gte": currentTime},
				},
				{
					"parent_event_id": event.ParentEventID,
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
func generateEventInstancesBatched(baseEvent *models.EventDao, recurrence *models.RecurrenceOptions) []*models.EventDao {
	var instances []*models.EventDao
	currentTime := *baseEvent.StartTime

	// Add safety limit to prevent infinite loops
	maxInstances := 365 // Maximum one year of daily events
	instanceCount := 0

	for currentTime <= recurrence.EndTime && instanceCount < maxInstances {
		if currentTime != *baseEvent.StartTime {
			instance := *baseEvent
			instance.StartTime = &currentTime

			if baseEvent.StopTime != nil {
				newStopTime := *baseEvent.StopTime + (currentTime - *baseEvent.StartTime)
				instance.StopTime = &newStopTime
			}

			instances = append(instances, &instance)
			instanceCount++
		}

		// Calculate next occurrence based on pattern
		switch *baseEvent.RecurrenceRule {
		case "DAILY":
			currentTime += int64(recurrence.Interval * 24 * 60 * 60)
		case "WEEKLY":
			currentTime += int64(recurrence.Interval * 7 * 24 * 60 * 60)
		case "MONTHLY":
			currentTime += int64(recurrence.Interval * 30 * 24 * 60 * 60)
		}
	}

	return instances
}
