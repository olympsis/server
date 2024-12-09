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
	s.Router.Handle("/v1/clubs",
		middleware.Chain(
			s.Service.GetClubsByLocation(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("GET", "OPTIONS")

	// get a club
	s.Router.Handle("/v1/clubs/{id}",
		middleware.Chain(
			s.Service.GetClub(),
			middleware.Logging(),
			middleware.CORS(),
		),
	).Methods("GET", "OPTIONS")

	// update a club - requires admin token
	s.Router.Handle("/v1/clubs/{id}",
		middleware.Chain(
			s.Service.ModifyClub(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("PUT", "OPTIONS")

	// create a club
	s.Router.Handle("/v1/clubs",
		middleware.Chain(
			s.Service.CreateClub(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("POST", "OPTIONS")

	// delete a club - requires admin token
	s.Router.Handle("/v1/clubs/{id}",
		middleware.Chain(
			s.Service.DeleteClub(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("DELETE", "OPTIONS")

	// leave a club
	s.Router.Handle("/v1/clubs/{id}/leave",
		middleware.Chain(
			s.Service.LeaveClub(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("PUT", "OPTIONS")

	/*
		Club Applications
	*/

	// get club application - requires admin token
	s.Router.Handle("/v1/clubs/{id}/applications",
		middleware.Chain(
			s.Service.GetApplications(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("GET", "OPTIONS")

	// create club application
	s.Router.Handle("/v1/clubs/{id}/applications",
		middleware.Chain(
			s.Service.CreateApplication(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("POST", "OPTIONS")

	// update club application - requires admin token
	s.Router.Handle("/v1/clubs/{id}/applications/{applicationID}",
		middleware.Chain(
			s.Service.UpdateApplication(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("PUT", "OPTIONS")

	// delete application
	s.Router.Handle("/v1/clubs/{id}/applications/{applicationID}",
		middleware.Chain(
			s.Service.DeleteApplication(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("DELETE", "OPTIONS")

	/*
		Club Members
	*/

	// change member rank
	s.Router.Handle("/v1/clubs/{id}/members/{memberID}/rank",
		middleware.Chain(
			s.Service.ChangeMemberRank(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("PUT", "OPTIONS")

	// kick member from club
	s.Router.Handle("/v1/clubs/{id}/members/{memberID}/kick",
		middleware.Chain(
			s.Service.KickMember(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("PUT")

	// leave club
	s.Router.Handle("/v1/clubs/{id}/members",
		middleware.Chain(
			s.Service.LeaveClub(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("PUT", "OPTIONS")

	/*
		Club Post
	*/

	s.Router.Handle("/v1/clubs/{id}/post/{postID}",
		middleware.Chain(
			s.Service.PinClubPost(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("PUT", "OPTIONS")

	s.Router.Handle("/v1/clubs/{id}/post",
		middleware.Chain(
			s.Service.UnpinClubPost(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("PUT", "OPTIONS")
}
