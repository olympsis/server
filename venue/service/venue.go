package service

import (
	"context"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Insert new venue into the database
func (f *Service) InsertVenue(ctx context.Context, field *models.Venue) (*mongo.InsertOneResult, error) {
	return f.Database.VenuesCollection.InsertOne(ctx, field)
}

// Get a venue from the database
func (f *Service) FindVenue(ctx context.Context, filter interface{}, field *models.Venue) error {
	f.Database.VenuesCollection.FindOne(ctx, filter).Decode(&field)
	return nil
}

// Get venues from the database
func (f *Service) FindVenues(ctx context.Context, filter interface{}, fields *[]models.Venue) error {

	cursor, err := f.Database.VenuesCollection.Find(ctx, filter)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

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

// Modify a venue in the database
func (f *Service) ModifyVenue(ctx context.Context, filter interface{}, update interface{}, field *models.Venue) error {
	// update user
	_, err := f.Database.VenuesCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	// find and return updated user
	err = f.FindVenue(ctx, filter, field)
	if err != nil {
		return err
	}

	return nil
}

// Modify venues in the database
func (f *Service) ModifyVenues(ctx context.Context, filter interface{}, update interface{}, fields *[]models.Venue) error {
	// update users
	_, err := f.Database.VenuesCollection.UpdateMany(ctx, filter, update)
	if err != nil {
		return err
	}

	// find updated users
	cursor, err := f.Database.VenuesCollection.Find(ctx, filter)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

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

// Remove venue from database
func (f *Service) DeleteField(ctx context.Context, filter interface{}) error {

	// delete user
	_, err := f.Database.VenuesCollection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}

// Remove venues from database
func (f *Service) DeleteFields(ctx context.Context, filter interface{}) error {

	// delete users
	_, err := f.Database.VenuesCollection.DeleteMany(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}
