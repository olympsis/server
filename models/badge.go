package models

import "go.mongodb.org/mongo-driver/bson/primitive"

/*
Badge
  - Badge object
*/
type Badge struct {
	ID          primitive.ObjectID `json:"_id,omitempty" bson:"_id"`
	Name        string             `json:"name" bson:"name"`
	Title       string             `json:"title" bson:"title"`
	ImageURL    string             `json:"image_url" bson:"image_url"`
	EventId     primitive.ObjectID `json:"event_id" bson:"event_id"`
	Description string             `json:"description" bson:"description"`
	AchievedAt  int64              `json:"achieved_at,omitempty" bson:"achieved_at,omitempty"`
}
