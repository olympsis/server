package server

import (
	"olympsis-server/database"
	"olympsis-server/utils"

	"firebase.google.com/go/auth"
	"github.com/gorilla/mux"
	"github.com/olympsis/search"
	"github.com/sirupsen/logrus"
)

type ServerInterface struct {
	Logger   *logrus.Logger
	Router   *mux.Router
	Database *database.Database

	Auth   *auth.Client    // firebase auth
	Search *search.Service // search service

	Notification *utils.NotificationInterface
}
