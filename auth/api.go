package auth

import (
	"olympsis-server/auth/service"
	"olympsis-server/database"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type AuthAPI struct {
	Logger  *logrus.Logger
	Router  *mux.Router
	Service *service.Service
}

func NewAuthAPI(l *logrus.Logger, r *mux.Router, d *database.Database) *AuthAPI {
	return &AuthAPI{Logger: l, Router: r, Service: service.NewAuthService(l, r, d)}
}

func (s *AuthAPI) Ready() {

	// routes
	s.Router.Handle("/v1/auth/signup", s.Service.SignUp()).Methods("POST")
	s.Router.Handle("/v1/auth/login", s.Service.Login()).Methods("PUT")
	s.Router.Handle("/v1/auth/delete", s.Service.Delete()).Methods("DELETE")
	s.Router.Handle("/v1/auth/apple/notification", s.Service.AppleNotifications()).Methods("POST")
}
