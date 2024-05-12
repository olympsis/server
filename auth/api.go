package auth

import (
	"olympsis-server/auth/service"
	"olympsis-server/database"
	"olympsis-server/middleware"

	"firebase.google.com/go/auth"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type AuthAPI struct {
	Logger  *logrus.Logger
	Router  *mux.Router
	Service *service.Service
}

func NewAuthAPI(l *logrus.Logger, r *mux.Router, d *database.Database, c *auth.Client) *AuthAPI {
	return &AuthAPI{Logger: l, Router: r, Service: service.NewAuthService(l, r, d, c)}
}

func (s *AuthAPI) Ready(firebase *auth.Client) {

	s.Router.Handle("/v1/auth/register",
		middleware.Chain(
			s.Service.Register(),
			middleware.Logging(),
		),
	).Methods("POST")

	s.Router.Handle("/v1/auth/login",
		middleware.Chain(
			s.Service.Login(),
			middleware.Logging(),
		),
	).Methods("POST")

	s.Router.Handle("/v1/auth/delete",
		middleware.Chain(
			s.Service.Delete(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("DELETE")
}
