package service

import (
	"context"
	"errors"

	"github.com/olympsis/models"
)

// Insert a new organization application into the database
func (s *Service) InsertApplication(ctx context.Context, event *models.OrganizationApplication) error {
	pong := s.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}
	s.Database.OrgApplicationCol.InsertOne(ctx, event)
	return nil
}

// Get an organization application from database
func (s *Service) FindApplication(ctx context.Context, filter interface{}, organization *models.OrganizationApplication) error {
	pong := s.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}
	s.Database.OrgApplicationCol.FindOne(ctx, filter).Decode(&organization)
	return nil
}

// Get organizations application from database
func (s *Service) FindApplications(ctx context.Context, filter interface{}, organizations *[]models.OrganizationApplication) error {
	pong := s.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}

	cursor, err := s.Database.OrgApplicationCol.Find(ctx, filter)
	if err != nil {
		return err
	}

	for cursor.Next(context.TODO()) {
		var event models.OrganizationApplication
		err := cursor.Decode(&event)
		if err != nil {
			return err
		}
		*organizations = append(*organizations, event)
	}
	return nil
}

// Update an organization application in database
func (s *Service) UpdateAnApplication(ctx context.Context, filter interface{}, update interface{}, organization *models.OrganizationApplication) error {
	pong := s.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}

	// update application
	_, err := s.Database.OrgApplicationCol.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	// find and return updated application
	err = s.FindApplication(ctx, filter, organization)
	if err != nil {
		return err
	}

	return nil
}

// Update an organization application in the database
func (s *Service) UpdateApplications(ctx context.Context, filter interface{}, update interface{}, organizations *[]models.OrganizationApplication) error {
	pong := s.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}

	// update application
	_, err := s.Database.OrgApplicationCol.UpdateMany(ctx, filter, update)
	if err != nil {
		return err
	}

	// find updated application
	err = s.FindApplications(ctx, filter, organizations)
	if err != nil {
		return err
	}

	return nil
}

// delete an organization application from the database
func (s *Service) DeleteAnApplication(ctx context.Context, filter interface{}) error {
	pong := s.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}

	// delete user
	_, err := s.Database.OrgApplicationCol.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}

// delete applications in database
func (s *Service) DeleteApplications(ctx context.Context, filter interface{}) error {
	pong := s.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}

	// delete applications
	_, err := s.Database.OrgApplicationCol.DeleteMany(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}
