package server

import (
	"olympsis-server/database"

	"firebase.google.com/go/auth"
	"github.com/gorilla/mux"
	"github.com/olympsis/search"
	"github.com/sirupsen/logrus"
)

type ServerInterface struct {
	Logger   *logrus.Logger
	Router   *mux.Router
	Database *database.Database

	Auth   *auth.Client
	Search *search.Service
}
