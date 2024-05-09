package event

import (
	"olympsis-server/database"
	"olympsis-server/event/service"
	"olympsis-server/middleware"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type EventAPI struct {
	Logger  *logrus.Logger   // logger for logging errors
	Router  *mux.Router      // router for handling requests
	Service *service.Service // service for handing requests to
}

func NewEventAPI(l *logrus.Logger, r *mux.Router, d *database.Database) *EventAPI {
	return &EventAPI{Logger: l, Router: r, Service: service.NewEventService(l, r, d)}
}

func (e *EventAPI) Ready() {

	/*
		EVENTS
	*/

	// get events
	e.Router.Handle("/events/location",
		middleware.Chain(
			e.Service.Location(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("GET")

	// get events
	e.Router.Handle("/events",
		middleware.Chain(
			e.Service.GetEventsByLocation(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("GET")

	// get events by field
	e.Router.Handle("/events/field/{id}",
		middleware.Chain(
			e.Service.GetEventsByField(),
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

	// notify participants
	e.Router.Handle("/events/{id}/notify/participants",
		middleware.Chain(
			e.Service.NotifyParticipants(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("POST")

	// notify club members
	e.Router.Handle("/events/{id}/notify/clbu",
		middleware.Chain(
			e.Service.NotifyClubMembers(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("POST")
}
