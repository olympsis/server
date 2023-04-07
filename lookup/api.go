package lookup

import (
	"olympsis-server/database"
	"olympsis-server/lookup/service"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type LookupAPI struct {
	Logger  *logrus.Logger
	Router  *mux.Router
	Service *service.Service
}

func NewLookUpAPI(l *logrus.Logger, r *mux.Router, d *database.Database) *LookupAPI {
	return &LookupAPI{Logger: l, Router: r, Service: service.NewLookupService(l, r, d)}
}

func (l *LookupAPI) Ready() {
	l.Router.Handle("/lookup/{id}", l.Service.LookUpUserById()).Methods("GET")
	l.Router.Handle("/lookup/username/{username}", l.Service.LookUpUserUsername()).Methods("GET")
	l.Router.Handle("/lookup/batch/id", l.Service.BatchLookupById()).Methods("POST")
}

func (l *LookupAPI) GetService() *service.Service {
	return l.Service
}
