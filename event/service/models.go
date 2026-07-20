package service

import (
	"olympsis-server/bus"
	"olympsis-server/database"
	"olympsis-server/notifications"
	"olympsis-server/push"

	"github.com/gorilla/mux"
	"github.com/olympsis/models"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/v2/bson"
)

/*
Event Service Struct
*/
type Service struct {
	Database     *database.Database     // database for read/write operations
	Logger       *logrus.Logger         // logger for logging errors
	Router       *mux.Router            // router for handling incoming requests
	Notification *notifications.Service // legacy rich notifications (event create/cancel, kick, etc.)
	Push         *push.Service          // loc_key event push notifications (reminders only — see Bus)
	Bus          *bus.Publisher         // domain events for invite-service / notif-service
}

// Query parameters structure for cleaner handling
type EventQueryParams struct {
	Location *models.GeoJSON
	Sports   []string
	VenueIDs []bson.ObjectID
	Radius   float64
	Skip     int
	Limit    int

	Status models.EventStatus
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
