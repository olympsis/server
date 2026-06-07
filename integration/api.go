package integration

import (
	"olympsis-server/middleware"
	"olympsis-server/server"
	"olympsis-server/utils"

	"firebase.google.com/go/auth"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// IntegrationAPI wires the club <-> chat (Telegram/Discord) integration endpoints.
type IntegrationAPI struct {
	Logger  *logrus.Logger
	Router  *mux.Router
	Service *Service
	Config  *utils.ServerConfig
}

func NewIntegrationAPI(i *server.ServerInterface, config *utils.ServerConfig) *IntegrationAPI {
	return &IntegrationAPI{
		Logger:  i.Logger,
		Router:  i.Router,
		Service: NewIntegrationService(i, config),
		Config:  config,
	}
}

func (a *IntegrationAPI) Ready(firebase *auth.Client) {
	// Start linking a club to a platform chat (admin only)
	a.Router.Handle("/v1/clubs/{id}/integrations/{platform}",
		middleware.Chain(
			a.Service.StartLink(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("POST", "OPTIONS")

	// List a club's chat links (members)
	a.Router.Handle("/v1/clubs/{id}/integrations",
		middleware.Chain(
			a.Service.GetLinks(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("GET", "OPTIONS")

	// Unlink a platform chat (admin only)
	a.Router.Handle("/v1/clubs/{id}/integrations/{platform}",
		middleware.Chain(
			a.Service.DeleteLink(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("DELETE", "OPTIONS")

	// Confirm a link from the bots service (internal only)
	a.Router.Handle("/v1/integrations/confirm",
		middleware.Chain(
			a.Service.ConfirmLink(),
			middleware.Logging(),
			middleware.InternalMiddleware(a.Config.BotsSecret),
		),
	).Methods("POST", "OPTIONS")
}
