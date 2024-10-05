package venue

import (
	"olympsis-server/database"
	"olympsis-server/middleware"
	"olympsis-server/venue/service"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type VenueAPI struct {
	Logger  *logrus.Logger
	Router  *mux.Router
	Service *service.Service
}

func NewVenueAPI(l *logrus.Logger, r *mux.Router, d *database.Database) *VenueAPI {
	return &VenueAPI{Logger: l, Router: r, Service: service.NewFieldService(l, r, d)}
}

func (s *VenueAPI) Ready() {
	/*
		ROUTES
	*/

	// get fields
	s.Router.Handle("/v1/fields",
		middleware.Chain(
			s.Service.GetFields(),
			middleware.Logging(),
			middleware.CORS(),
		),
	).Methods("GET", "OPTIONS")

	// get a field
	s.Router.Handle("/v1/fields/{id}",
		middleware.Chain(
			s.Service.GetAField(),
			middleware.Logging(),
			middleware.CORS(),
		),
	).Methods("GET", "OPTIONS")

	// create field
	s.Router.Handle("/v1/fields",
		middleware.Chain(
			s.Service.InsertAField(),
			middleware.Logging(),
			middleware.CORS(),
		),
	).Methods("POST", "OPTIONS")

	// update a field
	s.Router.Handle("/v1/fields/{id}",
		middleware.Chain(
			s.Service.UpdateAField(),
			middleware.Logging(),
			middleware.CORS(),
		),
	).Methods("PUT", "OPTIONS")

	// delete a field
	s.Router.Handle("/v1/fields/{id}",
		middleware.Chain(
			s.Service.DeleteAField(),
			middleware.Logging(),
			middleware.CORS(),
		),
	).Methods("DELETE", "OPTIONS")

	// VENUE CHANGE

	// get fields
	s.Router.Handle("/v1/venues",
		middleware.Chain(
			s.Service.GetFields(),
			middleware.Logging(),
			middleware.CORS(),
		),
	).Methods("GET", "OPTIONS")

	// get a field
	s.Router.Handle("/v1/venues/{id}",
		middleware.Chain(
			s.Service.GetAField(),
			middleware.Logging(),
			middleware.CORS(),
		),
	).Methods("GET", "OPTIONS")

	// create field
	s.Router.Handle("/v1/venues",
		middleware.Chain(
			s.Service.InsertAField(),
			middleware.Logging(),
			middleware.CORS(),
		),
	).Methods("POST", "OPTIONS")

	// update a field
	s.Router.Handle("/v1/venues/{id}",
		middleware.Chain(
			s.Service.UpdateAField(),
			middleware.Logging(),
			middleware.CORS(),
		),
	).Methods("PUT", "OPTIONS")

	// delete a field
	s.Router.Handle("/v1/venues/{id}",
		middleware.Chain(
			s.Service.DeleteAField(),
			middleware.Logging(),
			middleware.CORS(),
		),
	).Methods("DELETE", "OPTIONS")
}
