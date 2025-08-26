package service

import (
	"olympsis-server/database"
	"olympsis-server/notifications"

	"github.com/gorilla/mux"
	"github.com/olympsis/models"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/*
Event Service Struct
*/
type Service struct {
	Database     *database.Database     // database for read/write operations
	Logger       *logrus.Logger         // logger for logging errors
	Router       *mux.Router            // router for handling incoming requests
	Notification *notifications.Service // notification service for sending notifications
}

// Query parameters structure for cleaner handling
type EventQueryParams struct {
	Location *models.GeoJSON
	Sports   []string
	VenueIDs []primitive.ObjectID
	Radius   float64
	Skip     int
	Limit    int
}

// LocationQueryParams holds validated query parameters for the Location endpoint
type LocationQueryParams struct {
	Longitude float64
	Latitude  float64
	Radius    float64
	Sports    []string
	Status    string
	Skip      int
	Limit     int
}
