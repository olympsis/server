package models

import "go.mongodb.org/mongo-driver/bson/primitive"

/*
User Data
- 	Contains user identifiable data
*/
type User struct {
	ID          primitive.ObjectID `json:"id,omitempty" bson:"_id"`
	UUID        string             `json:"uuid,omitempty" bson:"uuid,omitempty"`
	UserName    string             `json:"username" bson:"username"`
	Bio         string             `json:"bio,omitempty" bson:"bio,omitempty"`
	ImageURL    string             `json:"image_url,omitempty" bson:"image_url,omitempty"`
	Visibility  string             `json:"visibility" bson:"visibility"`
	Clubs       []string           `json:"clubs,omitempty" bson:"clubs,omitempty"`
	Sports      []string           `json:"sports,omitempty" bson:"sports,omitempty"`
	DeviceToken string             `json:"device_token,omitempty" bson:"device_token,omitempty"`
}
