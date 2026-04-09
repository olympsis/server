package storage

import (
	"olympsis-server/middleware"
	"olympsis-server/server"
	"olympsis-server/storage/service"

	"firebase.google.com/go/auth"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type StorageAPI struct {
	Logger  *logrus.Logger   // logger for logging errors
	Router  *mux.Router      // router for handling requests
	Service *service.Service // service for handling requests to
}

func NewStorageAPI(i *server.ServerInterface) *StorageAPI {
	return &StorageAPI{
		Logger:  i.Logger,
		Router:  i.Router,
		Service: service.NewStorageService(i),
	}
}

func (s *StorageAPI) Ready(firebase *auth.Client) {
	// upload object to storage bucket
	s.Router.Handle("/v1/storage/{fileBucket}",
		middleware.Chain(
			s.Service.UploadObject(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("POST", "OPTIONS")

	// delete object from storage bucket
	s.Router.Handle("/v1/storage/{fileBucket}",
		middleware.Chain(
			s.Service.DeleteObject(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("DELETE", "OPTIONS")
}
