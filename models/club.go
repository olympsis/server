package models

import "go.mongodb.org/mongo-driver/bson/primitive"

/*
Club
  - Club object
*/
type Club struct {
	ID          primitive.ObjectID `json:"id,omitempty" bson:"_id"`
	Name        string             `json:"name,omitempty" bson:"name"`
	Description string             `json:"description,omitempty" bson:"description,omitempty"`
	Sport       string             `json:"sport,omitempty" bson:"sport"`
	City        string             `json:"city,omitempty" bson:"city"`
	State       string             `json:"state,omitempty" bson:"state"`
	Country     string             `json:"country,omitempty" bson:"country"`
	ImageURL    string             `json:"image_url,omitempty" bson:"image_url"`
	Visibility  string             `json:"visibility,omitempty" bson:"visibility"`
	Members     []Member           `json:"members,omitempty" bson:"members"`
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

type ClubResponse struct {
	Token string `json:"token,omitempty"`
	Club  Club   `json:"club"`
}

type ClubsResponse struct {
	TotalClubs int    `json:"total_clubs"`
	Clubs      []Club `json:"clubs"`
}

type ClubInvite struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	UUID      string             `json:"uuid" bson:"uuid"`
	ClubID    string             `json:"club_id" bson:"club_id"`
	Data      *ClubInviteData    `json:"data,omitempty" bson:"data,omitempty"`
	Status    string             `json:"status" bson:"status"`
	CreatedAt int64              `json:"created_at" bson:"created_at"`
}

type ClubInviteData struct {
	Club    *Club     `json:"club,omitempty"`
	Inviter *UserData `json:"inviter,omitempty"`
}

type ClubInvitesResponse struct {
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
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	UUID      string             `json:"uuid" bson:"uuid"`
	ClubID    primitive.ObjectID `json:"club_id" bson:"club_id"`
	Status    string             `json:"status" bson:"status"`
	Data      *UserData          `json:"data,omitempty" bson:"data,omitempty"`
	CreatedAt int64              `json:"created_at" bson:"created_at"`
}

type ClubApplicationsResponse struct {
	TotalApplications int               `json:"total_applications"`
	Applications      []ClubApplication `json:"club_applications"`
}
