package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"olympsis-server/database"
	"os"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

/*
Creates New Auth Service

  - Creates new instace of auth service object

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

		// grab uuid from query
		keys, ok := r.URL.Query()["userName"]
		if !ok || len(keys[0]) < 1 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "no userName found in request" }`))
			return
		}
		userName := keys[0]

		_, c := context.WithTimeout(context.Background(), 30*time.Second)
		defer c()

		// find user data in database
		var user User
		filter := bson.D{primitive.E{Key: "userName", Value: userName}}
		err := u.Database.UserCol.FindOne(context.TODO(), filter).Decode(&user)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(http.StatusNotFound)
				rw.Write([]byte(`{ "isFound": false }`))
				return
			}
		}
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusFound)
		rw.Write([]byte(`{ "isFound": true }`))
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

		token, err := u.GrabToken(r)
		if err != nil {
			u.Log.Error(err.Error())
			http.Error(rw, "Unauthorized", http.StatusUnauthorized)
		}

		claims, err := u.DecodeToken(token)
		if err != nil {
			u.Log.Error("Failed to Decode Token: " + err.Error())
			http.Error(rw, "Forbidden", http.StatusForbidden)
			return
		}

		uuid := claims["sub"].(string)

		// decode request
		var req User
		err = json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			u.Log.Error(err.Error())
			http.Error(rw, "Bad Request", http.StatusBadRequest)
			return
		}

		user := User{
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

		token, err := u.GrabToken(r)
		if err != nil {
			u.Log.Error(err.Error())
			http.Error(rw, "Unauthorized", http.StatusUnauthorized)
		}

		claims, err := u.DecodeToken(token)
		if err != nil {
			u.Log.Error("Failed to Decode Token: " + err.Error())
			http.Error(rw, "Forbidden", http.StatusForbidden)
			return
		}

		uuid := claims["sub"].(string)

		// decode request
		var req User
		err = json.NewDecoder(r.Body).Decode(&req)
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
			changes["imageURL"] = req.ImageURL
		}
		if req.Bio != "" {
			changes["bio"] = req.Bio
		}
		if len(req.Sports) > 0 {
			changes["sports"] = req.Sports
		}
		if req.DeviceToken != "" {
			changes["deviceToken"] = req.DeviceToken
		}
		if req.Visibility != "" {
			changes["visibility"] = req.Visibility
		}

		update := bson.M{"$set": changes}

		err = u.UpdateUser(context.Background(), filter, update, &req)
		if err != nil {
			u.Log.Debug(err)
		}

		rw.Header().Set("Content-Type", "application/json")
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
		token, err := u.GrabToken(r)
		if err != nil {
			u.Log.Error(err.Error())
			http.Error(rw, "Unauthorized", http.StatusUnauthorized)
		}

		claims, err := u.DecodeToken(token)
		if err != nil {
			u.Log.Error("Failed to Decode Token: " + err.Error())
			http.Error(rw, "Forbidden", http.StatusForbidden)
			return
		}

		uuid := claims["sub"].(string)

		// find user data in database
		var user User
		filter := bson.M{"uuid": uuid}
		err = u.FindUser(context.Background(), filter, &user)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				rw.WriteHeader(http.StatusNotFound)
				rw.Write([]byte(`{ "msg": "user not found" }`))
				return
			}
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
		token, err := u.GrabToken(r)
		if err != nil {
			u.Log.Error(err.Error())
			http.Error(rw, "Unauthorized", http.StatusUnauthorized)
		}

		claims, err := u.DecodeToken(token)
		if err != nil {
			u.Log.Error("Failed to Decode Token: " + err.Error())
			http.Error(rw, "Forbidden", http.StatusForbidden)
			return
		}

		uuid := claims["sub"].(string)

		// delete user data from database
		filter := bson.M{"uuid": uuid}
		err = u.DeleteUser(context.Background(), filter)
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

/*
Peek User
  - Peeks at user and return data

Args:

	r - http request
	user - user id

Returns:

	LookUpUser - data on requested user
*/
func (u *Service) FetchUser(r http.Request, user string) LookUpUser {
	token, err := u.GrabToken(&r)
	if err != nil {
		u.Log.Error(err.Error())
	}
	client := &http.Client{}

	req, err := http.NewRequest("GET", "http://lookup.olympsis.internal/v1/lookup/"+user, nil)
	if err != nil {
		u.Log.Error(err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		u.Log.Error(err)
	}

	defer resp.Body.Close()

	var lookup LookUpUser
	err = json.NewDecoder(resp.Body).Decode(&lookup)
	if err != nil {
		u.Log.Error(err)
	}
	return lookup
}

/*
Send Notification
  - send a notification to user

Args:

	r - http request
	title - title of notification
	body - body of notification
	tokens - user tokens to notify
*/
func (u *Service) sendNotification(r http.Request, title string, body string, tokens []string) {
	token, err := u.GrabToken(&r)
	if err != nil {
		u.Log.Error(err.Error())
	}
	client := &http.Client{}

	request := NotificationRequest{
		Tokens: tokens,
		Title:  title,
		Body:   body,
	}

	data, err := json.Marshal(request)
	if err != nil {
		u.Log.Error(err.Error())
	}

	req, err := http.NewRequest("POST", "http://pushnote.olympsis.internal/v1/pushnote/device", bytes.NewBuffer(data))
	if err != nil {
		u.Log.Error(err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		u.Log.Error(err)
	}

	defer resp.Body.Close()
}

/*
Decode an Authentication Token
  - Decodes auth token
  - uses go jwt

Args:

	token - string of token

Returns:

	claims - jwt claims
	error -  if there is an error return error else nil
*/
func (u *Service) DecodeToken(token string) (jwt.MapClaims, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("KEY")), nil
	})

	if err != nil {
		return nil, err
	} else {
		return claims, nil
	}
}

/*
Grab Token from request
Args:

	r - http request

Returns:

	string - token
	error -  if there is an error return error else nil
*/
func (u *Service) GrabToken(r *http.Request) (string, error) {
	bearerToken := r.Header.Get("Authorization")

	if bearerToken == "" {
		return "", errors.New("no token found")
	}

	return bearerToken, nil
}
