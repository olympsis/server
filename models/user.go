package models

import "go.mongodb.org/mongo-driver/bson/primitive"

/*
User Data
  - Contains user identifiable data
*/
type User struct {
	ID          primitive.ObjectID `json:"id,omitempty" bson:"_id"`
	UUID        string             `json:"uuid,omitempty" bson:"uuid"`
	UserName    string             `json:"username,omitempty" bson:"username"`
	Bio         string             `json:"bio,omitempty" bson:"bio,omitempty"`
	ImageURL    string             `json:"image_url,omitempty" bson:"image_url,omitempty"`
	Visibility  string             `json:"visibility,omitempty" bson:"visibility"`
	Clubs       []string           `json:"clubs,omitempty" bson:"clubs,omitempty"`
	Sports      []string           `json:"sports,omitempty" bson:"sports,omitempty"`
	DeviceToken string             `json:"device_token,omitempty" bson:"device_token,omitempty"`
}

// User data to return when looking up info about a user
type UserData struct {
	UUID        string   `json:"uuid"`
	Username    string   `json:"username"`
	FirstName   string   `json:"first_name"`
	LastName    string   `json:"last_name"`
	ImageURL    string   `json:"image_url"`
	Visibility  string   `json:"visibility"`
	Bio         string   `json:"bio,omitempty"`
	Clubs       []string `json:"clubs,omitempty"`
	Sports      []string `json:"sports,omitempty"`
	DeviceToken string   `json:"device_token,omitempty" bson:"device_token,omitempty"`
}
