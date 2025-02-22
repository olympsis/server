package service

import (
	"context"

	"github.com/olympsis/models"
)

// Insert new user into database
func (f *Service) InsertField(ctx context.Context, field *models.Venue) error {
	f.Database.VenueCol.InsertOne(ctx, field)
	return nil
}

// Get user from database
func (f *Service) FindField(ctx context.Context, filter interface{}, field *models.Venue) error {
	f.Database.VenueCol.FindOne(ctx, filter).Decode(&field)
	return nil
}

// get users from database
func (f *Service) FindFields(ctx context.Context, filter interface{}, fields *[]models.Venue) error {

	cursor, err := f.Database.VenueCol.Find(ctx, filter)
	if err != nil {
		return err
	}

	for cursor.Next(context.TODO()) {
		var user models.Venue
		err := cursor.Decode(&user)
		if err != nil {
			return err
		}
		*fields = append(*fields, user)
	}
	return nil
}

// update user in database
func (f *Service) UpdateField(ctx context.Context, filter interface{}, update interface{}, field *models.Venue) error {
	// update user
	_, err := f.Database.VenueCol.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	// find and return updated user
	err = f.FindField(ctx, filter, field)
	if err != nil {
		return err
	}

	return nil
}

// update users in database
func (f *Service) UpdateFields(ctx context.Context, filter interface{}, update interface{}, fields *[]models.Venue) error {
	// update users
	_, err := f.Database.VenueCol.UpdateMany(ctx, filter, update)
	if err != nil {
		return err
	}

	// find updated users
	cursor, err := f.Database.VenueCol.Find(ctx, filter)
	if err != nil {
		return err
	}

	for cursor.Next(context.TODO()) {
		var user models.Venue
		err := cursor.Decode(&user)
		if err != nil {
			return err
		}
		*fields = append(*fields, user)
	}

	return nil
}

// delete user in database
func (f *Service) DeleteField(ctx context.Context, filter interface{}) error {

	// delete user
	_, err := f.Database.VenueCol.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}

// delete users in database
func (f *Service) DeleteFields(ctx context.Context, filter interface{}) error {

	// delete users
	_, err := f.Database.VenueCol.DeleteMany(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}
