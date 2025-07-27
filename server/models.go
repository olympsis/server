package server

import (
	"olympsis-server/database"
	"olympsis-server/utils"

	"firebase.google.com/go/auth"
	"github.com/gorilla/mux"
	"github.com/olympsis/search"
	"github.com/sirupsen/logrus"
	"github.com/stripe/stripe-go/v82"
)

type ServerInterface struct {
	Logger   *logrus.Logger
	Router   *mux.Router
	Database *database.Database

	Stripe *stripe.Client // Stripe client

	Auth   *auth.Client    // Firebase auth client
	Search *search.Service // Search service

	Notification *utils.NotificationInterface
}
