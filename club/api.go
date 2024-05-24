package club

import (
	"olympsis-server/club/service"
	"olympsis-server/database"
	"olympsis-server/middleware"

	"firebase.google.com/go/v4/auth"
	"github.com/gorilla/mux"
	"github.com/olympsis/search"
	"github.com/sirupsen/logrus"
)

type ClubAPI struct {
	Logger  *logrus.Logger
	Router  *mux.Router
	Service *service.Service
}

func NewClubAPI(l *logrus.Logger, r *mux.Router, d *database.Database, sh *search.Service) *ClubAPI {
	return &ClubAPI{Logger: l, Router: r, Service: service.NewClubService(l, r, d, sh)}
}

func (s *ClubAPI) Ready(firebase *auth.Client) {
	/*
		BASIC
	*/

	// get clubs
	s.Router.Handle("/clubs",
		middleware.Chain(
			s.Service.GetClubsByLocation(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("GET")

	// get a club
	s.Router.Handle("/clubs/{id}",
		middleware.Chain(
			s.Service.GetClub(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("GET")

	// update a club - requires admin token
	s.Router.Handle("/clubs/{id}",
		middleware.Chain(
			s.Service.ModifyClub(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("PUT")

	// create a club
	s.Router.Handle("/clubs",
		middleware.Chain(
			s.Service.CreateClub(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("POST")

	// delete a club - requires admin token
	s.Router.Handle("/clubs/{id}",
		middleware.Chain(
			s.Service.DeleteClub(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("DELETE")

	// leave a club
	s.Router.Handle("/clubs/{id}/leave",
		middleware.Chain(
			s.Service.LeaveClub(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("PUT")

	/*
		Club Applications
	*/

	// get club application - requires admin token
	s.Router.Handle("/clubs/{id}/applications",
		middleware.Chain(
			s.Service.GetApplications(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("GET")

	// create club application
	s.Router.Handle("/clubs/{id}/applications",
		middleware.Chain(
			s.Service.CreateApplication(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("POST")

	// update club application - requires admin token
	s.Router.Handle("/clubs/{id}/applications/{applicationID}",
		middleware.Chain(
			s.Service.UpdateApplication(),
			middleware.Logging(),

			middleware.UserMiddleware(firebase),
		),
	).Methods("PUT")

	// delete application
	s.Router.Handle("/clubs/{id}/applications/{applicationID}",
		middleware.Chain(
			s.Service.DeleteApplication(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("DELETE")

	/*
		Club Members
	*/

	// change member rank
	s.Router.Handle("/clubs/{id}/members/{memberID}/rank",
		middleware.Chain(
			s.Service.ChangeMemberRank(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("PUT")

	// kick member from club
	s.Router.Handle("/clubs/{id}/members/{memberID}/kick",
		middleware.Chain(
			s.Service.KickMember(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("PUT")

	// leave club
	s.Router.Handle("/clubs/{id}/members",
		middleware.Chain(
			s.Service.LeaveClub(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("PUT")

	/*
		Club Post
	*/

	s.Router.Handle("/clubs/{id}/post/{postID}",
		middleware.Chain(
			s.Service.PinClubPost(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("PUT")

	s.Router.Handle("/clubs/{id}/post",
		middleware.Chain(
			s.Service.UnpinClubPost(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("PUT")
}
