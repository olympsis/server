package service

import (
	"context"
	"encoding/json"
	"net/http"
	"olympsis-server/auth/apple"
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
Authentication Service
  - reference object for auth service
*/
type Service struct {
	// mongodb Client
	Database *database.Database

	// logrus logger to Log information about service and errors
	Log *logrus.Logger

	// mux Router to complete http requests
	Router *mux.Router
}

type Notification struct {
	Payload string `json:"payload"`
}

/*
Apple Public Key
- Public key from apple to confirm jwt token
*/
type ApplePublicKey struct {
	KTY string `json:"kty"`
	KID string `json:"kid"`
	USE string `json:"use"`
	ALG string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

/*
Apple Public Keys
- list of public keys
*/
type ApplePublicKeys struct {
	Keys []ApplePublicKey `json:"keys"`
}

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
		var request models.AuthRequest

		err := json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			a.Log.Error(err.Error())
			http.Error(rw, "bad body", http.StatusBadRequest)
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
				rw.WriteHeader(http.StatusInternalServerError)
				return
			}

			uuid := uuid.New().String()
			token, err := utils.GenerateAuthToken(uuid, request.Provider)
			if err != nil {
				a.Log.Error(err.Error())
				rw.WriteHeader(http.StatusInternalServerError)
				return
			}

			user := models.AuthUser{
				UUID:        uuid,
				FirstName:   request.FirstName,
				LastName:    request.LastName,
				Email:       request.Email,
				Token:       token,
				AccessToken: resp.AccessToken,
				Provider:    request.Provider,
				CreatedAt:   time.Now().Unix(),
			}

			err = a.InsertUser(context.Background(), &user)
			if err != nil {
				a.Log.Error(err.Error())
				rw.WriteHeader(http.StatusInternalServerError)
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

		token, err := utils.GetTokenFromHeader(r)
		if err != nil {
			a.Log.Error(err.Error())
			http.Error(rw, "Forbidden", http.StatusForbidden)
			return
		}

		uuid, _, _, err := utils.ValidateAuthToken(token)
		if err != nil {
			a.Log.Error("Failed to Decode Token: " + err.Error())
			http.Error(rw, "Forbidden", http.StatusForbidden)
			return
		}

		// find user
		var user models.AuthUser
		filter := bson.M{"uuid": uuid}
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
