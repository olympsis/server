package server

import (
	"olympsis-server/database"
	"olympsis-server/notifications"
	"olympsis-server/types"

	"firebase.google.com/go/auth"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stripe/stripe-go/v82"
)

type ServerInterface struct {
	Logger   *logrus.Logger
	Router   *mux.Router
	Database *database.Database

	Stripe *stripe.Client // Stripe client

	Auth *auth.Client // Firebase auth client

	Notification *notifications.Service
	Storage      types.StorageUploader // GCP Storage upload capability
}
