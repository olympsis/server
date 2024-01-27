package service

import (
	"context"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Insert new event into database
func (s *Service) InsertEvent(ctx context.Context, event *models.EventDao) (*primitive.ObjectID, error) {
	resp, err := s.Database.EventCol.InsertOne(ctx, event)
	if err != nil {
		return nil, err
	}
	id := resp.InsertedID.(primitive.ObjectID)
	return &id, err
}

// Get event from database
func (s *Service) FindEvent(ctx context.Context, filter interface{}) (*models.EventDao, error) {
	var event models.EventDao
	err := s.Database.EventCol.FindOne(ctx, filter).Decode(&event)
	if err != nil {
		return nil, err
	}
	return &event, nil
}

// get events from database
func (s *Service) FindEvents(ctx context.Context, filter interface{}) (*[]models.EventDao, error) {
	var events []models.EventDao
	cursor, err := s.Database.EventCol.Find(ctx, filter)
	if err != nil {
		return nil, err
	}

	for cursor.Next(context.TODO()) {
		var event models.EventDao
		err := cursor.Decode(&event)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return &events, nil
}

// update user in database
func (s *Service) UpdateEvent(ctx context.Context, filter interface{}, update interface{}) error {

	// update user
	_, err := s.Database.EventCol.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	return nil
}

// update event in database
func (s *Service) UpdateEvents(ctx context.Context, filter interface{}, update interface{}) error {

	// update event
	_, err := s.Database.EventCol.UpdateMany(ctx, filter, update)
	if err != nil {
		return err
	}

	return nil
}

// delete event in database
func (s *Service) DeleteEvent(ctx context.Context, filter interface{}) error {

	// delete user
	_, err := s.Database.EventCol.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}

// delete users in database
func (s *Service) DeleteEvents(ctx context.Context, filter interface{}) error {

	// delete users
	_, err := s.Database.EventCol.DeleteMany(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}
