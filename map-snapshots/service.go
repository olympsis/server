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
	// Create a service instance first
	snapService := service.NewSnapService(i)

	// Set the storage interface using the storage service directly (no external HTTP calls)
	snapService.StorageInterface = utils.NewStorageInterface(
		i.Storage,
		config.MapKitConfig,
		i.Logger,
	)

	return &MapSnapShotAPI{
		Logger:  i.Logger,
		Router:  i.Router,
		Service: snapService,
	}
}

func (e *MapSnapShotAPI) Ready() {

	e.Router.Handle("/v1/map-snapshot",
		middleware.Chain(
			e.Service.GetMapSnapShot(),
			middleware.Logging(),
		),
	).Methods("GET", "OPTIONS")
}
