package service

import (
	"olympsis-server/database"
	lService "olympsis-server/lookup/service"
	"olympsis-server/pushnote/service"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Service struct {
	Database      *database.Database
	Logger        *logrus.Logger
	Router        *mux.Router
	NotifService  *service.Service
	LookUpService *lService.Service
}

type Club struct {
	ID          primitive.ObjectID `json:"id" bson:"_id"`
	Name        string             `json:"name" bson:"name"`
	Description string             `json:"description,omitempty" bson:"description,omitempty"`
	Sport       string             `json:"sport" bson:"sport"`
	City        string             `json:"city" bson:"city"`
	State       string             `json:"state" bson:"state"`
	Country     string             `json:"country" bson:"country"`
	ImageURL    string             `json:"imageURL,omitempty" bson:"imageURL,omitempty"`
	IsPrivate   bool               `json:"isPrivate" bson:"isPrivate"`
	Members     []Member           `json:"members" bson:"members"`
	Rules       []string           `json:"rules,omitempty" bson:"rules,omitempty"`
	CreatedAt   int64              `json:"createdAt" bson:"createdAt"`
}

type Member struct {
	ID       primitive.ObjectID   `json:"id" bson:"_id"`
	UUID     string               `json:"uuid" bson:"uuid"`
	Role     string               `json:"role" bson:"role"`
	Data     *lService.LookUpUser `json:"data,omitempty" bson:"data,omitempty"`
	JoinedAt int64                `json:"joinedAt" bson:"joinedAt"`
}

type CreateClubResponse struct {
	Token string `json:"token"`
	Club  Club   `json:"club"`
}

type ClubsResponse struct {
	TotalClubs int    `json:"totalClubs"`
	Clubs      []Club `json:"clubs"`
}

type ClubApplication struct {
	ID        primitive.ObjectID   `json:"id" bson:"_id"`
	UUID      string               `json:"uuid" bson:"uuid"`
	ClubId    primitive.ObjectID   `json:"clubId,omitempty" bson:"clubId,omitempty"`
	Data      *lService.LookUpUser `json:"data,omitempty" bson:"data,omitempty"`
	Status    string               `json:"status" bson:"status"`
	CreatedAt int64                `json:"createdAt" bson:"createdAt"`
}

type ClubInvitation struct {
	ID        primitive.ObjectID `json:"_id" bson:"_id"`
	UUID      string             `json:"uuid" bson:"uuid"`
	User      LookUpUser         `json:"user" bson:"user,omitempty"`
	ClubId    primitive.ObjectID `json:"clubId" bson:"clubId"`
	Status    string             `json:"status" bson:"status"`
	CreatedAt int64              `json:"createdAt" bson:"createdAt"`
}

type ApplicationUpdateRequest struct {
	Status string `json:"status"`
}

type ClubApplicationsResponse struct {
	TotalApplications int               `json:"totalApplications"`
	Applications      []ClubApplication `json:"applications"`
}

type ChangeRoleRequest struct {
	Role string `json:"role"`
}

// MIGHT REWORK
// USER LOOKUP

/*
Lookup User
- contains identifiable user data that others can see
*/
type LookUpUser struct {
	FirstName   string   `json:"firstName,omitempty" bson:"firstName,omitempty"`
	LastName    string   `json:"lastName,omitempty" bson:"lastName,omitempty"`
	Username    string   `json:"username,omitempty" bson:"username,omitempty"`
	ImageURL    string   `json:"imageURL,omitempty" bson:"imageURL,omitempty"`
	Clubs       []string `json:"clubs,omitempty" bson:"clubs,omitempty"`
	Sports      []string `json:"sports,omitempty" bson:"sports,omitempty"`
	Badges      []Badge  `json:"badges,omitempty" bson:"badges,omitempty"`
	Trophies    []Trophy `json:"trophies,omitempty" bson:"trophies,omitempty"`
	Friends     []Friend `json:"friends,omitempty" bson:"friends,omitempty"`
	DeviceToken string   `json:"deviceToken,omitempty" bson:"deviceToken,omitempty"`
}

/*
Trophy
  - Trophy object
*/
type Trophy struct {
	ID          primitive.ObjectID `json:"_id" bson:"_id"`
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

/*
Friend
  - Friend object
*/
type Friend struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	UUID      string             `json:"uuid" bson:"uuid"`
	CreatedAt int64              `json:"createdAt" bson:"createdAt"`
}

type NotificationRequest struct {
	Tokens []string `json:"tokens"`
	Title  string   `json:"title"`
	Body   string   `json:"body"`
	Topic  string   `json:"topic"`
}
