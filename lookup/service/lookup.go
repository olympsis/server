package service

import (
	"context"
	"encoding/json"
	"net/http"
	"olympsis-server/database"
	"os"
	"sync"

	"github.com/golang-jwt/jwt"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
)

/*
Create new lookup service struct
*/
func NewLookupService(l *logrus.Logger, r *mux.Router, d *database.Database) *Service {
	return &Service{Logger: l, Router: r, Database: d}
}

/*
Get a Club (GET)

  - Fetches and returns a club object

  - Grab path values

    Returns:
    Http handler

  - Writes a club object back to client
*/
func (l *Service) LookUpUserById() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		if len(vars["id"]) < 24 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "no uuid found in request" }`))
			return
		}
		uuid := vars["id"]

		var auth AuthUser
		var user UserData
		var look LookUpUser

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			auth = l.GetAuthData(uuid)
			defer wg.Done()
		}()

		wg.Add(1)
		go func() {
			user = l.GetUserData(uuid)
			defer wg.Done()
		}()
		wg.Wait()

		if user.IsPublic {
			look = LookUpUser{
				ID:          user.UUID,
				FirstName:   auth.FirstName,
				LastName:    auth.LastName,
				Username:    user.UserName,
				ImageURL:    user.ImageURL,
				Bio:         user.Bio,
				Clubs:       user.Clubs,
				Sports:      user.Sports,
				Badges:      user.Badges,
				Trophies:    user.Trophies,
				Friends:     user.Friends,
				DeviceToken: user.DeviceToken,
			}
		} else {
			look = LookUpUser{
				ID:          user.UUID,
				FirstName:   auth.FirstName,
				LastName:    auth.LastName,
				Username:    user.UserName,
				ImageURL:    user.ImageURL,
				Bio:         user.Bio,
				DeviceToken: user.DeviceToken,
			}
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(look)

	}
}

func (l *Service) LookUpUserUsername() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		if len(vars["username"]) < 5 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "no username found in request" }`))
			return
		}
		u := vars["username"]

		var auth AuthUser
		var user UserData
		var look LookUpUser

		user = l.GetUserDataByUsername(u)
		auth = l.GetAuthData(user.UUID)

		if user.IsPublic {
			look = LookUpUser{
				ID:          user.UUID,
				FirstName:   auth.FirstName,
				LastName:    auth.LastName,
				Username:    user.UserName,
				ImageURL:    user.ImageURL,
				Bio:         user.Bio,
				Clubs:       user.Clubs,
				Sports:      user.Sports,
				Badges:      user.Badges,
				Trophies:    user.Trophies,
				Friends:     user.Friends,
				DeviceToken: user.DeviceToken,
			}
		} else {
			look = LookUpUser{
				ID:          user.UUID,
				FirstName:   auth.FirstName,
				LastName:    auth.LastName,
				Username:    user.UserName,
				ImageURL:    user.ImageURL,
				Bio:         user.Bio,
				DeviceToken: user.DeviceToken,
			}
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(look)

	}
}

func (l *Service) BatchLookupById() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		var req BatchRequest
		json.NewDecoder(r.Body).Decode(&req)

		var resp []LookUpUser

		for i := 0; i < len(req.UUIDS); i++ {
			var auth AuthUser
			var user UserData
			var look LookUpUser

			var wg sync.WaitGroup

			wg.Add(1)
			go func() {
				auth = l.GetAuthData(req.UUIDS[i])
				defer wg.Done()
			}()

			wg.Add(1)
			go func() {
				user = l.GetUserData(req.UUIDS[i])
				defer wg.Done()
			}()
			wg.Wait()

			if user.IsPublic {
				look = LookUpUser{
					ID:          user.UUID,
					FirstName:   auth.FirstName,
					LastName:    auth.LastName,
					Username:    user.UserName,
					ImageURL:    user.ImageURL,
					Bio:         user.Bio,
					Clubs:       user.Clubs,
					Sports:      user.Sports,
					Badges:      user.Badges,
					Trophies:    user.Trophies,
					Friends:     user.Friends,
					DeviceToken: user.DeviceToken,
				}
			} else {
				look = LookUpUser{
					ID:          user.UUID,
					FirstName:   auth.FirstName,
					LastName:    auth.LastName,
					Username:    user.UserName,
					ImageURL:    user.ImageURL,
					Bio:         user.Bio,
					DeviceToken: user.DeviceToken,
				}
			}

			resp = append(resp, look)
		}
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(resp)
	}
}

/*
Get Auth Data

  - fetches the auth data from database

  - UUID param of user to lookup

Returns:

  - return user data object
*/
func (l *Service) GetAuthData(uuid string) AuthUser {
	filter := bson.M{"uuid": uuid}

	var userAuth AuthUser

	err := l.Database.AuthCol.FindOne(context.TODO(), filter).Decode(&userAuth)
	if err != nil {
		l.Logger.Error(err)
	}
	return userAuth
}

/*
Get User Data

  - fetches the user data from database

  - UUID param of user to lookup

Returns:

  - return user data object
*/
func (l *Service) GetUserData(uuid string) UserData {
	filter := bson.M{"uuid": uuid}

	var userData UserData

	err := l.Database.UserCol.FindOne(context.TODO(), filter).Decode(&userData)
	if err != nil {
		l.Logger.Error(err)
	}
	return userData
}

/*
Get User Data

  - fetches the user data from database

  - UUID param of user to lookup

Returns:

  - return user data object
*/
func (l *Service) GetUserDataByUsername(username string) UserData {
	filter := bson.M{"username": username}

	var userData UserData

	err := l.Database.UserCol.FindOne(context.TODO(), filter).Decode(&userData)
	if err != nil {
		l.Logger.Error(err)
	}
	return userData
}

/*
Validate an Parse JWT Token

  - parse jwt token

  - return values

Returns:

	uuid - string of the user id token
	createdAt - string of the session token created date
	role - role of user
	error -  if there is an error return error else nil
*/
func (l *Service) ValidateAndParseJWTToken(tokenString string) (string, string, float64, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("KEY")), nil
	})

	if err != nil {
		return "", "", 0, err
	} else {
		uuid := claims["uuid"].(string)
		provider := claims["provider"].(string)
		createdAt := claims["createdAt"].(float64)
		return uuid, provider, createdAt, nil
	}
}
