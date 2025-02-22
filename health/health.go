package health

import (
	"net/http"
	"olympsis-server/server"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type HealthAPI struct {
	Logger *logrus.Logger // logger for logging errors
	Router *mux.Router    // router for handling requests
}

func NewHealthAPI(i *server.ServerInterface) *HealthAPI {
	return &HealthAPI{
		Logger: i.Logger,
		Router: i.Router,
	}
}

func (h *HealthAPI) Ready() {
	h.Router.Handle(
		"/v1/health",
		HealthCheckHandler(),
	)

	h.Router.Handle(
		"/v1/wsg",
		HandleWhatsGood(),
	)
}

func HealthCheckHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{ "msg": "OK" }`))
	}
}

func HandleWhatsGood() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{ "msg": "OK" }`))
	}
}
