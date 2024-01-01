package service

import (
	"context"

	"github.com/olympsis/models"
)

// Insert new user into database
func (a *Service) InsertUser(ctx context.Context, user *models.AuthUser) error {
	a.Database.AuthCol.InsertOne(ctx, user)
	return nil
}

// Get user from database
func (a *Service) FindUser(ctx context.Context, filter interface{}, user *models.AuthUser) error {
	a.Database.AuthCol.FindOne(ctx, filter).Decode(&user)
	return nil
}

// get users from database
func (a *Service) FindUsers(ctx context.Context, filter interface{}, users *[]models.AuthUser) error {

	cursor, err := a.Database.AuthCol.Find(ctx, filter)
	if err != nil {
		return err
	}

	for cursor.Next(context.TODO()) {
		var user models.AuthUser
		err := cursor.Decode(&user)
		if err != nil {
			return err
		}
		*users = append(*users, user)
	}
	return nil
}

// update user in database
func (a *Service) UpdateUser(ctx context.Context, filter interface{}, update interface{}, user *models.AuthUser) error {

	// update user
	_, err := a.Database.AuthCol.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	// find and return updated user
	err = a.FindUser(ctx, filter, user)
	if err != nil {
		return err
	}

	return nil
}

// update users in database
func (a *Service) UpdateUsers(ctx context.Context, filter interface{}, update interface{}, users *[]models.AuthUser) error {

	// update users
	_, err := a.Database.AuthCol.UpdateMany(ctx, filter, update)
	if err != nil {
		return err
	}

	// find updated users
	cursor, err := a.Database.AuthCol.Find(ctx, filter)
	if err != nil {
		return err
	}

	for cursor.Next(context.TODO()) {
		var user models.AuthUser
		err := cursor.Decode(&user)
		if err != nil {
			return err
		}
		*users = append(*users, user)
	}

	return nil
}

// delete user in database
func (a *Service) DeleteUser(ctx context.Context, filter interface{}) error {

	// delete user
	_, err := a.Database.AuthCol.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}

// delete users in database
func (a *Service) DeleteUsers(ctx context.Context, filter interface{}) error {

	// delete users
	_, err := a.Database.AuthCol.DeleteMany(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}
