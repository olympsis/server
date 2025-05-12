package service

import (
	"olympsis-server/database"
	"olympsis-server/utils"

	"firebase.google.com/go/auth"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

/*
Authentication Service
*/
type Service struct {
	// Database object
	Database *database.Database

	// Logger
	Log *logrus.Logger

	// Router
	Router *mux.Router

	// Firebase Auth
	Client *auth.Client
	
	// Notification Service
	Notification *utils.NotificationInterface
}
