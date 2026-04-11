package organization

import (
	"olympsis-server/middleware"
	"olympsis-server/organization/service"
	"olympsis-server/server"

	"firebase.google.com/go/auth"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type OrganizationAPI struct {
	Logger  *logrus.Logger
	Router  *mux.Router
	Service *service.Service
}

func NewOrganizationAPI(i *server.ServerInterface) *OrganizationAPI {
	return &OrganizationAPI{
		Logger:  i.Logger,
		Router:  i.Router,
		Service: service.NewOrgService(i),
	}
}

func (e *OrganizationAPI) Ready(firebase *auth.Client) {

	/*
		ORGANIZATIONS
	*/

	// create an organization
	e.Router.Handle("/v1/organizations",
		middleware.Chain(
			e.Service.CreateOrganization(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("POST", "OPTIONS")

	// get organizations
	e.Router.Handle("/v1/organizations",
		middleware.Chain(
			e.Service.GetOrganizations(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("GET", "OPTIONS")

	// get an organization
	e.Router.Handle("/v1/organizations/{id}",
		middleware.Chain(
			e.Service.GetOrganization(),
			middleware.Logging(),
		),
	).Methods("GET", "OPTIONS")

	// update an organization
	e.Router.Handle("/v1/organizations/{id}",
		middleware.Chain(
			e.Service.UpdateOrganization(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("PUT", "OPTIONS")

	// delete an organization
	e.Router.Handle("/v1/organizations/{id}",
		middleware.Chain(
			e.Service.DeleteOrganization(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("DELETE", "OPTIONS")

	/*
		APPLICATION
	*/
	// create an application
	e.Router.Handle("/v1/organizations/applications",
		middleware.Chain(
			e.Service.CreateApplication(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("POST", "OPTIONS")

	// get applications
	e.Router.Handle("/v1/organizations/{id}/applications",
		middleware.Chain(
			e.Service.GetApplications(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("GET", "OPTIONS")

	// get an application
	e.Router.Handle("/v1/organizations/applications/{id}",
		middleware.Chain(
			e.Service.GetApplication(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("GET", "OPTIONS")

	// update an application
	e.Router.Handle("/v1/organizations/applications/{id}",
		middleware.Chain(
			e.Service.UpdateApplication(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("PUT", "OPTIONS")

	// delete an application
	e.Router.Handle("/v1/organizations/applications/{id}",
		middleware.Chain(
			e.Service.DeleteApplication(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("DELETE", "OPTIONS")

	/*
		INVITATION
	*/
	// create an invitation
	e.Router.Handle("/v1/organizations/invitations",
		middleware.Chain(
			e.Service.CreateInvitation(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("POST", "OPTIONS")

	// get invitations
	e.Router.Handle("/v1/organizations/{id}/invitations",
		middleware.Chain(
			e.Service.GetInvitations(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("GET", "OPTIONS")

	// get an invitation
	e.Router.Handle("/v1/organizations/invitations/{id}",
		middleware.Chain(
			e.Service.GetInvitation(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("GET", "OPTIONS")

	// update an invitation
	e.Router.Handle("/v1/organizations/invitations/{id}",
		middleware.Chain(
			e.Service.UpdateInvitation(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("PUT", "OPTIONS")

	// delete an invitation
	e.Router.Handle("/v1/organizations/invitations/{id}",
		middleware.Chain(
			e.Service.DeleteInvitation(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("DELETE", "OPTIONS")

	/*
		Club Post
	*/

	e.Router.Handle("/v1/organizations/{id}/post/{postID}",
		middleware.Chain(
			e.Service.PinOrgPost(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("PUT", "OPTIONS")

	e.Router.Handle("/v1/organizations/{id}/post",
		middleware.Chain(
			e.Service.UnpinOrgPost(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("PUT", "OPTIONS")
}
