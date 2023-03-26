package service

import (
	"olympsis-server/database"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/*
Authentication Service
- reference object for auth service
*/
type Service struct {
	// database
	Database *database.Database

	// logrus logger to Log information about service and errors
	Log *logrus.Logger

	// mux Router to complete http requests
	Router *mux.Router
}

// SERVICE OBJECTS

/*
User Data
- 	Contains user identifiable data
*/
type User struct {
	ID          primitive.ObjectID `json:"id" bson:"_id"`
	UUID        string             `json:"uuid" bson:"uuid"`
	UserName    string             `json:"username" bson:"username"`
	Bio         string             `json:"bio,omitempty" bson:"bio,omitempty"`
	ImageURL    string             `json:"imageURL,omitempty" bson:"imageURL,omitempty"`
	Visibility  string             `json:"visibility" bson:"visibility"`
	Clubs       []string           `json:"clubs,omitempty" bson:"clubs,omitempty"`
	Sports      []string           `json:"sports,omitempty" bson:"sports,omitempty"`
	Badges      []Badge            `json:"badges,omitempty" bson:"badges,omitempty"`
	Trophies    []Trophy           `json:"trophies,omitempty" bson:"trophies,omitempty"`
	Friends     []Friend           `json:"friends,omitempty" bson:"friends,omitempty"`
	DeviceToken string             `json:"deviceToken,omitempty" bson:"deviceToken,omitempty"`
}

/*
Friend
  - Friend object
*/
type Friend struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	UUID      string             `json:"uuid" bson:"uuid"`
	Data      LookUpUser         `json:"data,omitempty"`
	CreatedAt int64              `json:"createdAt" bson:"createdAt"`
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
	ID          primitive.ObjectID `json:"_id" bson:"_id"`
	Name        string             `json:"name" bson:"name"`
	Title       string             `json:"title" bson:"title"`
	ImageURL    string             `json:"imageURL" bson:"imageURL"`
	EventId     primitive.ObjectID `json:"eventId" bson:"eventId"`
	Description string             `json:"description" bson:"description"`
	AchievedAt  int64              `json:"achievedAt" bson:"achievedAt"`
}

/*
Friend Request

  - friend request object
*/
type FriendRequest struct {
	ID            primitive.ObjectID `json:"id" bson:"_id"`
	Requestor     string             `json:"requestor" bson:"requestor"`
	Requestee     string             `json:"requestee" bson:"requestee"`
	RequestorData LookUpUser         `json:"requestorData"`
	Status        string             `json:"status" bson:"status"`
	CreatedAt     int64              `json:"createdAt" bson:"createdAt"`
}

/*
Friend Requests

  - total number of friend request

  - friend requests
*/
type FriendRequests struct {
	TotalRequests int             `json:"totalRequests"`
	Requests      []FriendRequest `json:"requests"`
}

/*
New Friend Request

  - Request coming from client

  - just contains uuid of friend to request
*/
type NewFriendRequest struct {
	UUID string `json:"uuid" bson:"uuid"`
}

/*
	Club Invites
*/

type ClubInvite struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	UUID      string             `json:"uuid" bson:"uuid"`
	ClubId    string             `json:"clubId" bson:"clubId"`
	Status    string             `json:"status" bson:"status"`
	Data      Club               `json:"club,omitempty" bson:"club,omitempty"`
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

/*
Lookup User
- contains identifiable user data that others can see
*/
type LookUpUser struct {
	FirstName string   `json:"firstName" bson:"firstName"`
	LastName  string   `json:"lastName" bson:"lastName"`
	Username  string   `json:"username" bson:"username"`
	ImageURL  string   `json:"imageURL,omitempty" bson:"imageURL,omitempty"`
	Clubs     []string `json:"clubs,omitempty" bson:"clubs,omitempty"`
	Sports    []string `json:"sports,omitempty" bson:"sports,omitempty"`
	Badges    []Badge  `json:"badges,omitempty" bson:"badges,omitempty"`
	Trophies  []Trophy `json:"trophies,omitempty" bson:"trophies,omitempty"`
	Friends   []Friend `json:"friends,omitempty" bson:"friends,omitempty"`
}

type Club struct {
	ID          primitive.ObjectID `json:"_id" bson:"_id"`
	Name        string             `json:"name" bson:"name"`
	Description string             `json:"description" bson:"description"`
	Sport       string             `json:"sport" bson:"sport"`
	City        string             `json:"city" bson:"city"`
	State       string             `json:"state" bson:"state"`
	Country     string             `json:"country" bson:"country"`
	ImageURL    string             `json:"imageURL" bson:"imageURL"`
	IsPrivate   bool               `json:"isPrivate" bson:"isPrivate"`
	IsVisible   bool               `json:"isVisible" bson:"isVisible"`
	Members     []Member           `json:"members" bson:"members"`
	Rules       []string           `json:"rules" bson:"rules"`
	CreatedAt   int64              `json:"createdAt" bson:"createdAt"`
}

type Member struct {
	ID       primitive.ObjectID `json:"id" bson:"_id"`
	UUID     string             `json:"uuid" bson:"uuid"`
	Role     string             `json:"role" bson:"role"`
	JoinedAt int64              `json:"joinedAt" bson:"joinedAt"`
}

type NotificationRequest struct {
	Tokens []string `json:"tokens"`
	Title  string   `json:"title"`
	Body   string   `json:"body"`
	Topic  string   `json:"topic"`
}
