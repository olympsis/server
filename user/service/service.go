package service

import (
	"context"
	"encoding/json"
	"net/http"
	"olympsis-server/database"
	"time"

	"github.com/olympsis/notif"

	"github.com/gorilla/mux"
	"github.com/olympsis/models"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

	Notif *notif.Service

	// mux Router to complete http requests
	Router *mux.Router
}

/*
Creates New Auth Service

  - Creates new instace of auth service object

Args:

	l - logrus logger (log info, errors or statuses)
	r - mux router (handle http requests)

Returns:

	*AuthenticationService - pointer referencing to new instance of service object
*/
func NewUserService(l *logrus.Logger, r *mux.Router, d *database.Database, n *notif.Service) *Service {
	return &Service{Log: l, Router: r, Database: d, Notif: n}
}

/*
Check User Name (GET)

  - Grab uuid from params

  - Grabs user data from database

Returns:

	Http handler
		- Writes user data back to client
*/
func (u *Service) CheckUsername() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// grab username from query
		keys, ok := r.URL.Query()["username"]
		if !ok || len(keys[0]) < 1 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "no userName found in request" }`))
			return
		}
		userName := keys[0]

		_, c := context.WithTimeout(context.Background(), 30*time.Second)
		defer c()

		// find user data in database
		var user models.User
		filter := bson.D{primitive.E{Key: "username", Value: userName}}
		err := u.Database.UserCol.FindOne(context.TODO(), filter).Decode(&user)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				rw.WriteHeader(http.StatusOK)
				rw.Write([]byte(`{ "is_available": true }`))
				return
			}
		}

		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{ "is_available": false }`))
	}
}

/*
Create User Data (PUT)

  - Creates new user for playfest (on sign up)

  - Grab request body

  - Create User data in user databse

Returns:

	Http handler
		- Writes token back to client
*/
func (u *Service) CreateUserData() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		uuid := r.Header.Get("UUID")

		// decode request
		var req models.User
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			u.Log.Error(err.Error())
			http.Error(rw, "Bad Request", http.StatusBadRequest)
			return
		}

		user := models.User{
			ID:         primitive.NewObjectID(),
			UUID:       uuid,
			UserName:   req.UserName,
			Sports:     req.Sports,
			Visibility: "public",
		}

		// insert auth user in database
		err = u.InsertUser(context.Background(), &user)
		if err != nil {
			u.Log.Error(err.Error())
			http.Error(rw, "Failed to Insert User", http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusCreated)
		json.NewEncoder(rw).Encode(user)
	}
}

/*
Update User Data (POST)

  - Updates user data

  - Grab parameters and update

Returns:

	Http handler
		- Writes token back to client
*/
func (u *Service) UpdateUserData() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		uuid := r.Header.Get("UUID")

		// decode request
		var req models.User
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			u.Log.Debug(err.Error())
			http.Error(rw, "Bad Request", http.StatusBadRequest)
		}

		filter := bson.M{"uuid": uuid}
		changes := bson.M{}
		if req.UserName != "" {
			changes["username"] = req.UserName
		}
		if req.ImageURL != "" {
			changes["image_url"] = req.ImageURL
		}
		if req.Bio != "" {
			changes["bio"] = req.Bio
		}
		if len(req.Sports) > 0 {
			changes["sports"] = req.Sports
		}
		if req.DeviceToken != "" {
			changes["device_token"] = req.DeviceToken
		}
		if req.Visibility != "" {
			changes["visibility"] = req.Visibility
		}

		update := bson.M{"$set": changes}

		err = u.UpdateUser(context.Background(), filter, update, &req)
		if err != nil {
			u.Log.Debug(err)
		}

		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(req)
	}
}

/*
Get User Data (GET)

  - Grab uuid from params

  - Grabs user data from database

Returns:

	Http handler
		- Writes user data back to client
*/
func (u *Service) GetUserData() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		uuid := r.Header.Get("UUID")

		// find user data in database
		var user models.User
		filter := bson.M{"uuid": uuid}
		err := u.FindUser(context.Background(), filter, &user)

		// username is a temp fix because empty users are not throwing an error
		if err != nil || user.UserName == "" {
			http.Error(rw, "user data not found", http.StatusNotFound)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(user)
	}
}

/*
Delete User Data (DELETE)

  - Delete User data

Returns:

	Http handler
		- Writes user data back to client
*/
func (u *Service) DeleteUserData() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		uuid := r.Header.Get("UUID")

		// delete user data from database
		filter := bson.M{"uuid": uuid}
		err := u.DeleteUser(context.Background(), filter)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				rw.WriteHeader(http.StatusNotFound)
				rw.Write([]byte(`{ "msg": "user not found" }`))
				return
			}
		}
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
	}
}

func (u *Service) GetOrganizationInvitations() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		uuid := r.Header.Get("UUID")

		filter := bson.M{
			"recipient": uuid,
			"status":    "pending",
		}

		var invitations []models.Invitation
		cursor, err := u.Database.OrgInvitationCol.Find(context.TODO(), filter)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			u.Log.Error("Failed to fetch invitations: " + err.Error())
			return
		}
		for cursor.Next(context.TODO()) {
			var invite models.Invitation
			err := cursor.Decode(&invite)
			if err != nil {
				u.Log.Error("Failed to decode invitation: " + err.Error())
			}
			var org models.Organization
			err = u.Database.OrgCol.FindOne(context.TODO(), bson.M{"_id": invite.SubjectID}).Decode(&org)
			if err != nil {
				u.Log.Error("Failed to fetch org data: " + err.Error())
			}
			invite.Data = &models.InvitationData{
				Organization: &org,
			}
			invitations = append(invitations, invite)
		}

		if len(invitations) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		response := models.InvitationsResponse{
			TotalInvitations: len(invitations),
			Invitations:      invitations,
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

func (u *Service) SearchUsersByUserName() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// grab username from query
		keys, ok := r.URL.Query()["username"]
		if !ok || len(keys[0]) < 1 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{ "msg": "no userName found in request" }`))
			return
		}
		userName := keys[0]

		// fetch users that might be related data
		var users []models.UserData
		regex := primitive.Regex{Pattern: userName, Options: "i"}
		filter := bson.M{"username": regex}
		cur, err := u.Database.UserCol.Find(context.TODO(), filter)
		if err != nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		for cur.Next(context.TODO()) {
			var meta models.User
			var data models.UserData
			err := cur.Decode(&meta)
			if err != nil {
				u.Log.Error("Failed to decode user data: " + err.Error())
			}

			data.Bio = meta.Bio
			data.UUID = meta.UUID
			data.Username = meta.UserName
			data.ImageURL = meta.ImageURL
			data.Visibility = meta.Visibility
			data.DeviceToken = meta.DeviceToken

			if data.Visibility == "public" {
				data.Clubs = meta.Clubs
				data.Sports = meta.Sports
				data.Organizations = meta.Organizations
			}
			users = append(users, data)
		}

		// fetch first and last name
		for i := range users {
			var auth models.AuthUser
			err := u.Database.AuthCol.FindOne(context.TODO(), bson.M{"uuid": users[i].UUID}).Decode(&auth)
			if err != nil {
				u.Log.Error("Failed to decode user auth data: " + err.Error())
			} else {
				users[i].FirstName = auth.FirstName
				users[i].LastName = auth.LastName
			}
		}

		response := models.UsersDataResponse{
			TotalUsers: len(users),
			Users:      users,
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

func (u *Service) SearchUserByUUID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// grab username from query
		keys, ok := r.URL.Query()["uuid"]
		if !ok || len(keys[0]) < 1 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{ "msg": "no uuid found in request" }`))
			return
		}
		uuid := keys[0]

		// context/filter
		ctx := context.Background()
		filter := bson.M{"uuid": uuid}
		opts := options.FindOneOptions{}

		// find and decode auth user data
		var auth models.AuthUser
		err := u.Database.AuthCol.FindOne(ctx, filter).Decode(&auth)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
		}

		// find and decode user metadata
		var user models.User
		err = u.Database.UserCol.FindOne(ctx, filter, &opts).Decode(&user)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
		}

		// create user data object
		userData := models.UserData{
			UUID:        user.UUID,
			Bio:         user.Bio,
			Username:    user.UserName,
			FirstName:   auth.FirstName,
			LastName:    auth.LastName,
			ImageURL:    user.ImageURL,
			Visibility:  user.Visibility,
			DeviceToken: user.DeviceToken,
		}

		// if user visibility is public display this data if not then dont
		if user.Visibility == "public" {
			userData.Clubs = user.Clubs
			userData.Sports = user.Sports
			userData.Organizations = user.Organizations
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(userData)
	}
}
