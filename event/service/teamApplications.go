package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// teamApplicationsResponse is the list shape returned to the owner. Applications
// carry the raw applicant uid (TeamApplicationDao); the client resolves the
// display name (the invite/notification it already received is enriched).
type teamApplicationsResponse struct {
	TotalApplications int                         `json:"total_applications"`
	Applications      []models.TeamApplicationDao `json:"team_applications"`
}

// CreateTeamApplication lets a user apply to a CLOSED team. It mirrors the club
// application flow: dedupe an existing pending application, otherwise insert one
// and notify the owner. The applicant becomes a member only once the owner
// approves (UpdateTeamApplication).
func (s *Service) CreateTeamApplication() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		userID := r.Header.Get("userID")
		vars := mux.Vars(r)
		eventID, teamID := vars["id"], vars["teamID"]
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

		event, err := s.FindEvent(ctx, bson.M{"_id": eventOID})
		if err != nil {
			http.Error(w, `{"msg": "event not found"}`, http.StatusNotFound)
			return
		}
		team, err := s.FindTeam(ctx, bson.M{"_id": teamOID, "event_id": eventOID})
		if err != nil {
			http.Error(w, `{"msg": "team not found"}`, http.StatusNotFound)
			return
		}
		if team.IsOpen != nil && *team.IsOpen {
			http.Error(w, `{"msg": "this team is open; join directly instead of applying"}`, http.StatusBadRequest)
			return
		}

		// One team per user per event.
		if existing, _ := s.FindTeams(ctx, bson.M{"event_id": eventOID, "members.user_id": userID}, nil); len(existing) > 0 {
			http.Error(w, `{"msg": "you are already on a team for this event"}`, http.StatusConflict)
			return
		}

		// Idempotent: return the existing pending application if there is one.
		pending := bson.M{"team_id": teamOID, "applicant": userID, "status": models.PendingApplicationStatus}
		if existing, err := s.FindTeamApplication(ctx, pending); err == nil {
			w.WriteHeader(http.StatusCreated)
			w.Write(fmt.Appendf(nil, `{"id": "%s"}`, existing.ID.Hex()))
			return
		} else if !errors.Is(err, mongo.ErrNoDocuments) {
			s.Logger.Errorf("Failed to check for existing application. Team: %s - Error: %s", teamID, err.Error())
			http.Error(w, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			return
		}

		timestamp := bson.NewDateTimeFromTime(time.Now())
		status := models.PendingApplicationStatus
		app := models.TeamApplicationDao{
			Applicant: &userID,
			TeamID:    &teamOID,
			EventID:   &eventOID,
			Status:    &status,
			CreatedAt: &timestamp,
		}
		id, err := s.InsertTeamApplication(ctx, &app)
		if err != nil {
			s.Logger.Errorf("Failed to create team application. Team: %s - Error: %s", teamID, err.Error())
			http.Error(w, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			return
		}

		// Notify the owner that there's an application to review.
		if owner, ok := teamOwnerID(team); ok {
			if err = s.Notification.TeamApplication(event, team, userID, []string{owner}); err != nil {
				s.Logger.Errorf("Failed to notify owner of application. Team: %s - Error: %s", teamID, err.Error())
			}
		}

		w.WriteHeader(http.StatusCreated)
		w.Write(fmt.Appendf(nil, `{"id": "%s"}`, id.Hex()))
	}
}

// GetTeamApplications lists a team's applications for the owner. Defaults to
// pending; pass ?status=ACCEPTED|DENIED to filter otherwise.
func (s *Service) GetTeamApplications() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		userID := r.Header.Get("userID")
		vars := mux.Vars(r)
		eventID, teamID := vars["id"], vars["teamID"]
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

		team, err := s.FindTeam(ctx, bson.M{"_id": teamOID, "event_id": eventOID})
		if err != nil {
			http.Error(w, `{"msg": "team not found"}`, http.StatusNotFound)
			return
		}
		if !isTeamOwner(team, userID) {
			http.Error(w, `{"msg": "only the team owner can view applications"}`, http.StatusForbidden)
			return
		}

		// Normalize so ?status=pending works as well as PENDING; default to pending.
		status := models.ApplicationStatus(strings.ToUpper(r.URL.Query().Get("status")))
		if status == "" {
			status = models.PendingApplicationStatus
		}

		opts := options.Find().SetSort(bson.M{"created_at": 1})
		apps, err := s.FindTeamApplications(ctx, bson.M{"team_id": teamOID, "status": status}, opts)
		if err != nil {
			s.Logger.Errorf("Failed to get team applications. Team: %s - Error: %s", teamID, err.Error())
			http.Error(w, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(teamApplicationsResponse{
			TotalApplications: len(apps),
			Applications:      apps,
		})
	}
}

// UpdateTeamApplication approves or denies a pending application (owner only).
// On approval the applicant is added to the roster (membership = RSVP); the
// applicant is notified either way.
func (s *Service) UpdateTeamApplication() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		userID := r.Header.Get("userID")
		vars := mux.Vars(r)
		eventID, teamID, applicationID := vars["id"], vars["teamID"], vars["applicationID"]
		if len(eventID) < 24 || len(teamID) < 24 || len(applicationID) < 24 {
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
		appOID, err := bson.ObjectIDFromHex(applicationID)
		if err != nil {
			http.Error(w, `{"msg": "failed to encode application id"}`, http.StatusBadRequest)
			return
		}

		var req models.UpdateStatusRequest
		if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"msg": "failed to decode request"}`, http.StatusBadRequest)
			return
		}

		event, err := s.FindEvent(ctx, bson.M{"_id": eventOID})
		if err != nil {
			http.Error(w, `{"msg": "event not found"}`, http.StatusNotFound)
			return
		}
		team, err := s.FindTeam(ctx, bson.M{"_id": teamOID, "event_id": eventOID})
		if err != nil {
			http.Error(w, `{"msg": "team not found"}`, http.StatusNotFound)
			return
		}
		if !isTeamOwner(team, userID) {
			http.Error(w, `{"msg": "only the team owner can review applications"}`, http.StatusForbidden)
			return
		}

		app, err := s.FindTeamApplication(ctx, bson.M{"_id": appOID, "team_id": teamOID})
		if err != nil {
			http.Error(w, `{"msg": "application not found"}`, http.StatusNotFound)
			return
		}
		// Only pending applications can be acted on; anything else is a no-op.
		if app.Status == nil || *app.Status != models.PendingApplicationStatus {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"msg": "OK"}`))
			return
		}
		applicant := ""
		if app.Applicant != nil {
			applicant = *app.Applicant
		}

		// Approval path: add the member, then mark the application accepted.
		if models.ApplicationStatus(req.Status) == models.AcceptedApplicationStatus {
			// Guard against approving someone who has since joined another team for
			// this event (which the unique index would otherwise reject at insert).
			if existing, _ := s.FindTeams(ctx, bson.M{"event_id": eventOID, "members.user_id": applicant}, nil); len(existing) > 0 {
				http.Error(w, `{"msg": "applicant is already on a team for this event"}`, http.StatusConflict)
				return
			}

			added, reason, err := s.AddTeamMemberViaInvite(ctx, teamOID, applicant)
			if err != nil {
				s.Logger.Errorf("Failed to add applicant to team. Team: %s - Error: %s", teamID, err.Error())
				http.Error(w, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
				return
			}
			if !added && reason == reasonTeamFull {
				http.Error(w, `{"msg": "team is full"}`, http.StatusConflict)
				return
			}

			if err = s.UpdateTeamApplicationStatus(ctx, bson.M{"_id": appOID}, models.AcceptedApplicationStatus); err != nil {
				s.Logger.Errorf("Failed to update application. Team: %s - Error: %s", teamID, err.Error())
				http.Error(w, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
				return
			}

			if err = s.Notification.TeamApplicationUpdate(event, team, applicant, true); err != nil {
				s.Logger.Errorf("Failed to notify applicant. Team: %s - Error: %s", teamID, err.Error())
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"msg": "OK"}`))
			return
		}

		// Denial path (any non-accepted target status is treated as a denial).
		if err = s.UpdateTeamApplicationStatus(ctx, bson.M{"_id": appOID}, models.DeniedApplicationStatus); err != nil {
			s.Logger.Errorf("Failed to deny application. Team: %s - Error: %s", teamID, err.Error())
			http.Error(w, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			return
		}
		if err = s.Notification.TeamApplicationUpdate(event, team, applicant, false); err != nil {
			s.Logger.Errorf("Failed to notify applicant. Team: %s - Error: %s", teamID, err.Error())
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"msg": "OK"}`))
	}
}

/*****************
DATABASE FUNCTIONS
******************/

func (s *Service) InsertTeamApplication(ctx context.Context, app *models.TeamApplicationDao) (bson.ObjectID, error) {
	resp, err := s.Database.EventTeamApplicationsCollection.InsertOne(ctx, app)
	if err != nil {
		return bson.NilObjectID, err
	}
	return resp.InsertedID.(bson.ObjectID), nil
}

func (s *Service) FindTeamApplication(ctx context.Context, filter bson.M) (*models.TeamApplicationDao, error) {
	var app models.TeamApplicationDao
	if err := s.Database.EventTeamApplicationsCollection.FindOne(ctx, filter).Decode(&app); err != nil {
		return nil, err
	}
	return &app, nil
}

func (s *Service) FindTeamApplications(ctx context.Context, filter bson.M, opts *options.FindOptionsBuilder) ([]models.TeamApplicationDao, error) {
	cursor, err := s.Database.EventTeamApplicationsCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	apps := []models.TeamApplicationDao{}
	if err := cursor.All(ctx, &apps); err != nil {
		return nil, err
	}
	return apps, nil
}

func (s *Service) UpdateTeamApplicationStatus(ctx context.Context, filter bson.M, status models.ApplicationStatus) error {
	_, err := s.Database.EventTeamApplicationsCollection.UpdateOne(ctx, filter, bson.M{"$set": bson.M{"status": status}})
	return err
}

// DeleteTeamApplications removes every application matching the filter. Used to
// cascade-clean a team's applications when the team is disbanded.
func (s *Service) DeleteTeamApplications(ctx context.Context, filter bson.M) error {
	_, err := s.Database.EventTeamApplicationsCollection.DeleteMany(ctx, filter)
	return err
}
