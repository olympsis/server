package service

import (
	"fmt"
	"olympsis-server/utils"
	"time"

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
func generateEventInstancesBatched(baseEventID *primitive.ObjectID, baseEvent *models.EventDao, recurrence *models.RecurrenceOptions) []*models.EventDao {
	var instances []*models.EventDao
	currentStartTime := *baseEvent.StartTime

	// Calculate the original duration between start and stop time
	var eventDuration int64
	if baseEvent.StopTime != nil {
		eventDuration = *baseEvent.StopTime - *baseEvent.StartTime
	}

	// Add safety limit to prevent infinite loops
	maxInstances := 365 // Maximum one year of daily events
	instanceCount := 0

	for currentStartTime <= recurrence.EndTime && instanceCount < maxInstances {
		// Skip the first occurrence as it's the parent event
		if currentStartTime != *baseEvent.StartTime {
			// Create a copy of the base event
			instance := &models.EventDao{} // Create new instance
			*instance = *baseEvent         // Copy all fields

			// Create new time value for this instance
			startTimeCopy := currentStartTime
			instance.StartTime = &startTimeCopy
			instance.ParentEventID = baseEventID

			// Calculate the new stop time by adding the original duration
			if baseEvent.StopTime != nil {
				newStopTime := currentStartTime + eventDuration
				instance.StopTime = &newStopTime
			}

			instances = append(instances, instance)
			instanceCount++
		}

		// Calculate next occurrence based on pattern
		switch *baseEvent.RecurrenceRule {
		case "DAILY":
			// Convert to Time object for more accurate day calculations
			currentTime := time.Unix(currentStartTime, 0)
			// Add the specified number of days
			newTime := currentTime.AddDate(0, 0, recurrence.Interval)
			// Convert back to Unix timestamp
			currentStartTime = newTime.Unix()
		case "WEEKLY":
			// Convert to Time object for more accurate week calculations
			currentTime := time.Unix(currentStartTime, 0)
			// Add the specified number of weeks (7 days per week)
			newTime := currentTime.AddDate(0, 0, 7*recurrence.Interval)
			// Convert back to Unix timestamp
			currentStartTime = newTime.Unix()
		case "MONTHLY":
			// Get the current time as a Time object for more accurate month calculations
			currentTime := time.Unix(currentStartTime, 0)
			// Add the specified number of months
			newTime := currentTime.AddDate(0, recurrence.Interval, 0)
			// Convert back to Unix timestamp
			currentStartTime = newTime.Unix()
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

func generateNewEventNotification(id string, title string) models.PushNotification {
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
