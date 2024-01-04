package storage

import (
	"olympsis-server/database"
	"olympsis-server/middleware"
	"olympsis-server/storage/service"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type StorageAPI struct {
	Logger  *logrus.Logger
	Router  *mux.Router
	Service *service.Service
}

func NewStorageAPI(l *logrus.Logger, r *mux.Router, d *database.Database) *StorageAPI {
	return &StorageAPI{Logger: l, Router: r, Service: service.NewStorageService(l, r)}
}

func (s *StorageAPI) Ready() {
	s.Service.ConnectToClient()
	s.Router.Handle("/storage/{fileBucket}",
		middleware.Chain(
			s.Service.UploadObject(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("POST")

	s.Router.Handle("/storage/{fileBucket}",
		middleware.Chain(
			s.Service.DeleteObject(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("DELETE")
}
