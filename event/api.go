package event

import (
	"olympsis-server/database"
	"olympsis-server/event/service"
	notif "olympsis-server/pushnote/service"
	search "olympsis-server/search"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type EventAPI struct {
	Logger  *logrus.Logger
	Router  *mux.Router
	Service *service.Service
}

func NewEventAPI(l *logrus.Logger, r *mux.Router, d *database.Database, n *notif.Service, sh *search.Service) *EventAPI {
	return &EventAPI{Logger: l, Router: r, Service: service.NewEventService(l, r, d, n, sh)}
}

func (e *EventAPI) Ready() {
	// handlers for http requests
	e.Router.Handle("/events/{id}", e.Service.GetEvent()).Methods("GET")
	e.Router.Handle("/events", e.Service.GetEventsByLocation()).Methods("GET")
	e.Router.Handle("/events", e.Service.CreateEvent()).Methods("POST")
	e.Router.Handle("/events/{id}", e.Service.UpdateAnEvent()).Methods("PUT")
	e.Router.Handle("/events/{id}", e.Service.DeleteAnEvent()).Methods("DELETE")

	e.Router.Handle("/events/{id}/participants", e.Service.AddParticipant()).Methods("POST")
	e.Router.Handle("/events/{id}/participants/{participantId}", e.Service.RemoveParticipant()).Methods("DELETE")

	e.Router.Handle("/events/{id}/subscribe", e.Service.SubscribeToEvent()).Methods("POST")
	e.Router.Handle("/events/{id}/unsubscribe", e.Service.UpdateAnEvent()).Methods("POST")
}
