package health

import (
	"encoding/json"
	"net/http"
	"olympsis-server/middleware"
	"olympsis-server/server"
	"olympsis-server/version"

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
		middleware.Chain(
			HealthCheckHandler(),
		),
	).Methods("GET", "OPTIONS")

	h.Router.Handle(
		"/v1/health/wsg",
		middleware.Chain(
			HandleWhatsGood(),
		),
	).Methods("GET", "OPTIONS")
}

// healthResponse keeps the original "msg" field so existing clients that just
// check for "OK" are unaffected; the build block is purely additive. It is what
// makes a running process traceable back to an exact commit and image tag.
type healthResponse struct {
	Msg   string       `json:"msg"`
	Build version.Info `json:"build"`
}

func HealthCheckHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(healthResponse{Msg: "OK", Build: version.Get()})
	}
}

func HandleWhatsGood() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(healthResponse{Msg: "OK", Build: version.Get()})
	}
}
