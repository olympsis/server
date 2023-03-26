package field

import (
	"olympsis-server/database"
	"olympsis-server/field/service"

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

	// get fields
	s.Router.Handle("/v1/fields", s.Service.GetFields()).Methods("GET")

	// get a field
	s.Router.Handle("/v1/fields/{id}", s.Service.GetAField()).Methods("GET")

	// create field
	s.Router.Handle("/v1/fields", s.Service.InsertAField()).Methods("POST")

	// update a field
	s.Router.Handle("/v1/fields/{id}", s.Service.UpdateAField()).Methods("PUT")

	// delete a field
	s.Router.Handle("/v1/fields/{id}", s.Service.DeleteAField()).Methods("DELETE")
}
