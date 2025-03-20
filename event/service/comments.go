package service

import (
	"context"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Comment Functions

// Insert new comment into database
func (s *Service) InsertComment(ctx context.Context, comment *models.EventCommentDao) (primitive.ObjectID, error) {
	resp, err := s.Database.EventCommentsCollection.InsertOne(ctx, comment)
	if err != nil {
		return primitive.NilObjectID, err
	}
	id := resp.InsertedID.(primitive.ObjectID)
	return id, nil
}

// Insert multiple comments into database
func (s *Service) InsertComments(ctx context.Context, comments []interface{}) ([]primitive.ObjectID, error) {
	resp, err := s.Database.EventCommentsCollection.InsertMany(ctx, comments)
	if err != nil {
		return nil, err
	}

	// Convert inserted IDs to ObjectIDs
	ids := make([]primitive.ObjectID, len(resp.InsertedIDs))
	for i, id := range resp.InsertedIDs {
		ids[i] = id.(primitive.ObjectID)
	}

	return ids, nil
}

// Get comment from database
func (s *Service) FindComment(ctx context.Context, filter interface{}) (*models.EventCommentDao, error) {
	var comment models.EventCommentDao
	err := s.Database.EventCommentsCollection.FindOne(ctx, filter).Decode(&comment)
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

// Get comments from database
func (s *Service) FindComments(ctx context.Context, filter interface{}, limit int64, skip int64) ([]models.EventCommentDao, error) {
	opts := options.Find()
	if limit > 0 {
		opts.SetLimit(limit)
	}
	if skip > 0 {
		opts.SetSkip(skip)
	}
	// Sort by creation time, newest first
	opts.SetSort(bson.D{{Key: "created_at", Value: -1}})

	var comments []models.EventCommentDao
	cursor, err := s.Database.EventCommentsCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &comments); err != nil {
		return nil, err
	}
	return comments, nil
}

// Get all comments for an event
func (s *Service) FindEventComments(ctx context.Context, eventID primitive.ObjectID, limit int64, skip int64) ([]models.EventCommentDao, error) {
	filter := bson.M{"event_id": eventID}
	return s.FindComments(ctx, filter, limit, skip)
}

// Count comments for an event
func (s *Service) CountEventComments(ctx context.Context, eventID primitive.ObjectID) (int64, error) {
	filter := bson.M{"event_id": eventID}
	count, err := s.Database.EventCommentsCollection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// Get comments by a specific user
func (s *Service) FindUserComments(ctx context.Context, userUUID string, limit int64, skip int64) ([]models.EventCommentDao, error) {
	filter := bson.M{"uuid": userUUID}
	return s.FindComments(ctx, filter, limit, skip)
}

// Update comment in database
func (s *Service) UpdateComment(ctx context.Context, filter interface{}, update interface{}) error {
	_, err := s.Database.EventCommentsCollection.UpdateOne(ctx, filter, update)
	return err
}

// Delete comment from database
func (s *Service) DeleteComment(ctx context.Context, filter interface{}) error {
	_, err := s.Database.EventCommentsCollection.DeleteOne(ctx, filter)
	return err
}

// Delete all comments for an event
func (s *Service) DeleteEventComments(ctx context.Context, eventID primitive.ObjectID) (int64, error) {
	filter := bson.M{"event_id": eventID}
	result, err := s.Database.EventCommentsCollection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, err
	}
	return result.DeletedCount, nil
}
