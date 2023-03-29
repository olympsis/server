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

	// logrus logger to Logger information about service and errors
	Logger *logrus.Logger

	// mux Router to complete http requests
	Router *mux.Router
}

type Event struct {
	ID              primitive.ObjectID `json:"id" bson:"_id"`
	OwnerID         string             `json:"ownerId" bson:"ownerId"`
	OwnerData       *UserData          `json:"ownerData" bson:"ownerData,omitempty"`
	ClubID          primitive.ObjectID `json:"clubId" bson:"clubId"`
	FieldId         primitive.ObjectID `json:"fieldId" bson:"fieldId"`
	ImageURL        string             `json:"imageURL" bson:"imageURL"`
	Title           string             `json:"title" bson:"title"`
	Body            string             `json:"body" bson:"body"`
	Sport           string             `json:"sport" bson:"sport"`
	Level           int8               `json:"level" bson:"level"`
	Status          string             `json:"status" bson:"status"`
	StartTime       int64              `json:"startTime" bson:"startTime"`
	ActualStartTime int64              `json:"actualStartTime" bson:"actualStartTime"`
	StopTime        int64              `json:"stopTime" bson:"stopTime"`
	MaxParticipants int8               `json:"maxParticipants" bson:"maxParticipants"`
	Participants    []Participant      `json:"participants" bson:"participants"`
	Likes           []Like             `json:"likes" bson:"likes"`
	Visibility      string             `json:"visibility" bson:"visibility"`
	CreatedAt       int64              `json:"createdAt" bson:"createdAt"`
}

type EventsResponse struct {
	TotalEvents int     `json:"totalEvents"`
	Events      []Event `json:"events"`
}

type Like struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	UUID      string             `json:"uuid" bson:"uuid"`
	CreatedAt int64              `json:"createdAt" bson:"createdAt"`
}

type Participant struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	UUID      string             `json:"uuid" bson:"uuid"`
	Status    string             `json:"status" bson:"status"`
	Data      *UserData          `json:"data,omitempty" bson:"data,omitempty"`
	CreatedAt int64              `json:"createdAt" bson:"createdAt"`
}

type FullEventError struct {
	MSG   string `json:"msg"`
	Event Event  `json:"event"`
}

type Field struct {
	ID       primitive.ObjectID `json:"id" bson:"_id"`
	Owner    string             `json:"owner" bson:"owner"`
	Name     string             `json:"name" bson:"name"`
	Notes    string             `json:"notes" bson:"notes"`
	Sports   []string           `json:"sports" bson:"sports"`
	Images   []string           `json:"images" bson:"images"`
	Location GeoJSON            `json:"location" bson:"location"`
	City     string             `json:"city" bson:"city"`
	State    string             `json:"state" bson:"state"`
	Country  string             `json:"country" bson:"country"`
	IsPublic bool               `json:"isPublic" bson:"isPublic"`
}

type GeoJSON struct {
	Type        string    `json:"type" bson:"type"`
	Coordinates []float64 `json:"coordinates" bson:"coordinates"`
}

/// NOTIFICATIONS

type NotificationRequest struct {
	Tokens []string `json:"tokens"`
	Title  string   `json:"title"`
	Body   string   `json:"body"`
	Topic  string   `json:"topic"`
}

// USER LOOKUP
/*
Lookup User
- contains identifiable user data that others can see
*/
type UserData struct {
	FirstName   string `json:"firstName" bson:"firstName"`
	LastName    string `json:"lastName" bson:"lastName"`
	Username    string `json:"username" bson:"username"`
	ImageURL    string `json:"imageURL,omitempty" bson:"imageURL,omitempty"`
	DeviceToken string `json:"deviceToken,omitempty" bson:"deviceToken,omitempty"`
}

type UserIdentifiableData struct {
	FirstName string `json:"firstName" bson:"firstName"`
	LastName  string `json:"lastName" bson:"lastName"`
}

type UserMetadata struct {
	Username    string `json:"username" bson:"username"`
	ImageURL    string `json:"imageURL,omitempty" bson:"imageURL,omitempty"`
	DeviceToken string `json:"deviceToken,omitempty" bson:"deviceToken,omitempty"`
}
