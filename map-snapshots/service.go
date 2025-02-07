package mapsnapshots

import (
	"olympsis-server/map-snapshots/service"
	"olympsis-server/middleware"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type SnapShotAPI struct {
	Logger  *logrus.Logger
	Router  *mux.Router
	Service *service.Service
}

func (e *SnapShotAPI) Ready() {

	e.Router.Handle("/v1/map-snapshot",
		middleware.Chain(
			e.Service.GetMapSnapShot(),
			middleware.Logging(),
			middleware.CORS(),
		),
	).Methods("GET", "OPTIONS")
}
