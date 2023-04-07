package club

import (
	"olympsis-server/club/service"
	"olympsis-server/database"
	lkup "olympsis-server/lookup/service"
	notif "olympsis-server/pushnote/service"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type ClubAPI struct {
	Logger  *logrus.Logger
	Router  *mux.Router
	Service *service.Service
}

func NewClubAPI(l *logrus.Logger, r *mux.Router, d *database.Database, n *notif.Service, lk *lkup.Service) *ClubAPI {
	return &ClubAPI{Logger: l, Router: r, Service: service.NewClubService(l, r, d, n, lk)}
}

func (s *ClubAPI) Ready() {
	s.Router.Handle("/v1/clubs", s.Service.GetClubs()).Methods("GET")
	s.Router.Handle("/v1/clubs/{id}", s.Service.GetClub()).Methods("GET")
	s.Router.Handle("/v1/clubs/{id}", s.Service.UpdateClub()).Methods("PUT")
	s.Router.Handle("/v1/clubs", s.Service.CreateClub()).Methods("POST")
	s.Router.Handle("/v1/clubs/{id}", s.Service.DeleteClub()).Methods("DELETE")
	s.Router.Handle("/v1/clubs/{id}/leave", s.Service.LeaveClub()).Methods("PUT")

	// applications
	s.Router.Handle("/v1/clubs/{id}/applications", s.Service.GetApplications()).Methods("GET")
	s.Router.Handle("/v1/clubs/{id}/applications", s.Service.CreateApplication()).Methods("POST")
	s.Router.Handle("/v1/clubs/{id}/applications/{applicationId}", s.Service.UpdateApplication()).Methods("PUT")
	s.Router.Handle("/v1/clubs/{id}/applications/{applicationId}", s.Service.DeleteApplication()).Methods("DELETE")

	// members
	s.Router.Handle("/v1/clubs/{id}/members/{memberId}/rank", s.Service.ChangeMemberRank()).Methods("PUT")
	s.Router.Handle("/v1/clubs/{id}/members/{memberId}/kick", s.Service.KickMember()).Methods("PUT")
	s.Router.Handle("/v1/clubs/{id}/members", s.Service.LeaveClub()).Methods("PUT")

	// invites
	s.Router.Handle("/v1/clubs/invites", s.Service.CreateInvitation()).Methods("POST")

}
