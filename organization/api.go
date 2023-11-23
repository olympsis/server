package organization

import (
	"olympsis-server/database"
	"olympsis-server/middleware"
	"olympsis-server/organization/service"

	"github.com/gorilla/mux"
	"github.com/olympsis/notif"
	"github.com/olympsis/search"
	"github.com/sirupsen/logrus"
)

type OrganizationAPI struct {
	Logger  *logrus.Logger
	Router  *mux.Router
	Service *service.Service
}

func NewOrganizationAPI(l *logrus.Logger, r *mux.Router, d *database.Database, n *notif.Service, sh *search.Service) *OrganizationAPI {
	return &OrganizationAPI{Logger: l, Router: r, Service: service.NewOrganizationService(l, r, d, n, sh)}
}

func (e *OrganizationAPI) Ready() {

	/*
		ORGANIZATIONS
	*/

	// create an organization
	e.Router.Handle("/organizations",
		middleware.Chain(
			e.Service.CreateOrganization(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("POST")

	// get organizations
	e.Router.Handle("/organizations",
		middleware.Chain(
			e.Service.GetOrganizations(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("GET")

	// get an organization
	e.Router.Handle("/organizations/{id}",
		middleware.Chain(
			e.Service.GetOrganization(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("GET")

	// update an organization
	e.Router.Handle("/organizations/{id}",
		middleware.Chain(
			e.Service.UpdateOrganization(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("PUT")

	// delete an organization
	e.Router.Handle("/organizations/{id}",
		middleware.Chain(
			e.Service.DeleteOrganization(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("DELETE")

	/*
		APPLICATION
	*/
	// create an application
	e.Router.Handle("/organizations/applications",
		middleware.Chain(
			e.Service.CreateApplication(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("POST")

	// get applications
	e.Router.Handle("/organizations/applications",
		middleware.Chain(
			e.Service.GetApplications(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("GET")

	// get an application
	e.Router.Handle("/organizations/applications/{id}",
		middleware.Chain(
			e.Service.GetApplication(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("GET")

	// update an application
	e.Router.Handle("/organizations/applications/{id}",
		middleware.Chain(
			e.Service.UpdateApplication(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("PUT")

	// delete an application
	e.Router.Handle("/organizations/applications/{id}",
		middleware.Chain(
			e.Service.DeleteApplication(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("DELETE")
}
