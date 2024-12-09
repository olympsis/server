package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"olympsis-server/aggregations"
	"olympsis-server/database"
	"time"

	"firebase.google.com/go/v4/auth"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/olympsis/models"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

/*
  - Creates new instance of auth service object

Args:

	l - logrus logger (log info, errors or statuses)
	r - mux router (handle http requests)

Returns:

	*AuthenticationService - pointer referencing to new instance of service object
*/
func NewAuthService(l *logrus.Logger, r *mux.Router, d *database.Database, c *auth.Client) *Service {
	return &Service{Log: l, Router: r, Database: d, Client: c}
}

/*
Create User (PUT)
  - Creates new user for olympsis (sign up)
  - Grab request body
  - Create AuthUser data in auth database

Returns:

	Http handler
		- Writes token back to client
*/
func (a *Service) Register() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := context.TODO()

		// Decode request
		var request models.AuthRequest
		err := json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			a.Log.Error(fmt.Sprintf("Failed to decode AuthRequest object: %s\n", err.Error()))
			http.Error(w, `{ "msg": "bad body in request" }`, http.StatusBadRequest)
			return
		}

		// Verify user token
		token, err := a.Client.VerifyIDToken(ctx, *request.Token)
		if err != nil {
			a.Log.Error(fmt.Sprintf("Failed to verify ID Token: %s\n", err.Error()))
			http.Error(w, `{ "msg": "failed to register new user" }`, http.StatusInternalServerError)
			return
		}

		// New AuthUser
		timestamp := time.Now().Unix()
		user := &models.AuthUser{
			UUID:      &token.UID,
			FirstName: request.FirstName,
			LastName:  request.LastName,
			Email:     request.Email,
			CreatedAt: &timestamp,
		}

		// Insert AuthUser into database
		err = a.InsertUser(ctx, user)
		if err != nil {
			a.Log.Error(fmt.Sprintf("Failed to insert user into the database: %s\n", err.Error()))
			http.Error(w, `{ "msg": "failed to add user to the database" }`, http.StatusInternalServerError)
			return
		}

		// User Metadata
		tempUsername := "olympsis-user-" + uuid.NewString()
		hasOnboarded := false
		acceptedEULA := true
		visibility := "public"
		meta := models.User{
			ID:           primitive.NewObjectID(),
			UUID:         token.UID,
			UserName:     tempUsername,
			Visibility:   visibility,
			HasOnboarded: &hasOnboarded,
			AcceptedEULA: &acceptedEULA,
		}

		// Insert metadata into database
		_, err = a.Database.UserCol.InsertOne(ctx, meta)
		if err != nil {
			a.Log.Error(fmt.Sprintf("Failed to insert user into the database: %s\n", err.Error()))
			http.Error(w, `{ "msg": "failed to add user to the database" }`, http.StatusInternalServerError)
			return
		}

		response := models.AuthResponse{
			UUID:      user.UUID,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Email:     user.Email,
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

/*
Login User (POST)
  - Logs user into olympsis
  - Grab token from header
  - Generate new JWT auth token
  - Update AuthUser data in auth database

Returns:

	Http handler
		- Writes token back to client
		- Writes userData back to client
*/
func (a *Service) Login() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var request models.AuthRequest
		err := json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			a.Log.Error(fmt.Sprintf("Failed to decode AuthRequest: %s\n", err.Error()))
			http.Error(w, `{ "msg": "bad body in request"} `, http.StatusBadRequest)
			return
		}

		token, err := a.Client.VerifyIDToken(context.TODO(), *request.Token)
		if err != nil {
			a.Log.Error(fmt.Sprintf("Failed to verify ID Token: %s\n", err.Error()))
			http.Error(w, `{ "msg": "failed to register new user" }`, http.StatusInternalServerError)
			return
		}

		uuid := token.UID

		user, err := aggregations.AggregateUser(&uuid, a.Database)
		if err != nil {
			a.Log.Error(fmt.Sprintf("Failed to find user data: %s\n", err.Error()))
			http.Error(w, `{ "msg": "failed to find user data" }`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(user)

	}
}

/*
Delete User (DELETE)
  - Deletes auth user from olympsis

Returns:

	Http handler
		- Writes bool whether sign out was successful
*/
func (a *Service) Delete() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		uuid := r.Header.Get("UUID")
		filter := bson.M{"uuid": uuid}

		// DELETE USER FROM DATABASE
		err := a.DeleteUser(context.Background(), filter)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				a.Log.Error(fmt.Sprintf("Failed to find user data: %s\n", err.Error()))
				http.Error(rw, `{ "msg": "user data not found" }`, http.StatusNotFound)
				return
			}
			a.Log.Error(fmt.Sprintf("Failed to delete user data: %s\n", err.Error()))
			http.Error(rw, `{ "msg": "failed to delete user data" }`, http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusOK)
	}
}
