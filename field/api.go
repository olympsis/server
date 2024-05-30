package field

import (
	"olympsis-server/database"
	"olympsis-server/field/service"
	"olympsis-server/middleware"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type FieldAPI struct {
	Logger  *logrus.Logger
	Router  *mux.Router
	Service *service.Service
}

func NewFieldAPI(l *logrus.Logger, r *mux.Router, d *database.Database) *FieldAPI {
	return &FieldAPI{Logger: l, Router: r, Service: service.NewFieldService(l, r, d)}
}

func (s *FieldAPI) Ready() {
	/*
		ROUTES
	*/

	// get fields
	s.Router.Handle("/v1/fields",
		middleware.Chain(
			s.Service.GetFields(),
			middleware.Logging(),
		),
	).Methods("GET")

	// get a field
	s.Router.Handle("/v1/fields/{id}",
		middleware.Chain(
			s.Service.GetAField(),
			middleware.Logging(),
		),
	).Methods("GET")

	// create field
	s.Router.Handle("/v1/fields",
		middleware.Chain(
			s.Service.InsertAField(),
			middleware.Logging(),
		),
	).Methods("POST")

	// update a field
	s.Router.Handle("/v1/fields/{id}",
		middleware.Chain(
			s.Service.UpdateAField(),
			middleware.Logging(),
		),
	).Methods("PUT")

	// delete a field
	s.Router.Handle("/v1/fields/{id}",
		middleware.Chain(
			s.Service.DeleteAField(),
			middleware.Logging(),
		),
	).Methods("DELETE")
}
