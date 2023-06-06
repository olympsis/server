package auth

import (
	"olympsis-server/auth/service"
	"olympsis-server/database"
	"olympsis-server/middleware"

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
	s.Router.Handle("/auth/signup",
		middleware.Chain(
			s.Service.SignUp(),
			middleware.Logging(),
		),
	).Methods("POST")

	s.Router.Handle("/auth/login",
		middleware.Chain(
			s.Service.Login(),
			middleware.Logging(),
		),
	).Methods("POST")

	s.Router.Handle("/auth/logout",
		middleware.Chain(
			s.Service.Logout(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("POST")

	s.Router.Handle("/auth/delete",
		middleware.Chain(
			s.Service.Delete(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("DELETE")

	s.Router.Handle("/auth/apple/notification",
		middleware.Chain(
			s.Service.AppleNotifications(),
			middleware.Logging(),
		),
	).Methods("POST")
}
