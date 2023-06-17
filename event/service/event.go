package service

import (
	"context"
	"errors"

	"github.com/olympsis/models"
)

// Insert new event into database
func (s *Service) InsertEvent(ctx context.Context, event *models.Event) error {
	pong := s.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}
	s.Database.EventCol.InsertOne(ctx, event)
	return nil
}

// Get event from database
func (s *Service) FindEvent(ctx context.Context, filter interface{}, event *models.Event) error {
	pong := s.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}
	s.Database.EventCol.FindOne(ctx, filter).Decode(&event)
	return nil
}

// get events from database
func (s *Service) FindEvents(ctx context.Context, filter interface{}, events *[]models.Event) error {
	pong := s.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}

	cursor, err := s.Database.EventCol.Find(ctx, filter)
	if err != nil {
		return err
	}

	for cursor.Next(context.TODO()) {
		var event models.Event
		err := cursor.Decode(&event)
		if err != nil {
			return err
		}
		*events = append(*events, event)
	}
	return nil
}

// update user in database
func (s *Service) UpdateEvent(ctx context.Context, filter interface{}, update interface{}, event *models.Event) error {
	pong := s.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}

	// update user
	_, err := s.Database.EventCol.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	// find and return updated user
	err = s.FindEvent(ctx, filter, event)
	if err != nil {
		return err
	}

	return nil
}

// update event in database
func (s *Service) UpdateEvents(ctx context.Context, filter interface{}, update interface{}, events *[]models.Event) error {
	pong := s.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}

	// update event
	_, err := s.Database.EventCol.UpdateMany(ctx, filter, update)
	if err != nil {
		return err
	}

	// find updated users
	err = s.FindEvents(ctx, filter, events)
	if err != nil {
		return err
	}

	return nil
}

// delete event in database
func (s *Service) DeleteEvent(ctx context.Context, filter interface{}) error {
	pong := s.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}

	// delete user
	_, err := s.Database.EventCol.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}

// delete users in database
func (s *Service) DeleteEvents(ctx context.Context, filter interface{}) error {
	pong := s.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}

	// delete users
	_, err := s.Database.EventCol.DeleteMany(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}
