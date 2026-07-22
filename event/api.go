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
		),
	).Methods("GET", "OPTIONS")

	e.Router.Handle("/v1/events/past/user/{user_id}",
		middleware.Chain(
			e.Service.GetUserPastEvents(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("GET", "OPTIONS")

	e.Router.Handle("/v1/events/past/group/{id}",
		middleware.Chain(
			e.Service.GetGroupPastEvents(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("GET", "OPTIONS")

	// get a venue's upcoming events — backs the iOS venue detail page.
	// Registered before "/v1/events/{id}" so "venue" isn't read as an event id.
	e.Router.Handle("/v1/events/venue/{id}",
		middleware.Chain(
			e.Service.GetEventsByVenue(),
			middleware.Logging(),
		),
	).Methods("GET", "OPTIONS")

	// get events
	//
	// Uses OptionalUserMiddleware so unauthenticated callers can still browse
	// upcoming events, but an authenticated caller gets userID set on the
	// request — required by status=ended (past events scoped to the user).
	e.Router.Handle("/v1/events",
		middleware.Chain(
			e.Service.GetEvents(),
			middleware.Logging(),
			middleware.OptionalUserMiddleware(firebase),
		),
	).Methods("GET", "OPTIONS")

	// get an event
	e.Router.Handle("/v1/events/{id}",
		middleware.Chain(
			e.Service.GetEvent(),
			middleware.Logging(),
		),
	).Methods("GET", "OPTIONS")

	// create an event
	e.Router.Handle("/v1/events",
		middleware.Chain(
			e.Service.CreateEvent(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("POST", "OPTIONS")

	// update an event
	e.Router.Handle("/v1/events/{id}",
		middleware.Chain(
			e.Service.UpdateAnEvent(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("PUT", "OPTIONS")

	// cancel an event
	e.Router.Handle("/v1/events/{id}/cancel",
		middleware.Chain(
			e.Service.Cancel(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("PUT", "OPTIONS")

	// delete an event
	e.Router.Handle("/v1/events/{id}",
		middleware.Chain(
			e.Service.DeleteAnEvent(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
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
		),
	).Methods("POST", "OPTIONS")

	// remove a participant
	e.Router.Handle("/v1/events/{id}/participants",
		middleware.Chain(
			e.Service.RemoveParticipant(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("DELETE", "OPTIONS")

	// remove a participant by ID
	e.Router.Handle("/v1/events/{id}/participants/{participantID}",
		middleware.Chain(
			e.Service.RemoveParticipant(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("DELETE", "OPTIONS")

	/*
		EVENT TEAMS
	*/

	// add a team
	e.Router.Handle("/v1/events/{id}/teams",
		middleware.Chain(
			e.Service.CreateTeam(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("POST", "OPTIONS")

	// remove a team
	e.Router.Handle("/v1/events/{id}/teams/{teamID}",
		middleware.Chain(
			e.Service.RemoveTeam(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("DELETE", "OPTIONS")

	// join an open team
	e.Router.Handle("/v1/events/{id}/teams/{teamID}/join",
		middleware.Chain(
			e.Service.JoinTeam(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("POST", "OPTIONS")

	// invite more users to a team (owner only)
	e.Router.Handle("/v1/events/{id}/teams/{teamID}/invites",
		middleware.Chain(
			e.Service.InviteToTeam(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("POST", "OPTIONS")

	// transfer team ownership (owner only)
	e.Router.Handle("/v1/events/{id}/teams/{teamID}/owner",
		middleware.Chain(
			e.Service.TransferTeamOwnership(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("PUT", "OPTIONS")

	// leave a team (self)
	e.Router.Handle("/v1/events/{id}/teams/{teamID}/members",
		middleware.Chain(
			e.Service.LeaveTeam(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("DELETE", "OPTIONS")

	// kick a member from a team (owner only)
	e.Router.Handle("/v1/events/{id}/teams/{teamID}/members/{memberUserID}",
		middleware.Chain(
			e.Service.KickTeamMember(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("DELETE", "OPTIONS")

	// apply to a closed team
	e.Router.Handle("/v1/events/{id}/teams/{teamID}/applications",
		middleware.Chain(
			e.Service.CreateTeamApplication(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("POST", "OPTIONS")

	// list a team's applications (owner only)
	e.Router.Handle("/v1/events/{id}/teams/{teamID}/applications",
		middleware.Chain(
			e.Service.GetTeamApplications(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("GET", "OPTIONS")

	// approve/deny a team application (owner only)
	e.Router.Handle("/v1/events/{id}/teams/{teamID}/applications/{applicationID}",
		middleware.Chain(
			e.Service.UpdateTeamApplication(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("PUT", "OPTIONS")

	/*
		EVENT COMMENTS
	*/

	// add a comment
	e.Router.Handle("/v1/events/{id}/comments",
		middleware.Chain(
			e.Service.AddComment(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("POST", "OPTIONS")

	// remove a comment
	e.Router.Handle("/v1/events/{id}/comments/{commentID}",
		middleware.Chain(
			e.Service.RemoveComment(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
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
		),
	).Methods("POST", "OPTIONS")
}
