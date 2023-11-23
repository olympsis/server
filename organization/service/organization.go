package service

import (
	"context"
	"errors"

	"github.com/olympsis/models"
)

// Insert a new organization into the database
func (s *Service) InsertAnOrganization(ctx context.Context, event *models.Organization) error {
	pong := s.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}
	s.Database.OrgCol.InsertOne(ctx, event)
	return nil
}

// Get an organization from database
func (s *Service) FindAnOrganization(ctx context.Context, filter interface{}, organization *models.Organization) error {
	pong := s.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}
	s.Database.OrgCol.FindOne(ctx, filter).Decode(&organization)
	return nil
}

// Get organizations from database
func (s *Service) FindOrganizations(ctx context.Context, filter interface{}, organizations *[]models.Organization) error {
	pong := s.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}

	cursor, err := s.Database.OrgCol.Find(ctx, filter)
	if err != nil {
		return err
	}

	for cursor.Next(context.TODO()) {
		var event models.Organization
		err := cursor.Decode(&event)
		if err != nil {
			return err
		}
		*organizations = append(*organizations, event)
	}
	return nil
}

// Update an organization in database
func (s *Service) UpdateAnOrganization(ctx context.Context, filter interface{}, update interface{}, organization *models.Organization) error {
	pong := s.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}

	// update user
	_, err := s.Database.OrgCol.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	// find and return updated user
	err = s.FindAnOrganization(ctx, filter, organization)
	if err != nil {
		return err
	}

	return nil
}

// Update an organization in the database
func (s *Service) UpdateAnOrganizations(ctx context.Context, filter interface{}, update interface{}, organizations *[]models.Organization) error {
	pong := s.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}

	// update event
	_, err := s.Database.OrgCol.UpdateMany(ctx, filter, update)
	if err != nil {
		return err
	}

	// find updated users
	err = s.FindOrganizations(ctx, filter, organizations)
	if err != nil {
		return err
	}

	return nil
}

// delete an organization from the database
func (s *Service) DeleteAnOrganization(ctx context.Context, filter interface{}) error {
	pong := s.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}

	// delete user
	_, err := s.Database.OrgCol.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}

// delete organizations in database
func (s *Service) DeleteOrganizations(ctx context.Context, filter interface{}) error {
	pong := s.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}

	// delete users
	_, err := s.Database.OrgCol.DeleteMany(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}
