package service

import (
	"context"
	"errors"
	"olympsis-server/database"

	"github.com/gorilla/mux"
	"github.com/olympsis/models"
	"github.com/olympsis/notif"
	"github.com/olympsis/search"
	"github.com/sirupsen/logrus"
)

type Service struct {
	Database      *database.Database
	Logger        *logrus.Logger
	Router        *mux.Router
	SearchService *search.Service
	NotifService  *notif.Service
}

// Insert new post into database
func (s *Service) InsertAPost(ctx context.Context, event *models.Post) error {
	pong := s.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}
	s.Database.PostCol.InsertOne(ctx, event)
	return nil
}

// Get a post from database
func (s *Service) FindAPost(ctx context.Context, filter interface{}, event *models.Post) error {
	pong := s.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}
	s.Database.PostCol.FindOne(ctx, filter).Decode(&event)
	return nil
}

// get posts from database
func (s *Service) FindPosts(ctx context.Context, filter interface{}, posts *[]models.Post) error {
	pong := s.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}

	cursor, err := s.Database.PostCol.Find(ctx, filter)
	if err != nil {
		return err
	}

	for cursor.Next(context.TODO()) {
		var event models.Post
		err := cursor.Decode(&event)
		if err != nil {
			return err
		}
		*posts = append(*posts, event)
	}
	return nil
}

// update a post in database
func (s *Service) UpdateAPost(ctx context.Context, filter interface{}, update interface{}, event *models.Post) error {
	pong := s.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}

	// update user
	_, err := s.Database.PostCol.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	// find and return updated user
	err = s.FindAPost(ctx, filter, event)
	if err != nil {
		return err
	}

	return nil
}

// update posts in database
func (s *Service) UpdatePosts(ctx context.Context, filter interface{}, update interface{}, events *[]models.Post) error {
	pong := s.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}

	// update event
	_, err := s.Database.PostCol.UpdateMany(ctx, filter, update)
	if err != nil {
		return err
	}

	// find updated users
	err = s.FindPosts(ctx, filter, events)
	if err != nil {
		return err
	}

	return nil
}

// delete a post in database
func (s *Service) DeleteAPost(ctx context.Context, filter interface{}) error {
	pong := s.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}

	// delete user
	_, err := s.Database.PostCol.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}

// delete posts in database
func (s *Service) DeletePosts(ctx context.Context, filter interface{}) error {
	pong := s.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}

	// delete users
	_, err := s.Database.PostCol.DeleteMany(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}
