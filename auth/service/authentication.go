package service

import (
	"context"
	"encoding/json"
	"net/http"
	"olympsis-server/auth/apple"
	"olympsis-server/auth/google"
	"olympsis-server/database"
	"olympsis-server/utils"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/olympsis/models"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

/*
  - Creates new instace of auth service object

Args:

	l - logrus logger (log info, errors or statuses)
	r - mux router (handle http requests)

Returns:

	*AuthenticationService - pointer referencing to new instance of service object
*/
func NewAuthService(l *logrus.Logger, r *mux.Router, d *database.Database) *Service {
	return &Service{Log: l, Router: r, Database: d}
}

/*
Create User (PUT)
  - Creates new user for olympsis (sign up)
  - Grab request body
  - Create AuthUser data in auth databse
  - Generate JWT auth token

Returns:

	Http handler
		- Writes token back to client
*/
func (a *Service) SignUp() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		/*
			Grab http request body
			If the process fails return bad request
		*/
		var request models.AuthRequest
		err := json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			a.Log.Error(err.Error())
			http.Error(rw, `{"error": "bad body in request"}`, http.StatusBadRequest)
			return
		}

		if request.Provider == "https://appleid.apple.com" {
			// Your 10-character Team ID
			teamID := "5A6H49Q85D"

			// ClientID is the "Services ID" value that you get when navigating to your "sign in with Apple"-enabled service ID
			clientID := "com.coronislabs.Olympsis"

			// Find the 10-char Key ID value from the portal
			keyID := "S3HDPU4ZC5"

			/*
				Read key file
				Generate client secret for verification
				If process fails return internal server error
			*/
			file, err := os.ReadFile("./auth/files/AuthKey_S3HDPU4ZC5.p8")
			if err != nil {
				a.Log.Error("Failed to find auth key file: " + err.Error())
				http.Error(rw, `{"error": "check logs"}`, http.StatusInternalServerError)
				return
			}
			secret, err := apple.GenerateClientSecret(file, teamID, clientID, keyID)
			if err != nil {
				a.Log.Error("Failed to generate secret: " + err.Error())
				http.Error(rw, `{"error": "check logs"}`, http.StatusInternalServerError)
				return
			}

			/*
				Create new apple validator client
				Create new token validation request
			*/
			client := apple.New()
			vReq := apple.AppValidationTokenRequest{
				ClientID:     clientID,
				ClientSecret: secret,
				Code:         request.Code,
			}

			/*
				Perform validation request
				Catch validation response and return error if process fails
			*/
			var resp apple.ValidationResponse
			err = client.VerifyAppToken(context.Background(), vReq, &resp)
			if err != nil {
				a.Log.Error("Failed to verify token: " + err.Error())
				http.Error(rw, `{"error": "token verification failed"}`, http.StatusUnauthorized)
				return
			}
			if resp.Error != "" {
				a.Log.Error("Failed to verify token: " + resp.Error + " - " + resp.ErrorDescription)
				http.Error(rw, `{"error": "token verification failed"}`, http.StatusInternalServerError)
				return
			}

			/*
				Generate new user data
				If process fails return internal server error
			*/
			user, err := a.CreateNewUserData(request.Provider, resp.AccessToken, request.FirstName, request.LastName, request.Email)
			if err != nil {
				a.Log.Error("Failed to create user data: " + err.Error())
				http.Error(rw, `{"error": "failed to create user data"}`, http.StatusInternalServerError)
				return
			}

			/*
				Insert user into the database
				Returns internal server error if process fails
			*/
			err = a.InsertUser(context.Background(), user)
			if err != nil {
				a.Log.Error("Failed to insert user into the database: " + err.Error())
				http.Error(rw, `{"error": "failed to add user to the databse"}`, http.StatusInternalServerError)
				return
			}

			rw.WriteHeader(http.StatusOK)
			json.NewEncoder(rw).Encode(user)

		} else if request.Provider == "https://accounts.google.com" {

			/*
				Create new google validator client
				Validate token
			*/
			client := google.NewClient()
			claims, err := client.ValidateJWT(request.Code)
			if err != nil {
				a.Log.Error("Failed to validate google token " + err.Error())
				http.Error(rw, `{"error": "failed to validate token"}`, http.StatusUnauthorized)
				return
			}

			/*
				Generate new user data
				If process fails return internal server error
			*/
			user, err := a.CreateNewUserData(claims["iss"].(string), claims["jti"].(string), claims["given_name"].(string), claims["family_name"].(string), claims["email"].(string))
			if err != nil {
				a.Log.Error("Failed creating user data: " + err.Error())
				http.Error(rw, `{"error": "failed to create user data"}`, http.StatusInternalServerError)
				return
			}

			/*
				Insert user into the database
				Returns internal server error if process fails
			*/
			err = a.InsertUser(context.Background(), user)
			if err != nil {
				a.Log.Error("Failed inserting user into the database: " + err.Error())
				http.Error(rw, `{"error": "failed to add user to the databse"}`, http.StatusInternalServerError)
				return
			}

			rw.WriteHeader(http.StatusOK)
			json.NewEncoder(rw).Encode(user)
		}
	}
}

/*
Login User (POST)
  - Logs user into olympsis
  - Grab token from header
  - Generate new JWT auth token
  - Update AuthUser data in auth databse

Returns:

	Http handler
		- Writes token back to client
		- Writes userData back to client
*/
func (a *Service) Login() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		var request models.AuthRequest

		err := json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "bad body in request" }`))
		}

		if request.Provider == "https://appleid.apple.com" {
			// Your 10-character Team ID
			teamID := "5A6H49Q85D"

			// ClientID is the "Services ID" value that you get when navigating to your "sign in with Apple"-enabled service ID
			clientID := "com.coronislabs.Olympsis"

			// Find the 10-char Key ID value from the portal
			keyID := "S3HDPU4ZC5"

			file, err := os.ReadFile("./auth/files/AuthKey_S3HDPU4ZC5.p8")
			if err != nil {
				a.Log.Error(err.Error())
				rw.WriteHeader(http.StatusInternalServerError)
				return
			}

			secret, err := apple.GenerateClientSecret(file, teamID, clientID, keyID)
			if err != nil {
				a.Log.Error("error generating secret: " + err.Error())
				rw.WriteHeader(http.StatusInternalServerError)
				return
			}

			// Generate a new validation client
			client := apple.New()

			vReq := apple.AppValidationTokenRequest{
				ClientID:     clientID,
				ClientSecret: secret,
				Code:         request.Code,
			}

			var resp apple.ValidationResponse

			// Do the verification
			err = client.VerifyAppToken(context.Background(), vReq, &resp)
			if err != nil {
				a.Log.Error("error verifying: " + err.Error())
				rw.WriteHeader(http.StatusInternalServerError)
				return
			}

			if resp.Error != "" {
				a.Log.Error(resp.Error)
				a.Log.Error(resp.ErrorDescription)
				rw.WriteHeader(http.StatusBadRequest)
				return
			}

			// get email
			claim, _ := apple.GetClaims(resp.IDToken)
			email := (*claim)["email"].(string)

			// find user
			var user models.AuthUser
			filter := bson.M{"email": email}
			err = a.FindUser(context.Background(), filter, &user)
			if err != nil {
				if err == mongo.ErrNoDocuments {
					a.Log.Error(err.Error())
					rw.WriteHeader(http.StatusNotFound)
					return
				}
				a.Log.Error(err.Error())
				rw.WriteHeader(http.StatusInternalServerError)
				return
			}

			// generate token for api
			token, err := utils.GenerateAuthToken(user.UUID, user.Provider)
			if err != nil {
				a.Log.Error(err.Error())
				rw.WriteHeader(http.StatusInternalServerError)
				return
			}

			// update tokens
			update := bson.M{"$set": bson.M{"token": token, "accessToken": resp.AccessToken}}
			err = a.UpdateUser(context.Background(), filter, update, &user)
			if err != nil {
				if err == mongo.ErrNoDocuments {
					a.Log.Error(err.Error())
					rw.WriteHeader(http.StatusNotFound)
					return
				}
				a.Log.Error(err.Error())
				rw.WriteHeader(http.StatusInternalServerError)
				return
			}

			rw.WriteHeader(http.StatusOK)
			json.NewEncoder(rw).Encode(user)
		}

	}
}

/*
Logout User (POST)
  - logout user out of olympsis

Returns:

	Http handler
		- Writes token back to client
		- Writes userData back to client
*/

func (a *Service) Logout() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

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

		// find user
		var user models.AuthUser
		filter := bson.M{"uuid": uuid}
		err := a.FindUser(context.Background(), filter, &user)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				a.Log.Error(err.Error())
				rw.WriteHeader(http.StatusNotFound)
				return
			}
			a.Log.Error(err.Error())
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Your 10-character Team ID
		teamID := "5A6H49Q85D"

		// ClientID is the "Services ID" value that you get when navigating to your "sign in with Apple"-enabled service ID
		clientID := "com.coronislabs.Olympsis"

		// Find the 10-char Key ID value from the portal
		keyID := "S3HDPU4ZC5"

		file, err := os.ReadFile("./auth/files/AuthKey_S3HDPU4ZC5.p8")
		if err != nil {
			a.Log.Error(err.Error())
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		secret, err := apple.GenerateClientSecret(file, teamID, clientID, keyID)
		if err != nil {
			a.Log.Error("error generating secret: " + err.Error())
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Generate a new validation client
		client := apple.New()

		// REVOKE ACCESS TOKEN
		vReq := apple.RevokeAccessTokenRequest{
			ClientID:     clientID,
			ClientSecret: secret,
			AccessToken:  user.AccessToken,
		}

		var resp apple.RevokeResponse

		// Revoke token request
		err = client.RevokeAccessToken(context.Background(), vReq, &resp)
		if err != nil {
			if err.Error() != "EOF" {
				a.Log.Error("error revoking: " + err.Error())
				return
			}
		}

		if resp.Error != "" {
			a.Log.Error(resp.Error)
			a.Log.Error(resp.ErrorDescription)
			return
		}

		// DELETE USER FROM DATABASE
		err = a.DeleteUser(context.Background(), filter)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				a.Log.Error(err.Error())
				rw.WriteHeader(http.StatusNotFound)
				return
			}
			a.Log.Error(err.Error())
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusOK)
	}
}

/*
Token (GET)
  - Returns new auth token

Returns:

	Http handler
		- Writes back token
*/
func (a *Service) Token() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		uuid := r.Header.Get("UUID")

		// fetch user data
		var user models.AuthUser
		filter := bson.M{"uuid": uuid}
		err := a.FindUser(context.Background(), filter, &user)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				a.Log.Error(err.Error())
				http.Error(rw, "Failed to get user", http.StatusNotFound)
				return
			}
			a.Log.Error(err.Error())
			http.Error(rw, "Unexpected Error", http.StatusInternalServerError)
			return
		}

		// generate new auth token
		token, err := utils.GenerateAuthToken(uuid, user.Provider)
		if err != nil {
			a.Log.Error(err.Error())
			http.Error(rw, "failed to create token", http.StatusInternalServerError)
			return
		}

		// update tokens
		update := bson.M{"$set": bson.M{"token": token}}
		err = a.UpdateUser(context.Background(), filter, update, &user)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				a.Log.Error(err.Error())
				http.Error(rw, "Failed to get user", http.StatusNotFound)
				return
			}
			a.Log.Error(err.Error())
			http.Error(rw, "Unexpected Error", http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{"authToken": "` + token + `"}`))
	}
}

/*
Apple Notification(POST)
  - listens to apple server updates

Returns:

	Http handler
		- Writes bool whether sign out was successful
*/
func (a *Service) AppleNotifications() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		var request Notification
		json.NewDecoder(r.Body).Decode(&request)

		a.Log.Info(request.Payload)
	}
}

func (a *Service) CreateNewUserData(provider string, accessToken string, firstName string, lastName string, email string) (*models.AuthUser, error) {
	uuid := uuid.New().String()
	token, err := utils.GenerateAuthToken(uuid, provider)
	if err != nil {
		return nil, err
	}

	return &models.AuthUser{
		UUID:        uuid,
		FirstName:   firstName,
		LastName:    lastName,
		Email:       email,
		Token:       token,
		AccessToken: accessToken,
		Provider:    provider,
		CreatedAt:   time.Now().Unix(),
	}, nil
}
