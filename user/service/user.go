package service

import (
	"context"

	"github.com/olympsis/models"
)

// Insert new user into database
func (u *Service) InsertUser(ctx context.Context, user *models.User) error {
	u.Database.UserCollection.InsertOne(ctx, user)
	return nil
}

// Get user from database
func (u *Service) FindUser(ctx context.Context, filter interface{}, user *models.User) error {
	u.Database.UserCollection.FindOne(ctx, filter).Decode(&user)
	return nil
}

// get users from database
func (u *Service) FindUsers(ctx context.Context, filter interface{}, users *[]models.User) error {

	cursor, err := u.Database.UserCollection.Find(ctx, filter)
	if err != nil {
		return err
	}

	for cursor.Next(context.TODO()) {
		var user models.User
		err := cursor.Decode(&user)
		if err != nil {
			return err
		}
		*users = append(*users, user)
	}
	return nil
}

// update user in database
func (u *Service) UpdateUser(ctx context.Context, filter interface{}, update interface{}, user *models.UserDao) error {

	// update user
	_, err := u.Database.UserCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	return nil
}

// update users in database
func (u *Service) UpdateUsers(ctx context.Context, filter interface{}, update interface{}, users *[]models.User) error {
	// update users
	_, err := u.Database.UserCollection.UpdateMany(ctx, filter, update)
	if err != nil {
		return err
	}

	// find updated users
	cursor, err := u.Database.UserCollection.Find(ctx, filter)
	if err != nil {
		return err
	}

	for cursor.Next(context.TODO()) {
		var user models.User
		err := cursor.Decode(&user)
		if err != nil {
			return err
		}
		*users = append(*users, user)
	}

	return nil
}

// delete user in database
func (u *Service) DeleteUser(ctx context.Context, filter interface{}) error {

	// delete user
	_, err := u.Database.UserCollection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}

// delete users in database
func (u *Service) DeleteUsers(ctx context.Context, filter interface{}) error {

	// delete users
	_, err := u.Database.UserCollection.DeleteMany(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}
