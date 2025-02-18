package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"olympsis-server/aggregations"
	"olympsis-server/database"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/olympsis/models"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

/*
Creates New Auth Service

  - Creates new instance of auth service object

Args:

	l - logrus logger (log info, errors or statuses)
	r - mux router (handle http requests)

Returns:

	*AuthenticationService - pointer referencing to new instance of service object
*/
func NewUserService(l *logrus.Logger, r *mux.Router, d *database.Database) *Service {
	return &Service{Log: l, Router: r, Database: d}
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

  - Creates new user for olympsis (on sign up)

  - Grab request body

  - Create User data in user database

Returns:

	Http handler
		- Writes token back to client
*/
func (s *Service) CreateUserData() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		uuid := r.Header.Get("UUID")

		// decode request
		var req models.User
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			s.Log.Error(fmt.Sprintf("Failed to decode request: %s\n", err.Error()))
			http.Error(w, `{ "msg": "failed to decode request" }`, http.StatusBadRequest)
			return
		}

		user := models.User{
			ID:           primitive.NewObjectID(),
			UUID:         uuid,
			UserName:     req.UserName,
			Sports:       req.Sports,
			Visibility:   "public",
			HasOnboarded: req.HasOnboarded,
		}

		// insert auth user in database
		err = s.InsertUser(context.Background(), &user)
		if err != nil {
			s.Log.Error(fmt.Sprintf("Failed to insert user into database: %s\n", err.Error()))
			http.Error(w, `{ "msg": "failed to insert user"}`, http.StatusInternalServerError)
			return
		}

		usr, err := aggregations.AggregateUser(&uuid, s.Database)
		if err != nil {
			s.Log.Error(fmt.Sprintf("Failed to find user data: %s\n", err.Error()))
			http.Error(w, `{ "msg": "failed to find user data"}`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(usr)
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
func (s *Service) UpdateUserData() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		uuid := r.Header.Get("UUID")

		// decode request
		var req models.UserDao
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			s.Log.Error(fmt.Sprintf("Failed to decode request: %s\n", err.Error()))
			http.Error(w, `{ "msg": "failed to decode request" }`, http.StatusBadRequest)
			return
		}

		filter := bson.M{"uuid": uuid}
		changes := bson.M{}
		if req.UserName != nil {
			changes["username"] = req.UserName
		}
		if req.ImageURL != nil {
			changes["image_url"] = req.ImageURL
		}
		if req.Bio != nil {
			changes["bio"] = req.Bio
		}
		if req.Sports != nil && len(*req.Sports) > 0 {
			changes["sports"] = req.Sports
		}
		if req.DeviceTokens != nil {
			changes["device_tokens"] = req.DeviceTokens
		}
		if req.Visibility != nil {
			changes["visibility"] = req.Visibility
		}
		if req.AcceptedEULA != nil {
			changes["accepted_eula"] = req.AcceptedEULA
		}
		if req.HasOnboarded != nil {
			changes["has_onboarded"] = req.HasOnboarded
		}
		if req.Hometown != nil {
			changes["hometown"] = req.Hometown
		}
		if req.LastLocation != nil {
			changes["last_location"] = req.LastLocation
		}
		if req.BlockedUsers != nil {
			changes["blocked_users"] = req.BlockedUsers
		}
		if req.NotificationDevices != nil {
			changes["notification_devices"] = req.NotificationDevices
		}
		if req.NotificationPreference != nil {
			changes["notification_preference"] = req.NotificationPreference
		}

		update := bson.M{"$set": changes}

		err = s.UpdateUser(context.Background(), filter, update, &req)
		if err != nil {
			s.Log.Error(fmt.Sprintf("Failed to update user data: %s\n", err.Error()))
			http.Error(w, `{ "msg": "failed to update user data" }`, http.StatusInternalServerError)
			return
		}

		user, err := aggregations.AggregateUser(&uuid, s.Database)
		if err != nil {
			s.Log.Error(fmt.Sprintf("Failed to find user data: %s\n", err.Error()))
			http.Error(w, `{ "msg": "failed to find user data" }`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(user)
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
func (s *Service) GetUserData() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		uuid := r.Header.Get("UUID")

		user, err := aggregations.AggregateUser(&uuid, s.Database)
		if err != nil || user.Username == "" {
			s.Log.Error(fmt.Sprintf("Failed to find user data: %s\n", err.Error()))
			http.Error(w, `{ "msg": "failed to find user data" }`, http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(user)
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
			if meta.ImageURL != nil {
				data.ImageURL = *meta.ImageURL
			}
			data.Visibility = meta.Visibility
			data.DeviceTokens = meta.DeviceTokens

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
				users[i].FirstName = *auth.FirstName
				users[i].LastName = *auth.LastName
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

		imageURL := ""
		if user.ImageURL != nil {
			imageURL = *user.ImageURL
		}

		// create user data object
		userData := models.UserData{
			UUID:         user.UUID,
			Bio:          user.Bio,
			Username:     user.UserName,
			FirstName:    *auth.FirstName,
			LastName:     *auth.LastName,
			ImageURL:     imageURL,
			Visibility:   user.Visibility,
			DeviceTokens: user.DeviceTokens,
		}

		// if user visibility is public display this data if not then don't
		if user.Visibility == "public" {
			userData.Clubs = user.Clubs
			userData.Sports = user.Sports
			userData.Organizations = user.Organizations
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(userData)
	}
}

func (s *Service) CheckIn() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		uuid := r.Header.Get("UUID")
		response := models.CheckIn{}

		// find user data
		user, err := aggregations.AggregateUser(&uuid, s.Database)
		if err != nil {
			s.Log.Error(fmt.Sprintf("Failed to check user in: %s\n", err.Error()))
			http.Error(w, `{ "msg": "failed to check user in" }`, http.StatusInternalServerError)
			return
		}
		if user == nil {
			s.Log.Error("Failed to get user. user object is nill")
			http.Error(w, `{ "msg": "failed to get user" }`, http.StatusNotFound)
			return
		}
		response.User = *user
		var wg sync.WaitGroup

		if user.Clubs != nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				filter := bson.M{
					"$match": bson.M{
						"_id": bson.M{
							"$in": user.Clubs,
						},
					},
				}
				clubs, err := aggregations.AggregateClubs(filter, s.Database)
				if err != nil {
					s.Log.Error("failed to check user in: ", err.Error())
				}
				response.Clubs = clubs
			}()
		}

		if user.Organizations != nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				filter := bson.M{
					"$match": bson.M{
						"_id": bson.M{
							"$in": user.Organizations,
						},
					},
				}
				orgs, err := aggregations.AggregateOrganizations(filter, s.Database)
				if err != nil {
					s.Log.Error("failed to check user in: ", err.Error())
				}
				response.Organizations = orgs
			}()
		}

		wg.Wait()

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}
