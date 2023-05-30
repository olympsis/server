package models

import "go.mongodb.org/mongo-driver/bson/primitive"

/*
Club
  - Club object
*/
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
	CreatedAt   int64              `json:"created_at,omitempty" bson:"created_at,omitempty"`
}

type Member struct {
	ID       primitive.ObjectID `json:"id,omitempty" bson:"_id"`
	UUID     string             `json:"uuid" bson:"uuid"`
	Role     string             `json:"role" bson:"role"`
	Data     *UserData          `json:"data,omitempty" bson:"data,omitempty"`
	JoinedAt int64              `json:"joined_at,omitempty" bson:"joined_at"`
}

type ClubInvite struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	UUID      string             `json:"uuid" bson:"uuid"`
	ClubID    string             `json:"club_id" bson:"club_id"`
	Status    string             `json:"status" bson:"status"`
	CreatedAt int64              `json:"created_at" bson:"created_at"`
}

type ClubsResponse struct {
	TotalClubs int    `json:"total_clubs"`
	Clubs      []Club `json:"clubs"`
}

type ClubInvites struct {
	TotalInvites int          `json:"total_invites"`
	Invites      []ClubInvite `json:"invites"`
}

type ChangeRoleRequest struct {
	Role string `json:"role"`
}

type CreateClubResponse struct {
	Token string `json:"token"`
	Club  Club   `json:"club"`
}

type ApplicationUpdateRequest struct {
	Status string `json:"status"`
}

type ClubApplication struct {
	ID        primitive.ObjectID `json:"id"`
	UUID      string             `json:"uuid"`
	ClubID    primitive.ObjectID `json:"club_id"`
	Status    string             `json:"status"`
	Data      *UserData          `json:"data,omitempty" bson:"data,omitempty"`
	CreatedAt int64              `json:"created_at"`
}

type ClubApplicationsResponse struct {
	TotalApplications int               `json:"total_applications"`
	Applications      []ClubApplication `json:"club_applications"`
}

type ClubInvitation struct {
	ID        primitive.ObjectID `json:"id"`
	UUID      string             `json:"uuid"`
	ClubID    primitive.ObjectID `json:"club_id"`
	Status    string             `json:"status"`
	Data      *Club              `json:"data,omitempty" bson:"data,omitempty"`
	CreatedAt int64              `json:"created_at"`
}
