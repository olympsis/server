package user

import (
	"olympsis-server/database"
	"olympsis-server/user/service"

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

func (u *UserAPI) Ready() {

	// handlers for http requests
	u.Router.Handle("/v1/users/username", u.Service.CheckUsername()).Methods("GET")
	u.Router.Handle("/v1/users/user", u.Service.GetUserData()).Methods("GET")
	u.Router.Handle("/v1/users", u.Service.CreateUserData()).Methods("POST")
	u.Router.Handle("/v1/users/user", u.Service.UpdateUserData()).Methods("PUT")
	u.Router.Handle("/v1/users/user", u.Service.DeleteUserData()).Methods("DELETE")

	// friends
	u.Router.Handle("/v1/users/friends/requests", u.Service.GetFriendRequests()).Methods("GET")
	u.Router.Handle("/v1/users/friends/requests", u.Service.CreateFriendRequest()).Methods("POST")
	u.Router.Handle("/v1/users/friends/requests/{id}", u.Service.UpdateFriendRequest()).Methods("PUT")
	u.Router.Handle("/v1/users/friends/requests/{id}", u.Service.DeleteFriendRequest()).Methods("DELETE")
	u.Router.Handle("/v1/users/friends/{id}", u.Service.RemoveFriend()).Methods("DELETE")

	// badges
	u.Router.Handle("/v1/users/badges", u.Service.AddBadge()).Methods("POST")
	u.Router.Handle("/v1/users/badges/{id}", u.Service.RemoveBadge()).Methods("DELETE")

	// trophies
	u.Router.Handle("/v1/users/trophies", u.Service.AddTrophy()).Methods("POST")
	u.Router.Handle("/v1/users/trophies/{id}", u.Service.RemoveTrophy()).Methods("DELETE")

	// club invites
	u.Router.Handle("/v1/users/club/invites", u.Service.GetClubInvites()).Methods("GET")
	u.Router.Handle("/v1/users/club/invites/{id}", u.Service.UpdateClubInvite()).Methods("PUT")
}
