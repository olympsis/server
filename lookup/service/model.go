package service

import (
	"olympsis-server/database"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/*
Field Service Struct
*/
type Service struct {
	Database *database.Database
	Logger   *logrus.Logger
	Router   *mux.Router
}

/*
Lookup User
- contains identifiable user data that others can see
*/
type LookUpUser struct {
	ID          string   `json:"id" bson:"_id"`
	FirstName   string   `json:"firstName" bson:"firstName"`
	LastName    string   `json:"lastName" bson:"lastName"`
	Username    string   `json:"username" bson:"username"`
	ImageURL    string   `json:"imageURL,omitempty" bson:"imageURL,omitempty"`
	Bio         string   `json:"bio,omitempty" bson:"bio,omitempty"`
	Clubs       []string `json:"clubs,omitempty" bson:"clubs,omitempty"`
	Sports      []string `json:"sports,omitempty" bson:"sports,omitempty"`
	Badges      []Badge  `json:"badges,omitempty" bson:"badges,omitempty"`
	Trophies    []Trophy `json:"trophies,omitempty" bson:"trophies,omitempty"`
	Friends     []Friend `json:"friends,omitempty" bson:"friends,omitempty"`
	DeviceToken string   `json:"deviceToken,omitempty" bson:"deviceToken,omitempty"`
}

/*
Auth User
- User data for auth database
- Contains user identifiable data
*/
type AuthUser struct {
	UUID      string `json:"uuid" bson:"uuid"`
	FirstName string `json:"firstName" bson:"firstName"`
	LastName  string `json:"lastName" bson:"lastName"`
	Email     string `json:"email" bson:"email"`
	Token     string `json:"token" bson:"token"`
	AuthToken string `json:"authToken,omitempty" bson:"authToken,omitempty"`
	Provider  string `json:"provider" bson:"provider"`
	CreatedAt int64  `json:"createdAt" bson:"createdAt"`
}

/*
User Data
- 	Contains user identifiable data
*/
type UserData struct {
	ID          primitive.ObjectID `json:"id" bson:"_id"`
	UUID        string             `json:"uuid" bson:"uuid"`
	UserName    string             `json:"username" bson:"username"`
	ImageURL    string             `json:"imageURL,omitempty" bson:"imageURL,omitempty"`
	Bio         string             `json:"bio,omitempty" bson:"bio,omitempty"`
	IsPublic    bool               `json:"isPublic" bson:"isPublic"`
	Clubs       []string           `json:"clubs,omitempty" bson:"clubs,omitempty"`
	Sports      []string           `json:"sports,omitempty" bson:"sports,omitempty"`
	Badges      []Badge            `json:"badges,omitempty" bson:"badges,omitempty"`
	Trophies    []Trophy           `json:"trophies,omitempty" bson:"trophies,omitempty"`
	Friends     []Friend           `json:"friends,omitempty" bson:"friends,omitempty"`
	DeviceToken string             `json:"deviceToken,omitempty" bson:"deviceToken,omitempty"`
}

/*
Trophy
  - Trophy object
*/
type Trophy struct {
	ID          primitive.ObjectID `json:"id" bson:"_id"`
	Name        string             `json:"name" bson:"name"`
	Title       string             `json:"title" bson:"title"`
	ImageURL    string             `json:"imageURL" bson:"imageURL"`
	EventId     primitive.ObjectID `json:"eventId" bson:"eventId"`
	Description string             `json:"description" bson:"description"`
	AchievedAt  int64              `json:"achievedAt" bson:"achievedAt"`
}

/*
Badge
  - Badge object
*/
type Badge struct {
	ID          primitive.ObjectID `json:"id" bson:"_id"`
	Name        string             `json:"name" bson:"name"`
	Title       string             `json:"title" bson:"title"`
	ImageURL    string             `json:"imageURL" bson:"imageURL"`
	EventId     primitive.ObjectID `json:"eventId" bson:"eventId"`
	Description string             `json:"description" bson:"description"`
	AchievedAt  int64              `json:"achievedAt" bson:"achievedAt"`
}

type BatchRequest struct {
	UUIDS []string `json:"uuids"`
}

/*
Friend
  - Friend object
*/
type Friend struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	UUID      string             `json:"uuid" bson:"uuid"`
	CreatedAt int64              `json:"createdAt" bson:"createdAt"`
}
