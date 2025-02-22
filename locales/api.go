package locales

import (
	"olympsis-server/locales/service"
	"olympsis-server/middleware"
	"olympsis-server/server"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type LocaleAPI struct {
	Logger  *logrus.Logger
	Router  *mux.Router
	Service *service.Service
}

func NewLocaleAPI(i *server.ServerInterface) *LocaleAPI {
	return &LocaleAPI{
		Logger:  i.Logger,
		Router:  i.Router,
		Service: service.NewLocaleService(i.Logger, i.Router, i.Database),
	}
}

func (e *LocaleAPI) Ready() {

	e.Router.Handle("/v1/locales/countries",
		middleware.Chain(
			e.Service.GetCountries(),
			middleware.Logging(),
			middleware.CORS(),
		),
	).Methods("GET", "OPTIONS")

	e.Router.Handle("/v1/locales/countries/{id}/administrativeAreas",
		middleware.Chain(
			e.Service.GetAdministrativeAreas(),
			middleware.Logging(),
			middleware.CORS(),
		),
	).Methods("GET", "OPTIONS")

	e.Router.Handle("/v1/locales/administrativeAreas/{id}/subAdministrativeAreas",
		middleware.Chain(
			e.Service.GetSubAdministrativeAreas(),
			middleware.Logging(),
			middleware.CORS(),
		),
	).Methods("GET", "OPTIONS")
}
