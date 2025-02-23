package service

import (
	"net/http"
	"olympsis-server/database"
	"olympsis-server/utils"

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

	StorageInterface *utils.StorageInterface
}

/*
Creates a new instance of the locale service
*/
func NewSnapshotService(l *logrus.Logger, r *mux.Router, d *database.Database, s *utils.StorageInterface) *Service {
	return &Service{Logger: l, Router: r, Database: d, StorageInterface: s}
}

func (s *Service) GetMapSnapShot() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		center := r.URL.Query().Get("center")
		if center == "" {
			http.Error(w, `{ "msg": "center query is required" }`, http.StatusBadRequest)
			return
		}
		token, err := utils.GetTokenFromHeader(r)
		if err != nil {
			http.Error(w, `{ "msg": "failed to find token" }`, http.StatusUnauthorized)
			return
		}

		data, err := s.StorageInterface.GetMapSnapshot(token, center)
		if err != nil {
			http.Error(w, `{ "msg": "failed to get map snapshot"} `, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "image/png")
		w.Write(data)
	}
}
