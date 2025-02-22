package event

import (
	"olympsis-server/event/service"
	"olympsis-server/middleware"
	"olympsis-server/server"

	"firebase.google.com/go/auth"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type EventAPI struct {
	Logger  *logrus.Logger   // logger for logging errors
	Router  *mux.Router      // router for handling requests
	Service *service.Service // service for handing requests to
}

func NewEventAPI(i *server.ServerInterface) *EventAPI {
	return &EventAPI{
		Logger:  i.Logger,
		Router:  i.Router,
		Service: service.NewEventService(i.Logger, i.Router, i.Database),
	}
}

func (e *EventAPI) Ready(firebase *auth.Client) {

	/*
		EVENTS
	*/

	// get events
	e.Router.Handle("/v1/events/location",
		middleware.Chain(
			e.Service.Location(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("GET", "OPTIONS")

	// get events
	e.Router.Handle("/v1/events",
		middleware.Chain(
			e.Service.GetEventsByLocation(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("GET", "OPTIONS")

	// get events by field
	e.Router.Handle("/v1/events/field/{id}",
		middleware.Chain(
			e.Service.GetEventsByField(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("GET", "OPTIONS")

	// get events by field
	e.Router.Handle("/v1/events/venue/{id}",
		middleware.Chain(
			e.Service.GetEventsByField(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("GET", "OPTIONS")

	// get an event
	e.Router.Handle("/v1/events/{id}",
		middleware.Chain(
			e.Service.GetEvent(),
			middleware.Logging(),
			middleware.CORS(),
		),
	).Methods("GET", "OPTIONS")

	// create an event
	e.Router.Handle("/v1/events",
		middleware.Chain(
			e.Service.CreateEvent(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("POST", "OPTIONS")

	// update an event
	e.Router.Handle("/v1/events/{id}",
		middleware.Chain(
			e.Service.UpdateAnEvent(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("PUT", "OPTIONS")

	// delete an event
	e.Router.Handle("/v1/events/{id}",
		middleware.Chain(
			e.Service.DeleteAnEvent(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("DELETE", "OPTIONS")

	/*
		EVENT PARTICIPANTS
	*/

	// add a participant
	e.Router.Handle("/v1/events/{id}/participants",
		middleware.Chain(
			e.Service.AddParticipant(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("POST", "OPTIONS")

	// remove a participant
	e.Router.Handle("/v1/events/{id}/participants",
		middleware.Chain(
			e.Service.RemoveParticipant(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("DELETE", "OPTIONS")

	// remove a participant by ID
	e.Router.Handle("/v1/events/{id}/participants/{participantID}",
		middleware.Chain(
			e.Service.RemoveParticipant(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("DELETE", "OPTIONS")

	/*
		EVENT NOTIFICATIONS
	*/

	// notify participants
	e.Router.Handle("/v1/events/{id}/notify/participants",
		middleware.Chain(
			e.Service.NotifyParticipants(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("POST", "OPTIONS")

	// notify club members
	e.Router.Handle("/v1/events/{id}/notify/club",
		middleware.Chain(
			e.Service.NotifyClubMembers(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("POST", "OPTIONS")
}
