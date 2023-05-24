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
	s.Router.Handle("/auth/signup", s.Service.SignUp()).Methods("POST")
	s.Router.Handle("/auth/login", s.Service.Login()).Methods("POST")
	s.Router.Handle("/auth/logout", s.Service.Logout()).Methods("POST")
	s.Router.Handle("/auth/delete", s.Service.Delete()).Methods("DELETE")
	s.Router.Handle("/auth/apple/notification", s.Service.AppleNotifications()).Methods("POST")
}
