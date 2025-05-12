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
		Service: service.NewEventService(i),
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

	e.Router.Handle("/v1/events/past/user/{uuid}",
		middleware.Chain(
			e.Service.GetUserPastEvents(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("GET", "OPTIONS")

	e.Router.Handle("/v1/events/past/group/{id}",
		middleware.Chain(
			e.Service.GetGroupPastEvents(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("GET", "OPTIONS")

	// get events
	e.Router.Handle("/v1/events",
		middleware.Chain(
			e.Service.GetEvents(),
			middleware.Logging(),
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
		EVENT COMMENTS
	*/

	// add a comment
	e.Router.Handle("/v1/events/{id}/comments",
		middleware.Chain(
			e.Service.AddComment(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("POST", "OPTIONS")

	// remove a comment
	e.Router.Handle("/v1/events/{id}/comments/{commentID}",
		middleware.Chain(
			e.Service.RemoveComment(),
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
	e.Router.Handle("/v1/events/{id}/notify/organizers",
		middleware.Chain(
			e.Service.NotifyOrganizers(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("POST", "OPTIONS")
}
