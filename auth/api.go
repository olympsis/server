package auth

import (
	"olympsis-server/auth/service"
	"olympsis-server/middleware"
	"olympsis-server/server"

	"firebase.google.com/go/auth"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type AuthAPI struct {
	Logger  *logrus.Logger
	Router  *mux.Router
	Service *service.Service
}

func NewAuthAPI(i *server.ServerInterface) *AuthAPI {
	return &AuthAPI{Logger: i.Logger, Router: i.Router, Service: service.NewAuthService(i)}
}

func (s *AuthAPI) Ready(firebase *auth.Client) {

	s.Router.Handle("/v1/auth/register",
		middleware.Chain(
			s.Service.Register(),
			middleware.Logging(),
			middleware.CORS(),
		),
	).Methods("POST", "OPTIONS")

	s.Router.Handle("/v1/auth/login",
		middleware.Chain(
			s.Service.Login(),
			middleware.Logging(),
			middleware.CORS(),
		),
	).Methods("POST", "OPTIONS")

	s.Router.Handle("/v1/auth/delete",
		middleware.Chain(
			s.Service.Delete(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("DELETE", "OPTIONS")
}
