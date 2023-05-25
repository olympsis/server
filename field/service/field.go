package service

import (
	"context"
	"errors"
	"olympsis-server/models"
)

// Insert new user into database
func (f *Service) InsertField(ctx context.Context, field *models.Field) error {
	pong := f.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}
	f.Database.FieldCol.InsertOne(ctx, field)
	return nil
}

// Get user from database
func (f *Service) FindField(ctx context.Context, filter interface{}, field *models.Field) error {
	pong := f.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}
	f.Database.FieldCol.FindOne(ctx, filter).Decode(&field)
	return nil
}

// get users from database
func (f *Service) FindFields(ctx context.Context, filter interface{}, fields *[]models.Field) error {
	pong := f.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}

	cursor, err := f.Database.FieldCol.Find(ctx, filter)
	if err != nil {
		return err
	}

	for cursor.Next(context.TODO()) {
		var user models.Field
		err := cursor.Decode(&user)
		if err != nil {
			return err
		}
		*fields = append(*fields, user)
	}
	return nil
}

// update user in database
func (f *Service) UpdateField(ctx context.Context, filter interface{}, update interface{}, field *models.Field) error {
	pong := f.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}

	// update user
	_, err := f.Database.FieldCol.UpdateOne(ctx, filter, update)
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
func (f *Service) UpdateFields(ctx context.Context, filter interface{}, update interface{}, fields *[]models.Field) error {
	pong := f.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}

	// update users
	_, err := f.Database.FieldCol.UpdateMany(ctx, filter, update)
	if err != nil {
		return err
	}

	// find updated users
	cursor, err := f.Database.FieldCol.Find(ctx, filter)
	if err != nil {
		return err
	}

	for cursor.Next(context.TODO()) {
		var user models.Field
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
	pong := f.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}

	// delete user
	_, err := f.Database.FieldCol.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}

// delete users in database
func (f *Service) DeleteFields(ctx context.Context, filter interface{}) error {
	pong := f.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}

	// delete users
	_, err := f.Database.FieldCol.DeleteMany(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}
