package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Event struct {
	ID              primitive.ObjectID `json:"id" bson:"_id"`
	Poster          string             `json:"poster" bson:"poster"`
	ClubID          primitive.ObjectID `json:"club_id" bson:"club_id"`
	FieldId         primitive.ObjectID `json:"field_id" bson:"field_id"`
	ImageURL        string             `json:"image_url" bson:"image_url"`
	Title           string             `json:"title" bson:"title"`
	Body            string             `json:"body" bson:"body"`
	Sport           string             `json:"sport" bson:"sport"`
	Level           int8               `json:"level" bson:"level"`
	Status          string             `json:"status" bson:"status"`
	StartTime       int64              `json:"start_time" bson:"start_time"`
	ActualStartTime int64              `json:"actual_start_time" bson:"actual_start_time"`
	StopTime        int64              `json:"stop_time" bson:"stop_time"`
	MaxParticipants int8               `json:"max_participants" bson:"max_participants"`
	Participants    []Participant      `json:"participants" bson:"participants"`
	Likes           []Like             `json:"likes" bson:"likes"`
	Visibility      string             `json:"visibility" bson:"visibility"`
	CreatedAt       int64              `json:"created_at" bson:"created_at"`
}

type EventsResponse struct {
	TotalEvents int     `json:"total_events"`
	Events      []Event `json:"events"`
}
type Participant struct {
	ID        primitive.ObjectID `json:"id,omitempty" bson:"_id"`
	UUID      string             `json:"uuid" bson:"uuid"`
	Status    string             `json:"status" bson:"status"`
	CreatedAt int64              `json:"created_at,omitempty" bson:"created_at"`
}

type FullEventError struct {
	MSG   string `json:"msg"`
	Event Event  `json:"event"`
}
