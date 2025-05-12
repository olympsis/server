package service

import (
	"context"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *Service) InsertReaction(ctx context.Context, reaction *models.ReactionDao, opts *options.InsertOneOptions) (*primitive.ObjectID, error) {
	id, err := s.Database.PostReactionsCollection.InsertOne(ctx, reaction, opts)
	if err != nil {
		return nil, err
	}

	return id.InsertedID.(*primitive.ObjectID), nil
}

func (s *Service) UpdateReaction(ctx context.Context, filter bson.M, update bson.M) error {
	_, err := s.Database.PostReactionsCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) UpdateReactions(ctx context.Context, filter bson.M, update bson.M) error {
	_, err := s.Database.PostReactionsCollection.UpdateMany(ctx, filter, update)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) RemoveReaction(ctx context.Context, filter bson.M) error {
	_, err := s.Database.PostReactionsCollection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) RemoveReactions(ctx context.Context, filter bson.M) error {
	_, err := s.Database.PostReactionsCollection.DeleteMany(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}
