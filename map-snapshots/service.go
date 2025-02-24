package mapsnapshots

import (
	"olympsis-server/map-snapshots/service"
	"olympsis-server/middleware"
	"olympsis-server/server"
	"olympsis-server/utils"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type MapSnapShotAPI struct {
	Logger  *logrus.Logger
	Router  *mux.Router
	Service *service.Service
}

func NewMapSnapshotAPI(i *server.ServerInterface, config *utils.ServerConfig) *MapSnapShotAPI {
	return &MapSnapShotAPI{
		Logger: i.Logger,
		Router: i.Router,
		Service: service.NewSnapshotService(
			i.Logger,
			i.Router,
			i.Database,
			utils.NewStorageInterface(config.StorageServiceURL, config.MapKitToken, i.Logger),
		),
	}
}

func (e *MapSnapShotAPI) Ready() {

	e.Router.Handle("/v1/map-snapshot",
		middleware.Chain(
			e.Service.GetMapSnapShot(),
			middleware.Logging(),
			middleware.CORS(),
		),
	).Methods("GET", "OPTIONS")
}
