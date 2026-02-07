package service

import (
	"context"
	"olympsis-server/aggregations"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Insert auth user into database
func (a *Service) InsertUser(ctx context.Context, user *models.AuthUserDao) error {
	a.Database.AuthCollection.InsertOne(ctx, user)
	return nil
}

// Get auth user from database
func (a *Service) FindUser(ctx context.Context, filter interface{}) (*models.AuthUser, error) {
	var user models.AuthUser
	err := a.Database.AuthCollection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// get auth users from database
func (a *Service) FindUsers(ctx context.Context, filter bson.M, users *[]models.AuthUser) error {

	cursor, err := a.Database.AuthCollection.Find(ctx, filter)
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

// update auth user in database
func (a *Service) UpdateUser(ctx context.Context, uuid string, update bson.M) (*models.UserData, error) {

	// update user
	_, err := a.Database.AuthCollection.UpdateOne(ctx, bson.M{"uuid": uuid}, update)
	if err != nil {
		return nil, err
	}

	// find and return updated user
	user, err := aggregations.AggregateUser(&uuid, a.Database)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// update auth users in database
func (a *Service) UpdateUsers(ctx context.Context, filter bson.M, update bson.M, users *[]models.AuthUser) error {

	// update users
	_, err := a.Database.AuthCollection.UpdateMany(ctx, filter, update)
	if err != nil {
		return err
	}

	// find updated users
	cursor, err := a.Database.AuthCollection.Find(ctx, filter)
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

// delete auth user from database
func (a *Service) DeleteUser(ctx context.Context, filter bson.M) error {

	// delete user
	_, err := a.Database.AuthCollection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}

// delete auth users from database
func (a *Service) DeleteUsers(ctx context.Context, filter bson.M) error {

	// delete users
	_, err := a.Database.AuthCollection.DeleteMany(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}
