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

	// get fields
	s.Router.Handle("/v1/fields",
		middleware.Chain(
			s.Service.GetVenues(),
			middleware.Logging(),
		),
	).Methods("GET", "OPTIONS")

	// get a field
	s.Router.Handle("/v1/fields/{id}",
		middleware.Chain(
			s.Service.GetVenue(),
			middleware.Logging(),
		),
	).Methods("GET", "OPTIONS")

	// create field
	s.Router.Handle("/v1/fields",
		middleware.Chain(
			s.Service.CreateVenue(),
			middleware.Logging(),
		),
	).Methods("POST", "OPTIONS")

	// update a field
	s.Router.Handle("/v1/fields/{id}",
		middleware.Chain(
			s.Service.UpdateVenue(),
			middleware.Logging(),
		),
	).Methods("PUT", "OPTIONS")

	// delete a field
	s.Router.Handle("/v1/fields/{id}",
		middleware.Chain(
			s.Service.DeleteVenue(),
			middleware.Logging(),
		),
	).Methods("DELETE", "OPTIONS")

	// VENUE CHANGE

	// get fields
	s.Router.Handle("/v1/venues",
		middleware.Chain(
			s.Service.GetVenues(),
			middleware.Logging(),
		),
	).Methods("GET", "OPTIONS")

	// get a field
	s.Router.Handle("/v1/venues/{id}",
		middleware.Chain(
			s.Service.GetVenue(),
			middleware.Logging(),
		),
	).Methods("GET", "OPTIONS")

	// create field
	s.Router.Handle("/v1/venues",
		middleware.Chain(
			s.Service.CreateVenue(),
			middleware.Logging(),
		),
	).Methods("POST", "OPTIONS")

	// update a field
	s.Router.Handle("/v1/venues/{id}",
		middleware.Chain(
			s.Service.UpdateVenue(),
			middleware.Logging(),
		),
	).Methods("PUT", "OPTIONS")

	// delete a field
	s.Router.Handle("/v1/venues/{id}",
		middleware.Chain(
			s.Service.DeleteVenue(),
			middleware.Logging(),
		),
	).Methods("DELETE", "OPTIONS")
}
