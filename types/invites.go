package types

import (
	"context"

	"github.com/olympsis/models"
)

// InviteReader is the read-only view of invite-service the server needs.
// Defined in a standalone package (like StorageUploader) so the user service can
// depend on the behaviour without importing grpcapi, and so tests can supply a
// fake instead of dialing a real service.
type InviteReader interface {
	// GetUserInvites returns one page of a user's invites, newest first.
	// status "" means all statuses; limit <= 0 lets the service apply its default.
	GetUserInvites(ctx context.Context, userID string, status models.InviteStatus, limit int32) ([]models.InviteResponse, error)
}
