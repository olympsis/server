package service

import (
	"cloud.google.com/go/storage"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type Service struct {
	Client      *storage.Client
	Logger      *logrus.Logger
	Router      *mux.Router
	AccessToken *string
}
