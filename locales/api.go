package locales

import (
	"olympsis-server/database"
	"olympsis-server/locales/service"
	"olympsis-server/middleware"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type LocaleAPI struct {
	Logger  *logrus.Logger
	Router  *mux.Router
	Service *service.Service
}

func NewLocaleAPI(l *logrus.Logger, r *mux.Router, d *database.Database) *LocaleAPI {
	return &LocaleAPI{Logger: l, Router: r, Service: service.NewLocaleService(l, r, d)}
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
