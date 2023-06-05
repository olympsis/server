package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Event struct {
	ID              primitive.ObjectID `json:"id" bson:"_id"`
	Poster          string             `json:"poster" bson:"poster"`
	ClubID          primitive.ObjectID `json:"club_id" bson:"club_id"`
	FieldID         primitive.ObjectID `json:"field_id" bson:"field_id"`
	ImageURL        string             `json:"image_url" bson:"image_url"`
	Title           string             `json:"title" bson:"title"`
	Body            string             `json:"body" bson:"body"`
	Sport           string             `json:"sport" bson:"sport"`
	Level           int8               `json:"level" bson:"level"`
	Status          string             `json:"status" bson:"status"`
	StartTime       int64              `json:"start_time,omitempty" bson:"start_time,omitempty"`
	ActualStartTime int64              `json:"actual_start_time,omitempty" bson:"actual_start_time,omitempty"`
	StopTime        int64              `json:"stop_time,omitempty" bson:"stop_time,omitempty"`
	MaxParticipants int8               `json:"max_participants" bson:"max_participants"`
	Participants    []Participant      `json:"participants,omitempty" bson:"participants,omitempty"`
	Likes           []Like             `json:"likes,omitempty" bson:"likes,omitempty"`
	Visibility      string             `json:"visibility" bson:"visibility"`
	Data            *EventData         `json:"data" bson:"data,omitempty"`
	CreatedAt       int64              `json:"created_at" bson:"created_at"`
}

type EventData struct {
	Poster *UserData `json:"poster,omitempty"`
	Club   *Club     `json:"club,omitempty"`
	Field  *Field    `json:"field,omitempty"`
}

type EventsResponse struct {
	TotalEvents int     `json:"total_events"`
	Events      []Event `json:"events"`
}

type Participant struct {
	ID        primitive.ObjectID `json:"id,omitempty" bson:"_id"`
	UUID      string             `json:"uuid" bson:"uuid"`
	Data      *UserData          `json:"data,omitempty" bson:"data,omitempty"`
	Status    string             `json:"status" bson:"status"`
	CreatedAt int64              `json:"created_at,omitempty" bson:"created_at"`
}

type FullEventError struct {
	MSG   string `json:"msg"`
	Event Event  `json:"event"`
}
