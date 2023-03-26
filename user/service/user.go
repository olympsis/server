package service

import (
	"context"
	"errors"
)

// Insert new user into database
func (u *Service) InsertUser(ctx context.Context, user *User) error {
	pong := u.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}
	u.Database.UserCol.InsertOne(ctx, user)
	return nil
}

// Get user from database
func (u *Service) FindUser(ctx context.Context, filter interface{}, user *User) error {
	pong := u.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}
	u.Database.UserCol.FindOne(ctx, filter).Decode(&user)
	return nil
}

// get users from database
func (u *Service) FindUsers(ctx context.Context, filter interface{}, users *[]User) error {
	pong := u.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}

	cursor, err := u.Database.UserCol.Find(ctx, filter)
	if err != nil {
		return err
	}

	for cursor.Next(context.TODO()) {
		var user User
		err := cursor.Decode(&user)
		if err != nil {
			return err
		}
		*users = append(*users, user)
	}
	return nil
}

// update user in database
func (u *Service) UpdateUser(ctx context.Context, filter interface{}, update interface{}, user *User) error {
	pong := u.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}

	// update user
	_, err := u.Database.UserCol.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	// find and return updated user
	err = u.FindUser(ctx, filter, user)
	if err != nil {
		return err
	}

	return nil
}

// update users in database
func (u *Service) UpdateUsers(ctx context.Context, filter interface{}, update interface{}, users *[]User) error {
	pong := u.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}

	// update users
	_, err := u.Database.UserCol.UpdateMany(ctx, filter, update)
	if err != nil {
		return err
	}

	// find updated users
	cursor, err := u.Database.UserCol.Find(ctx, filter)
	if err != nil {
		return err
	}

	for cursor.Next(context.TODO()) {
		var user User
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
	pong := u.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}

	// delete user
	_, err := u.Database.UserCol.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}

// delete users in database
func (u *Service) DeleteUsers(ctx context.Context, filter interface{}) error {
	pong := u.Database.PingDatabase()
	if !pong {
		return errors.New("failed to connect to database")
	}

	// delete users
	_, err := u.Database.UserCol.DeleteMany(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}
