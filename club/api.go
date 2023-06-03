package club

import (
	"olympsis-server/club/service"
	"olympsis-server/database"
	"olympsis-server/middleware"
	notif "olympsis-server/pushnote/service"
	search "olympsis-server/search"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type ClubAPI struct {
	Logger  *logrus.Logger
	Router  *mux.Router
	Service *service.Service
}

func NewClubAPI(l *logrus.Logger, r *mux.Router, d *database.Database, n *notif.Service, sh *search.Service) *ClubAPI {
	return &ClubAPI{Logger: l, Router: r, Service: service.NewClubService(l, r, d, n, sh)}
}

func (s *ClubAPI) Ready() {
	/*
		BASIC
	*/

	// get clubs
	s.Router.Handle("/clubs",
		middleware.Chain(
			s.Service.GetClubs(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("GET")

	// get a club
	s.Router.Handle("/clubs/{id}",
		middleware.Chain(
			s.Service.GetClub(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("GET")

	// update a club - requires admin token
	s.Router.Handle("/clubs/{id}",
		middleware.Chain(
			s.Service.UpdateClub(),
			middleware.Logging(),
			middleware.ClubAdminMiddleware(),
			middleware.UserMiddleware(),
		),
	).Methods("PUT")

	// create a club
	s.Router.Handle("/clubs",
		middleware.Chain(
			s.Service.CreateClub(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("POST")

	// delete a club - requires admin token
	s.Router.Handle("/clubs/{id}",
		middleware.Chain(
			s.Service.DeleteClub(),
			middleware.Logging(),
			middleware.ClubAdminMiddleware(),
			middleware.UserMiddleware(),
		),
	).Methods("DELETE")

	// leave a club
	s.Router.Handle("/clubs/{id}/leave",
		middleware.Chain(
			s.Service.LeaveClub(),
			middleware.Logging(),
			middleware.UserMiddleware(),
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
			middleware.ClubAdminMiddleware(),
			middleware.UserMiddleware(),
		),
	).Methods("GET")

	// create club application
	s.Router.Handle("/clubs/{id}/applications",
		middleware.Chain(
			s.Service.CreateApplication(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("POST")

	// update club application - requires admin token
	s.Router.Handle("/clubs/{id}/applications/{applicationID}",
		middleware.Chain(
			s.Service.UpdateApplication(),
			middleware.Logging(),
			middleware.ClubAdminMiddleware(),
			middleware.UserMiddleware(),
		),
	).Methods("PUT")

	// delete application
	s.Router.Handle("/clubs/{id}/applications/{applicationID}",
		middleware.Chain(
			s.Service.DeleteApplication(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("DELETE")

	/*
		Club Members
	*/

	// change member rank
	s.Router.Handle("/clubs/{id}/members/{memberId}/rank",
		middleware.Chain(
			s.Service.ChangeMemberRank(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("PUT")

	// kick member from club
	s.Router.Handle("/clubs/{id}/members/{memberId}/kick",
		middleware.Chain(
			s.Service.KickMember(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("PUT")

	// leave club
	s.Router.Handle("/clubs/{id}/members",
		middleware.Chain(
			s.Service.LeaveClub(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("PUT")
}
