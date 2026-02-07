package service

import (
	"context"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Participant Functions

// Insert new participant into database
func (s *Service) InsertParticipant(ctx context.Context, participant *models.ParticipantDao) (bson.ObjectID, error) {
	resp, err := s.Database.EventParticipantsCollection.InsertOne(ctx, participant)
	if err != nil {
		return bson.NilObjectID, err
	}
	id := resp.InsertedID.(bson.ObjectID)
	return id, nil
}

// Insert multiple participants into database
func (s *Service) InsertParticipants(ctx context.Context, participants []interface{}) ([]bson.ObjectID, error) {
	resp, err := s.Database.EventParticipantsCollection.InsertMany(ctx, participants)
	if err != nil {
		return nil, err
	}

	// Convert inserted IDs to ObjectIDs
	ids := make([]bson.ObjectID, len(resp.InsertedIDs))
	for i, id := range resp.InsertedIDs {
		ids[i] = id.(bson.ObjectID)
	}

	return ids, nil
}

// Get participant from database
func (s *Service) FindParticipant(ctx context.Context, filter bson.M) (*models.ParticipantDao, error) {
	var participant models.ParticipantDao
	err := s.Database.EventParticipantsCollection.FindOne(ctx, filter).Decode(&participant)
	if err != nil {
		return nil, err
	}
	return &participant, nil
}

// Get participants from database
func (s *Service) FindParticipants(ctx context.Context, filter bson.M, opts *options.FindOptionsBuilder) ([]models.ParticipantDao, error) {
	var participants []models.ParticipantDao
	cursor, err := s.Database.EventParticipantsCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &participants); err != nil {
		return nil, err
	}
	return participants, nil
}

// Update participant in database
func (s *Service) UpdateParticipant(ctx context.Context, filter bson.M, update bson.M) error {
	_, err := s.Database.EventParticipantsCollection.UpdateOne(ctx, filter, update)
	return err
}

// Update multiple participants in database
func (s *Service) UpdateParticipants(ctx context.Context, filter bson.M, update bson.M) error {
	_, err := s.Database.EventParticipantsCollection.UpdateMany(ctx, filter, update)
	return err
}

// Delete participant from database
func (s *Service) DeleteParticipant(ctx context.Context, filter bson.M) error {
	_, err := s.Database.EventParticipantsCollection.DeleteOne(ctx, filter)
	return err
}

// Delete multiple participants from database
func (s *Service) DeleteParticipants(ctx context.Context, filter bson.M) error {
	_, err := s.Database.EventParticipantsCollection.DeleteMany(ctx, filter)
	return err
}
