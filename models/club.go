package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Club struct {
	ID          primitive.ObjectID `json:"id,omitempty" bson:"_id"`
	Name        string             `json:"name" bson:"name"`
	Description string             `json:"description,omitempty" bson:"description,omitempty"`
	Sport       string             `json:"sport" bson:"sport"`
	City        string             `json:"city" bson:"city"`
	State       string             `json:"state" bson:"state"`
	Country     string             `json:"country" bson:"country"`
	ImageURL    string             `json:"image_url" bson:"image_url"`
	Visibility  string             `json:"visibility" bson:"visibility"`
	Members     []Member           `json:"members" bson:"members"`
	Rules       []string           `json:"rules,omitempty" bson:"rules,omitempty"`
	CreatedAt   int64              `json:"created_at" bson:"created_at"`
}

type Member struct {
	ID       primitive.ObjectID `json:"id" bson:"_id"`
	UUID     string             `json:"uuid" bson:"uuid"`
	Role     string             `json:"role" bson:"role"`
	JoinedAt int64              `json:"joinedAt" bson:"joinedAt"`
}

type ClubInvite struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	UUID      string             `json:"uuid" bson:"uuid"`
	ClubId    string             `json:"clubId" bson:"clubId"`
	Status    string             `json:"status" bson:"status"`
	CreatedAt int64              `json:"createdAt" bson:"createdAt"`
}

/*
Friend Requests

  - total number of friend request

  - friend requests
*/
type ClubInvites struct {
	TotalInvites int          `json:"totalInvites"`
	Invites      []ClubInvite `json:"invites"`
}
