package user

import (
	"olympsis-server/database"
	"olympsis-server/middleware"
	"olympsis-server/user/service"

	"firebase.google.com/go/auth"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type UserAPI struct {
	Logger  *logrus.Logger
	Router  *mux.Router
	Service *service.Service
}

func NewUserAPI(l *logrus.Logger, r *mux.Router, d *database.Database) *UserAPI {
	return &UserAPI{Logger: l, Router: r, Service: service.NewUserService(l, r, d)}
}

func (u *UserAPI) Ready(firebase *auth.Client) {

	/*
		ROUTES
	*/

	// search username availability
	u.Router.Handle("/v1/users/check-in",
		middleware.Chain(
			u.Service.CheckIn(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("GET", "OPTIONS")

	// search username availability
	u.Router.Handle("/v1/users/username",
		middleware.Chain(
			u.Service.CheckUsername(),
			middleware.Logging(),
			middleware.CORS(),
		),
	).Methods("GET", "OPTIONS")

	// get user data
	u.Router.Handle("/v1/users/user",
		middleware.Chain(
			u.Service.GetUserData(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("GET", "OPTIONS")

	// create user data
	u.Router.Handle("/v1/users",
		middleware.Chain(
			u.Service.CreateUserData(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("POST", "OPTIONS")

	// update user data
	u.Router.Handle("/v1/users/user",
		middleware.Chain(
			u.Service.UpdateUserData(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("PUT", "OPTIONS")

	// find organizations invite
	u.Router.Handle("/v1/users/invitations/organizations",
		middleware.Chain(
			u.Service.GetOrganizationInvitations(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("GET", "OPTIONS")

	// search users by username
	u.Router.Handle("/v1/users/search/username",
		middleware.Chain(
			u.Service.SearchUsersByUserName(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("GET", "OPTIONS")

	// search user by uuid
	u.Router.Handle("/v1/users/search/uuid",
		middleware.Chain(
			u.Service.SearchUserByUUID(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("GET", "OPTIONS")
}
