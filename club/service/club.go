package service

import (
	"context"

	"github.com/olympsis/models"
)

func (s *Service) InsertClub(ctx context.Context, club *models.Club) error {
	s.Database.ClubCol.InsertOne(ctx, club)
	return nil
}

func (s *Service) FindClub(ctx context.Context, filter interface{}, club *models.Club) error {
	s.Database.ClubCol.FindOne(ctx, filter).Decode(&club)
	return nil
}

func (s *Service) FindClubs(ctx context.Context, filter interface{}, clubs *[]models.Club) error {
	cursor, err := s.Database.ClubCol.Find(ctx, filter)
	if err != nil {
		return err
	}

	for cursor.Next(context.TODO()) {
		var club models.Club
		err = cursor.Decode(&club)
		if err != nil {
			return err
		}
		*clubs = append(*clubs, club)
	}
	return nil
}

func (s *Service) UpdateClub(ctx context.Context, filter interface{}, update interface{}) error {
	// update user
	_, err := s.Database.ClubCol.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) UpdateClubs(ctx context.Context, filter interface{}, update interface{}, clubs *[]models.Club) error {
	return nil
}

func (s *Service) RemoveClub(ctx context.Context, filter interface{}) error {
	_, err := s.Database.ClubCol.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) RemoveClubs(ctx context.Context, filter interface{}) error {
	return nil
}
