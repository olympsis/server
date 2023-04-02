package service

import (
	"github.com/gorilla/mux"
	"github.com/minio/minio-go/v7"
	"github.com/sirupsen/logrus"
)

type Service struct {
	Client *minio.Client
	Logger *logrus.Logger
	Router *mux.Router
}
