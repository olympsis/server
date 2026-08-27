package service

import (
	"olympsis-server/database"
	"olympsis-server/notifications"
	"olympsis-server/types"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

/*
Authentication Service
- reference object for auth service
*/
type Service struct {
	// database
	Database *database.Database

	// logrus logger to Log information about service and errors
	Log *logrus.Logger

	// mux Router to complete http requests
	Router *mux.Router

	// Notification service
	Notification *notifications.Service

	// Reads pending invites from invite-service for the check-in response.
	// Nil when the gRPC client isn't configured — check-in then omits invites.
	Invites types.InviteReader
}
