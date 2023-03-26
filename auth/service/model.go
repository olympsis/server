package service

import (
	"olympsis-server/database"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
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

/*
Sign In Request
  - Request comming from client
  - Contains user identifiable data
*/
type SignInRequest struct {
	FirstName string `json:"firstName" bson:"firstName"`
	LastName  string `json:"lastName" bson:"lastName"`
	Email     string `json:"email" bson:"email"`
	Code      string `json:"code" bson:"code"`
	Provider  string `json:"provider" bson:"provider"`
}

/*
Log In Request
  - Request comming from client
*/
type LogInRequest struct {
	Code     string `json:"code" bson:"code"`
	Provider string `json:"provider" bson:"provider"`
}

/*
Log in Response
  - User identifiable data for client
*/
type LoginResponse struct {
	UUID      string `json:"uuid"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
}

/*
Auth User
  - User data for auth database
  - Contains user identifiable data
*/
type AuthUser struct {
	UUID        string `json:"uuid" bson:"uuid"`
	FirstName   string `json:"firstName" bson:"firstName"`
	LastName    string `json:"lastName" bson:"lastName"`
	Email       string `json:"email" bson:"email"`
	Token       string `json:"token" bson:"token"`
	AccessToken string `json:"accessToken" bson:"accessToken"`
	Provider    string `json:"provider" bson:"provider"`
	CreatedAt   int64  `json:"createdAt" bson:"createdAt"`
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
