package service

import (
	"context"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *Service) InsertComment(ctx context.Context, comment *models.PostCommentDao, opts *options.InsertOneOptions) (*primitive.ObjectID, error) {
	id, err := s.Database.PostCommentsCollection.InsertOne(ctx, comment, opts)
	if err != nil {
		return nil, err
	}

	return id.InsertedID.(*primitive.ObjectID), nil
}

func (s *Service) UpdateComment(ctx context.Context, filter bson.M, update bson.M) error {
	_, err := s.Database.PostCommentsCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) UpdateComments(ctx context.Context, filter bson.M, update bson.M) error {
	_, err := s.Database.PostCommentsCollection.UpdateMany(ctx, filter, update)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) RemoveComment(ctx context.Context, filter bson.M) error {
	_, err := s.Database.PostCommentsCollection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) RemoveComments(ctx context.Context, filter bson.M) error {
	_, err := s.Database.PostCommentsCollection.DeleteMany(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}
