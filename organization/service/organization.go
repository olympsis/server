package service

import (
	"context"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Insert a new organization into the database
func (s *Service) InsertAnOrganization(ctx context.Context, event *models.OrganizationDao) (*primitive.ObjectID, error) {
	resp, err := s.Database.OrgCol.InsertOne(ctx, event)
	if err != nil {
		return nil, err
	}
	id := resp.InsertedID.(primitive.ObjectID)
	return &id, nil
}

// Get an organization from database
func (s *Service) FindAnOrganization(ctx context.Context, filter interface{}) (*models.OrganizationDao, error) {
	var org models.OrganizationDao
	err := s.Database.OrgCol.FindOne(ctx, filter).Decode(&org)
	if err != nil {
		return nil, err
	}
	return &org, nil
}

// Get organizations from database
func (s *Service) FindOrganizations(ctx context.Context, filter interface{}, organizations *[]models.Organization) error {

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
func (s *Service) UpdateAnOrganization(ctx context.Context, filter interface{}, update interface{}) error {
	// update user
	_, err := s.Database.OrgCol.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	return nil
}

// Update an organization in the database
func (s *Service) UpdateOrganizations(ctx context.Context, filter interface{}, update interface{}, organizations *[]models.Organization) error {

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

	// delete user
	_, err := s.Database.OrgCol.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}

// delete organizations in database
func (s *Service) DeleteOrganizations(ctx context.Context, filter interface{}) error {

	// delete users
	_, err := s.Database.OrgCol.DeleteMany(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}
