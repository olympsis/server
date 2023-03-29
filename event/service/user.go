package service

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
)

func (e *Service) FetchDataAboutUser(userId string, data *UserData) error {
	// grab user identifable data (first/last name)
	filter := bson.M{"uuid": userId}
	var authData UserIdentifiableData
	err := e.Database.AuthCol.FindOne(context.Background(), filter).Decode(&authData)
	if err != nil {
		return err
	}

	// grab user metadata (deviceToken, imageURL)
	var metadata UserMetadata
	err = e.Database.UserCol.FindOne(context.Background(), filter).Decode(&metadata)
	if err != nil {
		return err
	}

	data.FirstName = authData.FirstName
	data.LastName = authData.LastName
	data.Username = metadata.Username
	data.ImageURL = metadata.ImageURL
	data.DeviceToken = metadata.DeviceToken

	return nil
}
