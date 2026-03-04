package service

import (
	"net/http"
	"olympsis-server/database"
	"olympsis-server/server"
	"olympsis-server/utils"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

/*
Service Struct
  - Database pointer
  - Logger pointer
  - Router pointer
  - Notification interface
*/
type Service struct {
	Database *database.Database
	Logger   *logrus.Logger
	Router   *mux.Router

	StorageInterface *utils.StorageInterface
}

/*
Creates a new instance of the map snapshot service
*/
func NewSnapService(i *server.ServerInterface) *Service {
	return &Service{
		Logger:   i.Logger,
		Router:   i.Router,
		Database: i.Database,
	}
}

func (s *Service) GetMapSnapShot() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		center := r.URL.Query().Get("center")
		if center == "" {
			http.Error(w, `{ "msg": "center query is required" }`, http.StatusBadRequest)
			return
		}

		data, err := s.StorageInterface.GetMapSnapshot(center)
		if err != nil {
			s.Logger.Errorf("Failed to get map snapshot. Error: %s", err.Error())
			http.Error(w, `{ "msg": "failed to get map snapshot"} `, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "image/png")
		w.Write(data)
	}
}
