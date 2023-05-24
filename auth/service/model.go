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
