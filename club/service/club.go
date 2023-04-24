package service

import (
	"context"
	"errors"
)

func (s *Service) InsertClub(ctx context.Context, club *Club) error {
	pong := s.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}
	s.Database.ClubCol.InsertOne(ctx, club)
	return nil
}

func (s *Service) FindClub(ctx context.Context, filter interface{}, club *Club) error {
	pong := s.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}
	s.Database.ClubCol.FindOne(ctx, filter).Decode(&club)
	return nil
}

func (s *Service) FindClubs(ctx context.Context, filter interface{}, clubs *[]Club) error {
	pong := s.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}
	cursor, err := s.Database.ClubCol.Find(ctx, filter)
	if err != nil {
		return err
	}

	for cursor.Next(context.TODO()) {
		var club Club
		err = cursor.Decode(&club)
		if err != nil {
			return err
		}
		*clubs = append(*clubs, club)
	}
	return nil
}

func (s *Service) UpdateAClub(ctx context.Context, filter interface{}, update interface{}) error {
	pong := s.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}

	// update user
	_, err := s.Database.ClubCol.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) UpdateClubs(ctx context.Context, filter interface{}, update interface{}, clubs *[]Club) error {
	return nil
}

func (s *Service) DeleteAClub(ctx context.Context, filter interface{}) error {
	return nil
}

func (s *Service) DeleteClubs(ctx context.Context, filter interface{}) error {
	return nil
}
