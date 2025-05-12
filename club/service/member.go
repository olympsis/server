package service

import (
	"context"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (s *Service) InsertMember(ctx context.Context, member *models.MemberDao) (*primitive.ObjectID, error) {
	resp, err := s.Database.ClubMembersCollection.InsertOne(ctx, member)
	if err != nil {
		return nil, err
	}
	id := resp.InsertedID.(primitive.ObjectID)
	return &id, nil
}

func (s *Service) FindMember(ctx context.Context, filter bson.M) (*models.MemberDao, error) {
	var member models.MemberDao
	err := s.Database.ClubMembersCollection.FindOne(ctx, filter).Decode(&member)
	if err != nil {
		return nil, err
	}
	return &member, nil
}

func (s *Service) FindMembers(ctx context.Context, filter bson.M) (*[]models.MemberDao, error) {

	var members []models.MemberDao

	cursor, err := s.Database.ClubMembersCollection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}

	err = cursor.All(ctx, members)
	if err != nil {
		return nil, err
	}

	return &members, nil
}

func (s *Service) UpdateMember(ctx context.Context, filter bson.M, update bson.M) error {
	_, err := s.Database.ClubMembersCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) UpdateMembers(ctx context.Context, filter bson.M, update bson.M) error {
	_, err := s.Database.ClubMembersCollection.UpdateMany(ctx, filter, update)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) DeleteMember(ctx context.Context, filter bson.M) error {
	_, err := s.Database.ClubMembersCollection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) DeleteMembers(ctx context.Context, filter bson.M) error {
	_, err := s.Database.ClubMembersCollection.DeleteMany(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}
