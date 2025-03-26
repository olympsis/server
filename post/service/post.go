package service

import (
	"context"
	"olympsis-server/database"
	"olympsis-server/utils"

	"github.com/gorilla/mux"
	"github.com/olympsis/models"
	"github.com/olympsis/search"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Service struct {
	Database      *database.Database
	Logger        *logrus.Logger
	Router        *mux.Router
	SearchService *search.Service
	Notification  *utils.NotificationInterface
}

// Insert one post into database
func (s *Service) InsertPost(ctx context.Context, post *models.PostDao, opts *options.InsertOneOptions) (*primitive.ObjectID, error) {
	id, err := s.Database.PostCol.InsertOne(ctx, post, opts)
	if err != nil {
		return nil, err
	}

	return id.InsertedID.(*primitive.ObjectID), nil
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
