package grpcapi

import (
	"context"
	"time"

	"olympsis-server/grpcapi/invitepb"

	"github.com/olympsis/models"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// InviteClient calls invite-service's InviteService over gRPC. The check-in
// handler uses it to embed a user's pending invites in the check-in response,
// so the app gets them on the same round trip it already makes at launch.
//
// invite-service owns the invites collection; this is a read-only view of it.
type InviteClient struct {
	conn   *grpc.ClientConn
	client invitepb.InviteServiceClient
	logger *logrus.Logger
}

// NewInviteClient dials addr (host:port) with plaintext transport. This is an
// internal-network call, not exposed through the public gateway, mirroring
// invite-service's own EventTeam client. grpc.NewClient connects lazily, so this
// returns immediately and only errors on a malformed target — an unreachable
// invite-service surfaces later, per-call, and is handled as a soft failure by
// the caller.
func NewInviteClient(addr string, logger *logrus.Logger) (*InviteClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &InviteClient{
		conn:   conn,
		client: invitepb.NewInviteServiceClient(conn),
		logger: logger,
	}, nil
}

// GetUserInvites returns one page of the user's invites, newest first, filtered
// by status ("" means all). The cursor is deliberately not plumbed through:
// check-in only ever wants the first page, and callers that need pagination use
// invite-service's REST endpoint directly.
func (c *InviteClient) GetUserInvites(ctx context.Context, userID string, status models.InviteStatus, limit int32) ([]models.InviteResponse, error) {
	resp, err := c.client.GetUserInvites(ctx, &invitepb.GetUserInvitesRequest{
		UserId: userID,
		Status: modelStatusToProto(status),
		Limit:  limit,
	})
	if err != nil {
		return nil, err
	}

	invites := make([]models.InviteResponse, 0, len(resp.GetInvites()))
	for _, inv := range resp.GetInvites() {
		invites = append(invites, protoToModelInvite(inv))
	}
	return invites, nil
}

// Close tears down the client connection.
func (c *InviteClient) Close() error {
	return c.conn.Close()
}

// ---------------------------------------------------------------------------
// proto -> model conversions
//
// The inverse of invite-service's modelToProtoInvite. Producing the same
// models.InviteResponse its REST handler returns means the JSON the client sees
// from check-in is identical to what GET /v1/invites/user/{id} would give it.
// ---------------------------------------------------------------------------

func protoToModelInvite(inv *invitepb.Invite) models.InviteResponse {
	return models.InviteResponse{
		ID:          inv.GetId(),
		Type:        protoTypeToModel(inv.GetType()),
		ContextID:   inv.GetContextId(),
		InviteeID:   inv.GetInviteeId(),
		RequestorID: inv.GetRequestorId(),
		Status:      protoStatusToModel(inv.GetStatus()),
		// The proto carries unix milliseconds, matching bson.DateTime's native
		// storage. ArchivedAt has no proto field, so it stays the zero time —
		// which is also what invite-service's own toInviteResponse emits.
		CreatedAt: time.UnixMilli(inv.GetCreatedAt()).UTC(),
		UpdatedAt: time.UnixMilli(inv.GetUpdatedAt()).UTC(),
	}
}

func protoTypeToModel(t invitepb.InviteType) models.InviteType {
	switch t {
	case invitepb.InviteType_EVENT:
		return models.InviteTypeEvent
	case invitepb.InviteType_TEAM:
		return models.InviteTypeTeam
	case invitepb.InviteType_CLUB:
		return models.InviteTypeClub
	case invitepb.InviteType_ORG:
		return models.InviteTypeOrg
	default:
		return ""
	}
}

func protoStatusToModel(st invitepb.InviteStatus) models.InviteStatus {
	switch st {
	case invitepb.InviteStatus_PENDING:
		return models.InviteStatusPending
	case invitepb.InviteStatus_ACCEPTED:
		return models.InviteStatusAccepted
	case invitepb.InviteStatus_DECLINED:
		return models.InviteStatusDeclined
	default:
		return ""
	}
}

// modelStatusToProto maps a model status filter onto the proto enum; "" becomes
// UNSPECIFIED, which invite-service treats as "all statuses".
func modelStatusToProto(st models.InviteStatus) invitepb.InviteStatus {
	switch st {
	case models.InviteStatusPending:
		return invitepb.InviteStatus_PENDING
	case models.InviteStatusAccepted:
		return invitepb.InviteStatus_ACCEPTED
	case models.InviteStatusDeclined:
		return invitepb.InviteStatus_DECLINED
	default:
		return invitepb.InviteStatus_INVITE_STATUS_UNSPECIFIED
	}
}
