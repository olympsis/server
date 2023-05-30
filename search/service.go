package search

import (
	"context"
	"olympsis-server/database"
	"olympsis-server/models"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Service struct {
	Database *database.Database
	Log      *logrus.Logger
}

func NewSearchService(l *logrus.Logger, d *database.Database) *Service {
	return &Service{Log: l, Database: d}
}

func (s *Service) SearchUserByUUID(uuid string) (models.UserData, error) {

	// context/filter
	ctx := context.Background()
	filter := bson.M{"uuid": uuid}
	opts := options.FindOneOptions{}

	// find and decode auth user data
	var auth models.AuthUser
	err := s.Database.AuthCol.FindOne(ctx, filter).Decode(&auth)
	if err != nil {
		return models.UserData{}, err
	}

	// find and decode user metadata
	var user models.User
	err = s.Database.UserCol.FindOne(ctx, filter, &opts).Decode(&user)
	if err != nil {
		return models.UserData{}, err
	}

	// create user data object
	userData := models.UserData{
		UUID:        auth.UUID,
		Username:    user.UserName,
		FirstName:   auth.FirstName,
		LastName:    auth.LastName,
		ImageURL:    user.ImageURL,
		Visibility:  user.Visibility,
		DeviceToken: user.DeviceToken,
	}

	// if user visibility is public display this data if not then dont
	if user.Visibility == "public" {
		userData.Bio = user.Bio
		userData.Clubs = user.Clubs
		userData.Sports = user.Sports
	}
	return userData, nil
}

func (s *Service) SearchUserByUsername(name string) (models.UserData, error) {

	// context/filter
	ctx := context.Background()
	filter := bson.M{"username": name}
	opts := options.FindOneOptions{}

	// find and decode user metadata
	var user models.User
	err := s.Database.UserCol.FindOne(ctx, filter, &opts).Decode(&user)
	if err != nil {
		return models.UserData{}, err
	}

	filter = bson.M{"uuid": user.UUID}

	// return only uuid, first name and last name

	// find and decode auth user data
	var auth models.AuthUser
	err = s.Database.AuthCol.FindOne(ctx, filter).Decode(&auth)
	if err != nil {
		return models.UserData{}, err
	}

	// create user data object
	userData := models.UserData{
		UUID:        auth.UUID,
		Username:    user.UserName,
		FirstName:   auth.FirstName,
		LastName:    auth.LastName,
		ImageURL:    user.ImageURL,
		Visibility:  user.Visibility,
		DeviceToken: user.DeviceToken,
	}

	// if user visibility is public display this data if not then dont
	if user.Visibility == "public" {
		userData.Bio = user.Bio
		userData.Clubs = user.Clubs
		userData.Sports = user.Sports
	}
	return userData, nil
}
