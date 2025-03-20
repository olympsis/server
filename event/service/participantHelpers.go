package service

import (
	"context"
	"fmt"
	"time"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Helper function to generate the document containing the changes for a participant dao
func BuildParticipantUpdateChanges(req *models.ParticipantDao) bson.M {
	changes := bson.M{}

	if req.UserID != nil {
		changes["user_id"] = req.UserID
	}
	if req.Status != nil {
		changes["status"] = req.Status
	}
	if req.EventID != nil {
		changes["event_id"] = req.EventID
	}
	if req.CreatedAt != nil {
		changes["created_at"] = req.CreatedAt
	}

	return bson.M{"$set": changes}
}

// Move a participant from waitlist to active participants
func (s *Service) MoveWaitlistParticipantToEvent(ctx context.Context, eventID primitive.ObjectID, participant models.ParticipantDao) error {
	// Find the participant in the waitlist
	filter := bson.M{"_id": participant.ID, "event_id": eventID}
	var waitlistedParticipant models.ParticipantDao
	err := s.Database.EventParticipantsWaitlistCollection.FindOne(ctx, filter).Decode(&waitlistedParticipant)
	if err != nil {
		return fmt.Errorf("failed to find participant in waitlist: %v", err)
	}

	// Create a new active participant entry
	activeParticipant := waitlistedParticipant

	// Insert participant into active participants collection
	_, err = s.Database.EventParticipantsCollection.InsertOne(ctx, activeParticipant)
	if err != nil {
		return fmt.Errorf("failed to add participant to event: %v", err)
	}

	// Remove from waitlist
	_, err = s.Database.EventParticipantsWaitlistCollection.DeleteOne(ctx, filter)
	if err != nil {
		// If this fails, we should still continue and update the event
		s.Logger.Warnf("Failed to remove participant %s from waitlist: %v", participant.ID.Hex(), err)
	}

	return nil
}

// Move all eligible waitlisted participants to an event (used when spots open up)
func (s *Service) PromoteFromWaitlist(ctx context.Context, eventID primitive.ObjectID, spotsAvailable int) (int, error) {
	if spotsAvailable <= 0 {
		return 0, nil // No spots available
	}

	// Get the waitlisted participants in order (usually by created_at)
	filter := bson.M{"event_id": eventID}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}).SetLimit(int64(spotsAvailable))

	cursor, err := s.Database.EventParticipantsWaitlistCollection.Find(ctx, filter, opts)
	if err != nil {
		return 0, fmt.Errorf("failed to find waitlisted participants: %v", err)
	}
	defer cursor.Close(ctx)

	var waitlistedParticipants []models.ParticipantDao
	if err := cursor.All(ctx, &waitlistedParticipants); err != nil {
		return 0, fmt.Errorf("failed to decode waitlisted participants: %v", err)
	}

	promotedCount := 0
	for _, participant := range waitlistedParticipants {
		err := s.MoveWaitlistParticipantToEvent(ctx, eventID, participant)
		if err != nil {
			s.Logger.Warnf("Failed to promote participant %s: %v", participant.ID.Hex(), err)
			continue
		}
		promotedCount++
	}

	return promotedCount, nil
}

// ChangeWaitlistParticipantRank allows modifying a participant's position in the waitlist
// positionType can be: "first", "last", "up", "down", or "position"
// positionValue is only used when positionType is "position" (1-based index, where 1 is the first position)
func (s *Service) ChangeWaitlistParticipantRank(ctx context.Context, eventID primitive.ObjectID, participantID primitive.ObjectID, positionType string, positionValue int) error {
	// First, get all waitlisted participants for this event, ordered by created_at
	filter := bson.M{"event_id": eventID}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}})

	cursor, err := s.Database.EventParticipantsWaitlistCollection.Find(ctx, filter, opts)
	if err != nil {
		return fmt.Errorf("failed to retrieve waitlist: %v", err)
	}
	defer cursor.Close(ctx)

	var waitlistedParticipants []models.Participant
	if err := cursor.All(ctx, &waitlistedParticipants); err != nil {
		return fmt.Errorf("failed to decode waitlisted participants: %v", err)
	}

	// Find the participant's current position
	currentPosition := -1
	for i, p := range waitlistedParticipants {
		if p.ID == participantID {
			currentPosition = i
			break
		}
	}

	if currentPosition == -1 {
		return fmt.Errorf("participant %s not found in the waitlist", participantID.Hex())
	}

	// Calculate the new timestamp based on position type
	var newTimestamp primitive.DateTime
	totalParticipants := len(waitlistedParticipants)

	switch positionType {
	case "first":
		// Make participant first in the list by setting their timestamp before the current first
		if currentPosition == 0 {
			return nil // Already at the front
		}
		firstTimestamp := waitlistedParticipants[0].CreatedAt
		newTimestamp = primitive.NewDateTimeFromTime(firstTimestamp.Time().Add(-time.Minute))

	case "last":
		// Make participant last in the list by setting their timestamp after the current last
		if currentPosition == totalParticipants-1 {
			return nil // Already at the end
		}
		lastTimestamp := waitlistedParticipants[totalParticipants-1].CreatedAt
		newTimestamp = primitive.NewDateTimeFromTime(lastTimestamp.Time().Add(time.Minute))

	case "up":
		// Move participant up one position (closer to the front)
		if currentPosition == 0 {
			return nil // Already at the front
		}

		// If moving to first position
		if currentPosition == 1 {
			firstTimestamp := waitlistedParticipants[0].CreatedAt
			newTimestamp = primitive.NewDateTimeFromTime(firstTimestamp.Time().Add(-time.Minute))
		} else {
			// Set timestamp between the two participants
			aboveTimestamp := waitlistedParticipants[currentPosition-2].CreatedAt
			belowTimestamp := waitlistedParticipants[currentPosition-1].CreatedAt
			midTime := aboveTimestamp.Time().Add(belowTimestamp.Time().Sub(aboveTimestamp.Time()) / 2)
			newTimestamp = primitive.NewDateTimeFromTime(midTime)
		}

	case "down":
		// Move participant down one position (further from the front)
		if currentPosition == totalParticipants-1 {
			return nil // Already at the end
		}

		// If moving to last position
		if currentPosition == totalParticipants-2 {
			lastTimestamp := waitlistedParticipants[totalParticipants-1].CreatedAt
			newTimestamp = primitive.NewDateTimeFromTime(lastTimestamp.Time().Add(time.Minute))
		} else {
			// Set timestamp between the two participants
			aboveTimestamp := waitlistedParticipants[currentPosition+1].CreatedAt
			belowTimestamp := waitlistedParticipants[currentPosition+2].CreatedAt
			midTime := aboveTimestamp.Time().Add(belowTimestamp.Time().Sub(aboveTimestamp.Time()) / 2)
			newTimestamp = primitive.NewDateTimeFromTime(midTime)
		}

	case "position":
		// Move to a specific position (1-based index)
		if positionValue < 1 || positionValue > totalParticipants {
			return fmt.Errorf("invalid position value: must be between 1 and %d", totalParticipants)
		}

		targetPosition := positionValue - 1 // Convert to 0-based index

		if targetPosition == currentPosition {
			return nil // Already at the requested position
		}

		if targetPosition == 0 {
			// Moving to first position
			firstTimestamp := waitlistedParticipants[0].CreatedAt
			newTimestamp = primitive.NewDateTimeFromTime(firstTimestamp.Time().Add(-time.Minute))
		} else if targetPosition == totalParticipants-1 {
			// Moving to last position
			lastTimestamp := waitlistedParticipants[totalParticipants-1].CreatedAt
			newTimestamp = primitive.NewDateTimeFromTime(lastTimestamp.Time().Add(time.Minute))
		} else {
			// Moving to a position in the middle
			// Set timestamp between the two participants at the target position
			aboveTimestamp := waitlistedParticipants[targetPosition-1].CreatedAt
			belowTimestamp := waitlistedParticipants[targetPosition].CreatedAt
			midTime := aboveTimestamp.Time().Add(belowTimestamp.Time().Sub(aboveTimestamp.Time()) / 2)
			newTimestamp = primitive.NewDateTimeFromTime(midTime)
		}

	default:
		return fmt.Errorf("invalid position type: must be 'first', 'last', 'up', 'down', or 'position'")
	}

	// Update the participant's created_at field to change their position
	updateFilter := bson.M{"_id": participantID, "event_id": eventID}
	update := bson.M{"$set": bson.M{"created_at": newTimestamp}}

	result, err := s.Database.EventParticipantsWaitlistCollection.UpdateOne(ctx, updateFilter, update)
	if err != nil {
		return fmt.Errorf("failed to update participant position: %v", err)
	}

	if result.ModifiedCount == 0 {
		return fmt.Errorf("participant not found or position not changed")
	}

	// Log the successful position change
	s.Logger.Infof("Changed participant %s position in event %s waitlist to %s",
		participantID.Hex(), eventID.Hex(), positionType)

	return nil
}
