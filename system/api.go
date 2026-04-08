package system

import (
	"olympsis-server/middleware"
	"olympsis-server/server"

	"firebase.google.com/go/auth"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type SystemAPI struct {
	Logger  *logrus.Logger // logger for logging errors
	Router  *mux.Router    // router for handling requests
	Service *Service       // service for handing requests to
}

func NewConfigApi(i *server.ServerInterface) *SystemAPI {
	return &SystemAPI{
		Logger:  i.Logger,
		Router:  i.Router,
		Service: NewSystemService(i),
	}
}

func (e *SystemAPI) Ready(firebase *auth.Client) {
	e.Router.Handle("/v1/system/config",
		middleware.Chain(
			e.Service.GetConfig(),
			middleware.Logging(),
			middleware.CORS(),
		),
	).Methods("GET", "OPTIONS")

	e.Router.Handle("/v1/system/mapkit-server-token",
		middleware.Chain(
			e.Service.GetMapkitServerToken(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("GET", "OPTIONS")
}
