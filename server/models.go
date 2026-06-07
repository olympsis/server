package server

import (
	"olympsis-server/database"
	"olympsis-server/notifications"
	redisDB "olympsis-server/redis"
	"olympsis-server/types"
	"olympsis-server/utils"

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

	Cache *redisDB.RedisDatabase // Redis cache

	Notification *notifications.Service
	Storage      types.StorageUploader // GCP Storage upload capability

	Bots *utils.BotInterface // Telegram/Discord bot father client (no-op when unconfigured)
}
