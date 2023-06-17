package event

import (
	"olympsis-server/database"
	"olympsis-server/event/service"
	"olympsis-server/middleware"

	"github.com/gorilla/mux"
	"github.com/olympsis/notif"
	"github.com/olympsis/search"
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

	/*
		EVENTS
	*/

	// get events
	e.Router.Handle("/events",
		middleware.Chain(
			e.Service.GetEventsByLocation(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("GET")

	// get an event
	e.Router.Handle("/events/{id}",
		middleware.Chain(
			e.Service.GetEvent(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("GET")

	// create an event
	e.Router.Handle("/events",
		middleware.Chain(
			e.Service.CreateEvent(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("POST")

	// update an event
	e.Router.Handle("/events/{id}",
		middleware.Chain(
			e.Service.UpdateAnEvent(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("PUT")

	// delete an event
	e.Router.Handle("/events/{id}",
		middleware.Chain(
			e.Service.DeleteAnEvent(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("DELETE")

	/*
		EVENT PARTICIPANTS
	*/

	// add a participant
	e.Router.Handle("/events/{id}/participants",
		middleware.Chain(
			e.Service.AddParticipant(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("POST")

	// remove a participant
	e.Router.Handle("/events/{id}/participants/{participantID}",
		middleware.Chain(
			e.Service.RemoveParticipant(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("DELETE")

	/*
		EVENT NOTIFICATIONS
	*/

	// subscribe to event notifications
	e.Router.Handle("/events/{id}/subscribe",
		middleware.Chain(
			e.Service.SubscribeToEvent(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("POST")

	// unsubscribe from event notifications
	e.Router.Handle("/events/{id}/unsubscribe",
		middleware.Chain(
			e.Service.UnsubscribeFromEvent(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("POST")
}
