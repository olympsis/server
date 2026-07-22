// Package grpcapi hosts the server's internal, inter-service gRPC endpoints.
//
// Today it exposes a single service, EventTeamService, whose sole caller is
// invite-service: when a user accepts a TEAM invite, invite-service calls
// AddTeamMember here so the main server (the only writer of the eventTeams
// collection) adds them to the roster. Team membership IS the RSVP, so this is
// what turns an accepted invite into an attendee.
//
// This listener is NOT exposed through the public gateway — it is reachable only
// by other olympsis services on the internal network.
package grpcapi

import (
	"context"

	"olympsis-server/grpcapi/eventteampb"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/v2/bson"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TeamMemberAdder is the behaviour the event service provides. Kept as an
// interface so this package doesn't depend on the whole event service and can be
// tested in isolation.
type TeamMemberAdder interface {
	// AddTeamMemberViaInvite adds userID to the team. added is false (with a
	// reason and no error) for the benign idempotent cases: already a member, the
	// team is full, or the team no longer exists.
	AddTeamMemberViaInvite(ctx context.Context, teamID bson.ObjectID, userID string) (added bool, reason string, err error)
}

// EventTeamServer implements eventteampb.EventTeamServiceServer.
type EventTeamServer struct {
	eventteampb.UnimplementedEventTeamServiceServer
	adder  TeamMemberAdder
	logger *logrus.Logger
}

func NewEventTeamServer(adder TeamMemberAdder, logger *logrus.Logger) *EventTeamServer {
	return &EventTeamServer{adder: adder, logger: logger}
}

// AddTeamMember validates the request and delegates to the event service. A
// benign "not added" outcome (already a member / full / team gone) returns a
// normal response with added=false; only a genuine failure returns a gRPC error,
// which invite-service treats as retryable (it leaves the invite PENDING).
func (s *EventTeamServer) AddTeamMember(ctx context.Context, req *eventteampb.AddTeamMemberRequest) (*eventteampb.AddTeamMemberResponse, error) {
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	teamID, err := bson.ObjectIDFromHex(req.GetTeamId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid team_id")
	}

	added, reason, err := s.adder.AddTeamMemberViaInvite(ctx, teamID, req.GetUserId())
	if err != nil {
		s.logger.Errorf("gRPC AddTeamMember failed. Team: %s - Error: %s", req.GetTeamId(), err.Error())
		return nil, status.Error(codes.Internal, "failed to add team member")
	}
	return &eventteampb.AddTeamMemberResponse{Added: added, Reason: reason}, nil
}

// NewGRPCServer builds a *grpc.Server with EventTeamService registered on it.
func NewGRPCServer(adder TeamMemberAdder, logger *logrus.Logger) *grpc.Server {
	srv := grpc.NewServer()
	eventteampb.RegisterEventTeamServiceServer(srv, NewEventTeamServer(adder, logger))
	return srv
}
