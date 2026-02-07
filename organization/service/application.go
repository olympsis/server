package service

import (
	"context"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Insert a new organization application into the database
func (s *Service) InsertApplication(ctx context.Context, event *models.OrganizationApplicationDao) (*bson.ObjectID, error) {
	res, err := s.Database.OrgApplicationCollection.InsertOne(ctx, event)
	if err != nil {
		return nil, err
	}
	id := res.InsertedID.(bson.ObjectID)
	return &id, nil
}

// Get an organization application from database
func (s *Service) FindApplication(ctx context.Context, filter interface{}, organization *models.OrganizationApplicationDao) error {
	s.Database.OrgApplicationCollection.FindOne(ctx, filter).Decode(&organization)
	return nil
}

// Get organizations application from database
func (s *Service) FindApplications(ctx context.Context, filter interface{}, organizations *[]models.OrganizationApplication) error {

	cursor, err := s.Database.OrgApplicationCollection.Find(ctx, filter)
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
func (s *Service) UpdateAnApplication(ctx context.Context, filter interface{}, update interface{}, invitation *models.OrganizationApplicationDao) error {

	// update application
	_, err := s.Database.OrgApplicationCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	// find and return updated application
	err = s.FindApplication(ctx, filter, invitation)
	if err != nil {
		return err
	}

	return nil
}

// Update an organization application in the database
func (s *Service) UpdateApplications(ctx context.Context, filter interface{}, update interface{}, organizations *[]models.OrganizationApplication) error {

	// update application
	_, err := s.Database.OrgApplicationCollection.UpdateMany(ctx, filter, update)
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

	// delete user
	_, err := s.Database.OrgApplicationCollection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}

// delete applications in database
func (s *Service) DeleteApplications(ctx context.Context, filter interface{}) error {

	// delete applications
	_, err := s.Database.OrgApplicationCollection.DeleteMany(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}
