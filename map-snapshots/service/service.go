package service

import (
	"net/http"
	"olympsis-server/database"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

/*
Service Struct
  - Database pointer
  - Logger pointer
  - Router pointer
*/
type Service struct {
	Database *database.Database
	Logger   *logrus.Logger
	Router   *mux.Router
}

/*
Creates a new instance of the locale service
*/
func NewSnapshotService(l *logrus.Logger, r *mux.Router, d *database.Database) *Service {
	return &Service{Logger: l, Router: r, Database: d}
}

func (s *Service) GetMapSnapShot() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.WriteHeader(http.StatusOK)
	}
}
