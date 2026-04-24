package venue

import (
	"olympsis-server/middleware"
	"olympsis-server/server"
	"olympsis-server/venue/service"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type VenueAPI struct {
	Logger  *logrus.Logger
	Router  *mux.Router
	Service *service.Service
}

func NewVenueAPI(i *server.ServerInterface) *VenueAPI {
	return &VenueAPI{
		Logger:  i.Logger,
		Router:  i.Router,
		Service: service.NewVenueService(i),
	}
}

func (s *VenueAPI) Ready() {
	/*
		ROUTES
	*/

	// Get venues
	s.Router.Handle("/v1/venues",
		middleware.Chain(
			s.Service.GetVenues(),
			middleware.Logging(),
		),
	).Methods("GET", "OPTIONS")

	// Get a venue
	s.Router.Handle("/v1/venues/{id}",
		middleware.Chain(
			s.Service.GetVenue(),
			middleware.Logging(),
		),
	).Methods("GET", "OPTIONS")

	// Create a venue
	s.Router.Handle("/v1/venues",
		middleware.Chain(
			s.Service.CreateVenue(),
			middleware.Logging(),
		),
	).Methods("POST", "OPTIONS")

	// Update a venue
	s.Router.Handle("/v1/venues/{id}",
		middleware.Chain(
			s.Service.UpdateVenue(),
			middleware.Logging(),
		),
	).Methods("PUT", "OPTIONS")

	// Delete a venue
	s.Router.Handle("/v1/venues/{id}",
		middleware.Chain(
			s.Service.RemoveVenue(),
			middleware.Logging(),
		),
	).Methods("DELETE", "OPTIONS")
}
