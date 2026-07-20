package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// CreateTeam adds a team to an event and invites the requested users to it.
//
// There is no GET counterpart on purpose: teams come back on the event itself
// via the aggregation's `teams` lookup (GET /v1/events/{id}).
//
// Invitees are NOT written onto the team. They are published on `team.created`,
// invite-service fans them into individual invite records, and a user only
// becomes a TeamMember once they accept.
func (s *Service) CreateTeam() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID := r.Header.Get("userID")

		// Grab event id from path
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(w, `{"msg": "bad event id"}`, http.StatusBadRequest)
			return
		}
		oid, err := bson.ObjectIDFromHex(id)
		if err != nil {
			s.Logger.Error("Failed to convert id to ObjectID. Error: ", err.Error())
			http.Error(w, `{"msg": "failed to encode id"}`, http.StatusBadRequest)
			return
		}

		// Decode request
		var req models.NewTeamDao
		if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.Logger.Error("Failed to decode request. Error: ", err.Error())
			http.Error(w, `{"msg":"failed to decode request"}`, http.StatusBadRequest)
			return
		}
		if req.Team.Name == nil || *req.Team.Name == "" {
			http.Error(w, `{"msg": "team name is required"}`, http.StatusBadRequest)
			return
		}

		// Confirm the event exists before writing a team that would dangle.
		event, err := s.FindEvent(ctx, bson.M{"_id": oid})
		if err != nil {
			s.Logger.Error("Failed to find event. Error: ", err.Error())
			http.Error(w, `{"msg": "event not found"}`, http.StatusNotFound)
			return
		}

		// Enforce the event's team cap, mirroring how AddParticipant honours
		// MaxParticipants.
		teams, err := s.FindTeams(ctx, bson.M{"event_id": oid}, nil)
		if err != nil {
			s.Logger.Error("Failed to find event's teams. Error: ", err.Error())
			http.Error(w, `{"msg": "failed to find event's teams"}`, http.StatusInternalServerError)
			return
		}
		if event.TeamsConfig != nil && event.TeamsConfig.MaxTeams != nil && *event.TeamsConfig.MaxTeams != 0 {
			if len(teams) >= int(*event.TeamsConfig.MaxTeams) {
				http.Error(w, `{"msg": "event is at its team limit"}`, http.StatusConflict)
				return
			}
		}

		// The creator is the team's first member; everyone else has to accept an
		// invite first.
		timestamp := bson.NewDateTimeFromTime(time.Now())
		isAnonymous := false
		team := models.TeamDao{
			Name:    req.Team.Name,
			EventID: &oid,
			Members: &[]models.TeamMemberDao{{
				UserID:      &userID,
				IsAnonymous: &isAnonymous,
				JoinedAt:    &timestamp,
			}},
			CreatedAt: &timestamp,
		}

		tid, err := s.InsertTeam(ctx, &team)
		if err != nil {
			s.Logger.Error("Failed to insert team into the database. Error: ", err.Error())
			http.Error(w, `{"msg": "failed to create team"}`, http.StatusInternalServerError)
			return
		}

		// Fan the invitees out to invite records via invite-service.
		s.publishInviteRequest(ctx, models.InviteTypeTeam, tid.Hex(), userID, req.Invitees)

		w.WriteHeader(http.StatusCreated)
		w.Write(fmt.Appendf(nil, `{"id": "%s"}`, tid.Hex()))
	}
}

// RemoveTeam deletes a team from an event.
func (s *Service) RemoveTeam() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		vars := mux.Vars(r)
		eventID := vars["id"]
		teamID := vars["teamID"]
		if len(eventID) < 24 || len(teamID) < 24 {
			http.Error(w, `{"msg": "bad id"}`, http.StatusBadRequest)
			return
		}

		eventOID, err := bson.ObjectIDFromHex(eventID)
		if err != nil {
			http.Error(w, `{"msg": "failed to encode event id"}`, http.StatusBadRequest)
			return
		}
		teamOID, err := bson.ObjectIDFromHex(teamID)
		if err != nil {
			http.Error(w, `{"msg": "failed to encode team id"}`, http.StatusBadRequest)
			return
		}

		// Scope the delete by event id too, so a team id from another event can't
		// be removed through this route.
		if err = s.DeleteTeam(ctx, bson.M{"_id": teamOID, "event_id": eventOID}); err != nil {
			s.Logger.Errorf("Failed to delete team. Error: %s", err.Error())
			http.Error(w, `{"msg": "failed to remove team"}`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"msg": "OK"}`))
	}
}

/*****************
DATABASE FUNCTIONS
******************/

// Insert new team into database
func (s *Service) InsertTeam(ctx context.Context, team *models.TeamDao) (bson.ObjectID, error) {
	resp, err := s.Database.EventTeamsCollection.InsertOne(ctx, team)
	if err != nil {
		return bson.NilObjectID, err
	}
	id := resp.InsertedID.(bson.ObjectID)
	return id, nil
}

// Get team from database
func (s *Service) FindTeam(ctx context.Context, filter bson.M) (*models.TeamDao, error) {
	var team models.TeamDao
	err := s.Database.EventTeamsCollection.FindOne(ctx, filter).Decode(&team)
	if err != nil {
		return nil, err
	}
	return &team, nil
}

// Get teams from database
func (s *Service) FindTeams(ctx context.Context, filter bson.M, opts *options.FindOptionsBuilder) ([]models.TeamDao, error) {
	var teams []models.TeamDao
	cursor, err := s.Database.EventTeamsCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &teams); err != nil {
		return nil, err
	}
	return teams, nil
}

// Update team in database
func (s *Service) UpdateTeam(ctx context.Context, filter bson.M, update bson.M) error {
	_, err := s.Database.EventTeamsCollection.UpdateOne(ctx, filter, update)
	return err
}

// Delete team from database
func (s *Service) DeleteTeam(ctx context.Context, filter bson.M) error {
	_, err := s.Database.EventTeamsCollection.DeleteOne(ctx, filter)
	return err
}

// Delete multiple teams from database
func (s *Service) DeleteTeams(ctx context.Context, filter bson.M) error {
	_, err := s.Database.EventTeamsCollection.DeleteMany(ctx, filter)
	return err
}
