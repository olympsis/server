package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"olympsis-server/aggregations"
	"olympsis-server/server"
	"time"

	"github.com/google/uuid"
	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func NewAuthService(serverInterface *server.ServerInterface) *Service {
	return &Service{
		Log:      serverInterface.Logger,
		Router:   serverInterface.Router,
		Database: serverInterface.Database,
		Client:   serverInterface.Auth,
	}
}

func (a *Service) Register() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := context.TODO()

		// Decode request
		var request models.AuthRequest
		err := json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			a.Log.Error(fmt.Sprintf("[Auth] Failed to decode AuthRequest object: %s\n", err.Error()))
			http.Error(w, `{ "msg": "bad body in request" }`, http.StatusBadRequest)
			return
		}

		// Verify user token
		if request.Token == nil {
			http.Error(w, `{ "msg": "missing token in request" }`, http.StatusBadRequest)
			return
		}
		token, err := a.Client.VerifyIDToken(ctx, *request.Token)
		if err != nil {
			a.Log.Error(fmt.Sprintf("[Auth] Failed to verify ID Token: %s\n", err.Error()))
			http.Error(w, `{ "msg": "failed to register new user" }`, http.StatusInternalServerError)
			return
		}

		// Check for duplicates
		existing, err := a.FindUser(ctx, bson.M{"user_id": token.UID})
		if err == nil {
			if existing != nil {
				response := models.AuthResponse{
					UserID:    existing.UserID,
					FirstName: existing.FirstName,
					LastName:  existing.LastName,
					Email:     existing.Email,
				}

				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(response)
				return
			}
		}

		if err != mongo.ErrNoDocuments {
			a.Log.Error("[Auth] Failed to check user data. Error: ", err.Error())
			http.Error(w, `{"msg": "something went wrong."}`, http.StatusInternalServerError)
			return
		}

		// New AuthUser
		timestamp := bson.NewDateTimeFromTime(time.Now())
		user := &models.AuthUserDao{
			UserID:    &token.UID,
			FirstName: request.FirstName,
			LastName:  request.LastName,
			Email:     request.Email,
			CreatedAt: &timestamp,
		}

		// Insert AuthUser into database
		err = a.InsertUser(ctx, user)
		if err != nil {
			a.Log.Error(fmt.Sprintf("[Auth] Failed to insert user into the database: %s\n", err.Error()))
			http.Error(w, `{ "msg": "failed to create auth user" }`, http.StatusInternalServerError)
			return
		}

		// User Metadata
		tempUsername := "olympsis-user-" + uuid.NewString()
		hasOnboarded := false
		visibility := "public"
		meta := models.User{
			ID:           bson.NewObjectID(),
			UserID:       token.UID,
			UserName:     tempUsername,
			Visibility:   visibility,
			HasOnboarded: hasOnboarded,
		}

		// Insert metadata into database
		_, err = a.Database.UserCollection.InsertOne(ctx, meta)
		if err != nil {
			a.Log.Error(fmt.Sprintf("[Auth] Failed to insert user into the database: %s\n", err.Error()))
			http.Error(w, `{ "msg": "failed to create user" }`, http.StatusInternalServerError)
			return
		}

		// Safely dereference pointer fields — these may be nil if the client
		// omitted them (e.g. Apple Sign In without a last name).
		response := models.AuthResponse{
			UserID: *user.UserID,
		}
		if user.FirstName != nil {
			response.FirstName = *user.FirstName
		}
		if user.LastName != nil {
			response.LastName = *user.LastName
		}
		if user.Email != nil {
			response.Email = *user.Email
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

func (a *Service) Login() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var request models.AuthRequest
		err := json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			a.Log.Error(fmt.Sprintf("[Auth] Failed to decode AuthRequest: %s\n", err.Error()))
			http.Error(w, `{ "msg": "bad body in request"} `, http.StatusBadRequest)
			return
		}

		// Validate token
		if request.Token == nil {
			a.Log.Error(fmt.Sprintf("[Auth] Failed to decode AuthRequest: %s\n", err.Error()))
			http.Error(w, `{ "msg": "no token found in request"} `, http.StatusBadRequest)
		}

		// Verify auth token with firebase
		token, err := a.Client.VerifyIDToken(context.TODO(), *request.Token)
		if err != nil {
			a.Log.Error(fmt.Sprintf("[Auth] Failed to verify ID Token: %s\n", err.Error()))
			http.Error(w, `{ "msg": "failed to register new user" }`, http.StatusInternalServerError)
			return
		}

		user, err := aggregations.AggregateUser(&token.UID, a.Database)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				a.Log.Error(fmt.Sprintf("[Auth] User data not found for uid: %s\n", token.UID))
				http.Error(w, `{ "msg": "user data not found" }`, http.StatusNotFound)
				return
			}
			a.Log.Error(fmt.Sprintf("[Auth] Failed to find user data: %s\n", err.Error()))
			http.Error(w, `{ "msg": "failed to find user data" }`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(user)

	}
}

func (s *Service) Modify() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx, cancel := context.WithTimeout(r.Context(), time.Second*5)
		defer cancel()

		var request models.AuthUserDao
		uuid := r.Header.Get("userID")

		err := json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			http.Error(w, `{"msg": "failed to decode request"}`, http.StatusBadRequest)
			s.Log.Error("[Auth] Failed to decode request. Error: ", err.Error())
			return
		}

		changes := bson.M{}
		if request.FirstName != nil {
			changes["first_name"] = request.FirstName
		}

		if request.LastName != nil {
			changes["last_name"] = request.LastName
		}

		if request.Email != nil {
			changes["email"] = request.Email
		}

		user, err := s.UpdateUser(ctx, uuid, changes)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				http.Error(w, `{"msg": "user not found."}`, http.StatusNotFound)
				s.Log.Error("[Auth] failed to update user. Document not found: ", err.Error())
				return
			}

			http.Error(w, `{"msg": "failed to update user."}`, http.StatusInternalServerError)
			s.Log.Error("[Auth] failed to update user. Error: ", err.Error())
			return
		}

		if user == nil {
			http.Error(w, `{"msg": "something went wrong."}`, http.StatusInternalServerError)
			s.Log.Error("[Auth] Failed to get user data after update.")
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(user)
	}
}

func (a *Service) Delete() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// DELETE USER FROM DATABASE
		uuid := r.Header.Get("userID")
		err := a.DeleteUser(context.Background(), bson.M{"user_id": uuid})
		if err != nil {
			if err == mongo.ErrNoDocuments {
				a.Log.Error(fmt.Sprintf("[Auth] Failed to find user data: %s\n", err.Error()))
				http.Error(rw, `{ "msg": "user data not found" }`, http.StatusNotFound)
				return
			}
			a.Log.Error(fmt.Sprintf("[Auth] Failed to delete user data: %s\n", err.Error()))
			http.Error(rw, `{ "msg": "failed to delete user data" }`, http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{"msg": "OK"}`))
	}
}
