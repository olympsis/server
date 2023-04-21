package storage

import (
	"olympsis-server/storage/service"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type StorageAPI struct {
	Logger  *logrus.Logger
	Router  *mux.Router
	Service *service.Service
}

func NewStorageAPI(l *logrus.Logger, r *mux.Router) *StorageAPI {
	return &StorageAPI{Logger: l, Router: r, Service: service.NewStorageService(l, r)}
}

func (s *StorageAPI) Ready() {
	s.Service.ConnectToClient()
	s.Router.Handle("/v1/storage/{fileBucket}", s.Service.UploadObject()).Methods("POST")
	s.Router.Handle("/v1/storage/{fileBucket}", s.Service.DeleteObject()).Methods("DELETE")
}
