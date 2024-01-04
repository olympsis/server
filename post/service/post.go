package service

import (
	"context"
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

// Insert one post into database
func (s *Service) InsertPost(ctx context.Context, event *models.Post) error {
	s.Database.PostCol.InsertOne(ctx, event)
	return nil
}

// Find one post from database
func (s *Service) FindPost(ctx context.Context, filter interface{}, event *models.Post) error {
	s.Database.PostCol.FindOne(ctx, filter).Decode(&event)
	return nil
}

// Find many posts from database
func (s *Service) FindPosts(ctx context.Context, filter interface{}, posts *[]models.Post) error {
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

// Update one post in the database
func (s *Service) UpdatePost(ctx context.Context, filter interface{}, update interface{}) error {
	// update user
	_, err := s.Database.PostCol.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	return nil
}

// Update many posts in the database
func (s *Service) UpdatePosts(ctx context.Context, filter interface{}, update interface{}) error {
	// update event
	_, err := s.Database.PostCol.UpdateMany(ctx, filter, update)
	if err != nil {
		return err
	}
	return nil
}

// Delete one post in the database
func (s *Service) RemovePost(ctx context.Context, filter interface{}) error {
	// delete user
	_, err := s.Database.PostCol.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}

// Delete many posts in the database
func (s *Service) RemovePosts(ctx context.Context, filter interface{}) error {
	// delete users
	_, err := s.Database.PostCol.DeleteMany(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}
