package announcement

import (
	"olympsis-server/announcement/service"
	"olympsis-server/middleware"
	"olympsis-server/server"

	"firebase.google.com/go/auth"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type AnnouncementAPI struct {
	Logger  *logrus.Logger   // logger for logging errors
	Router  *mux.Router      // router for handling requests
	Service *service.Service // service for handling requests
}

func NewAnnouncementAPI(i *server.ServerInterface) *AnnouncementAPI {
	return &AnnouncementAPI{
		Logger: i.Logger,
		Router: i.Router,
		Service: service.NewAnnouncementService(
			i.Logger,
			i.Router,
			i.Database,
		),
	}
}

func (a *AnnouncementAPI) Ready(firebase *auth.Client) {
	// Get all announcements (optionally filtered by location)
	a.Router.Handle("/v1/announcements",
		middleware.Chain(
			a.Service.GetAnnouncements(),
			middleware.Logging(),
			middleware.CORS(),
		),
	).Methods("GET", "OPTIONS")

	// Create a new announcement (admin only)
	a.Router.Handle("/v1/announcements",
		middleware.Chain(
			a.Service.CreateAnnouncement(),
			middleware.Logging(),
			middleware.CORS(),
		),
	).Methods("POST", "OPTIONS")

	// Update an announcement (admin only)
	a.Router.Handle("/v1/announcements/{id}",
		middleware.Chain(
			a.Service.UpdateAnnouncement(),
			middleware.Logging(),
			middleware.CORS(),
		),
	).Methods("PUT", "OPTIONS")
}
