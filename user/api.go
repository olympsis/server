package user

import (
	"olympsis-server/database"
	"olympsis-server/middleware"
	"olympsis-server/user/service"

	"github.com/gorilla/mux"
	"github.com/olympsis/notif"
	"github.com/sirupsen/logrus"
)

type UserAPI struct {
	Logger  *logrus.Logger
	Router  *mux.Router
	Service *service.Service
}

func NewUserAPI(l *logrus.Logger, r *mux.Router, d *database.Database, n *notif.Service) *UserAPI {
	return &UserAPI{Logger: l, Router: r, Service: service.NewUserService(l, r, d, n)}
}

func (u *UserAPI) Ready() {

	/*
		ROUTES
	*/

	// search username availability
	u.Router.Handle("/users/username",
		middleware.Chain(
			u.Service.CheckUsername(),
			middleware.Logging(),
		),
	).Methods("GET")

	// get user data
	u.Router.Handle("/users/user",
		middleware.Chain(
			u.Service.GetUserData(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("GET")

	// create user data
	u.Router.Handle("/users",
		middleware.Chain(
			u.Service.CreateUserData(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("POST")

	// update user data
	u.Router.Handle("/users/user",
		middleware.Chain(
			u.Service.UpdateUserData(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("PUT")
}
