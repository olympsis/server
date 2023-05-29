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

	// return only uuid, first name and last name
	opts := options.FindOneOptions{}
	opts.SetProjection(bson.M{
		"_id":          0,
		"uuid":         1,
		"fist_name":    1,
		"last_name":    1,
		"email":        0,
		"token":        0,
		"access_token": 0,
		"provider":     0,
		"created_at":   0,
	})

	// find and decode auth user data
	var auth models.AuthUser
	err := s.Database.AuthCol.FindOne(ctx, filter).Decode(&auth)
	if err != nil {
		return models.UserData{}, err
	}

	// return onl username, bio, image_url, visibility, clubs and sports
	opts.SetProjection(bson.M{
		"_id":          0,
		"uuid":         0,
		"username":     1,
		"bio":          1,
		"image_url":    1,
		"visibility":   1,
		"clubs":        1,
		"sports":       0,
		"device_token": 0,
	})

	// find and decode user metadata
	var user models.User
	err = s.Database.UserCol.FindOne(ctx, filter, &opts).Decode(&user)
	if err != nil {
		return models.UserData{}, err
	}

	// create user data object
	userData := models.UserData{
		UUID:       auth.UUID,
		Username:   user.UserName,
		FirstName:  auth.FirstName,
		LastName:   auth.LastName,
		ImageURL:   user.ImageURL,
		Visibility: user.Visibility,
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

	// return onl username, bio, image_url, visibility, clubs and sports
	opts := options.FindOneOptions{}
	opts.SetProjection(bson.M{
		"_id":          0,
		"uuid":         1,
		"username":     1,
		"bio":          1,
		"image_url":    1,
		"visibility":   1,
		"clubs":        1,
		"sports":       0,
		"device_token": 0,
	})

	// find and decode user metadata
	var user models.User
	err := s.Database.UserCol.FindOne(ctx, filter, &opts).Decode(&user)
	if err != nil {
		return models.UserData{}, err
	}

	filter = bson.M{"uuid": user.UUID}

	// return only uuid, first name and last name
	opts.SetProjection(bson.M{
		"_id":          0,
		"uuid":         1,
		"fist_name":    1,
		"last_name":    1,
		"email":        0,
		"token":        0,
		"access_token": 0,
		"provider":     0,
		"created_at":   0,
	})

	// find and decode auth user data
	var auth models.AuthUser
	err = s.Database.AuthCol.FindOne(ctx, filter).Decode(&auth)
	if err != nil {
		return models.UserData{}, err
	}

	// create user data object
	userData := models.UserData{
		UUID:       auth.UUID,
		Username:   user.UserName,
		FirstName:  auth.FirstName,
		LastName:   auth.LastName,
		ImageURL:   user.ImageURL,
		Visibility: user.Visibility,
	}

	// if user visibility is public display this data if not then dont
	if user.Visibility == "public" {
		userData.Bio = user.Bio
		userData.Clubs = user.Clubs
		userData.Sports = user.Sports
	}
	return userData, nil
}
