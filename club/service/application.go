package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"olympsis-server/aggregations"
	"olympsis-server/utils"
	"time"

	"github.com/gorilla/mux"
	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Get club applications
//
// - Validates club ID
// - Check query parameters for status
// - Aggregate club applications
//
// Returns: an array of application objects
func (c *Service) GetApplications() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// Validate club ID
		id := mux.Vars(r)["id"]
		oid, err := utils.ValidateObjectID(id)
		if err != nil {
			utils.HandleInvalidIDError(rw)
			c.Logger.Error("Invalid Club ID - Error: ", err.Error())
			return
		}

		// Check query parameters for status
		status := r.URL.Query().Get("status")
		if status == "" {
			status = "pending"
		}

		// Aggregate club applications
		apps, err := aggregations.AggregateClubApplications(&oid, status, c.Database)
		if err != nil {
			utils.HandleFindError(rw, err)
			c.Logger.Error(fmt.Sprintf("Failed to get club applications. ID: %s - Error: %s", id, err.Error()))
			return
		}

		// No content http return
		if len(*apps) == 0 {
			rw.WriteHeader(http.StatusNoContent)
			return
		}

		resp := models.ClubApplicationsResponse{
			TotalApplications: len(*apps),
			Applications:      *apps,
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(resp)
	}
}

// Create club application
//
// - Validates club ID
// - Validate request body
// - Check if application exists
// - Notify club admins of application
//
// Returns: the ID of the created application object
func (c *Service) CreateApplication() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		ctx, cancel := context.WithTimeout(r.Context(), time.Second*15)
		defer cancel()

		uuid := r.Header.Get("UUID")

		// Validate club ID
		id := mux.Vars(r)["id"]
		oid, err := utils.ValidateObjectID(id)
		if err != nil {
			utils.HandleInvalidIDError(rw)
			c.Logger.Error("Invalid Club ID - Error: ", err.Error())
			return
		}

		// Check for existing application
		// Return if found and no errors
		var _app models.ClubApplicationDao
		filter := bson.M{"applicant": uuid, "club_id": oid, "status": "pending"}
		err = c.Database.ClubApplicationCol.FindOne(ctx, filter).Decode(&_app)
		if err == nil {
			rw.WriteHeader(http.StatusCreated)
			rw.Write(fmt.Appendf(nil, `{ "id" : "%s" }`, _app.ID.Hex()))
			return
		}

		// If we have no existing events create a new one
		if err == mongo.ErrNoDocuments {
			timeStamp := primitive.NewDateTimeFromTime(time.Now())
			status := "pending"
			app := models.ClubApplicationDao{
				Applicant: &uuid,
				ClubID:    &oid,
				Status:    &status,
				CreatedAt: &timeStamp,
			}

			// Create new club application
			resp, err := c.Database.ClubApplicationCol.InsertOne(ctx, app)
			if err != nil {
				c.Logger.Error(fmt.Sprintf("Failed to create application. ID: %s - Error: %s ", id, err.Error()))
				http.Error(rw, `{ "msg": "something went wrong" }`, http.StatusInternalServerError)
				return
			}

			if err = c.Notification.NewApplication(&oid, &app); err != nil {
				c.Logger.Errorf("Failed to notify admins of new application. Club ID: %s - Error: %s", id, err.Error())
			}

			rw.WriteHeader(http.StatusCreated)
			rw.Write(fmt.Appendf(nil, `{ "id" : "%s" }`, resp.InsertedID.(primitive.ObjectID).Hex()))
			return
		} else {
			c.Logger.Error(fmt.Sprintf("Failed to check for application. ID: %s - Error: %s", id, err.Error()))
			http.Error(rw, `{ "msg": "something went wrong" }`, http.StatusInternalServerError)
			return
		}
	}
}

// Update club application
//
// - Validate club ID
// - Validate application ID
// - Decode request body
// - Handle application status update
// - Notify applicant of status change
//
// Returns: OK if successful
func (c *Service) UpdateApplication() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		ctx, cancel := context.WithTimeout(r.Context(), time.Second*15)
		defer cancel()

		// Validate club ID
		id := mux.Vars(r)["id"]
		oid, err := utils.ValidateObjectID(id)
		if err != nil {
			utils.HandleInvalidIDError(rw)
			c.Logger.Error("Invalid Club ID - Error: ", err.Error())
			return
		}

		// Validate application ID
		aid := mux.Vars(r)["applicationID"]
		aoid, err := utils.ValidateObjectID(aid)
		if err != nil {
			utils.HandleInvalidIDError(rw)
			c.Logger.Error("Invalid Application ID - Error: ", err.Error())
			return
		}

		// Decode request body
		var req models.UpdateStatusRequest
		err = json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
			return
		}

		// Handle accepting application
		if req.Status == models.AcceptedApplicationStatus {

			// Check for existing application
			var app models.ClubApplicationDao
			err = c.Database.ClubApplicationCol.FindOne(context.TODO(), bson.M{"_id": aoid}).Decode(&app)
			if err != nil {
				c.Logger.Error(fmt.Sprintf("Failed to find application. ID: %s - Error: %s ", id, err.Error()))
				http.Error(rw, `{ "msg": "something went wrong" }`, http.StatusInternalServerError)
				return
			}

			// Only process pending application
			if *app.Status != models.PendingApplicationStatus {
				rw.WriteHeader(http.StatusOK)
				rw.Write([]byte(`{"msg":"OK"}`))
				return
			}

			// Update club application status
			filter := bson.M{"_id": aoid}
			change := bson.M{"$set": bson.M{"status": req.Status}}
			_, err = c.Database.ClubApplicationCol.UpdateOne(context.TODO(), filter, change)
			if err != nil {
				c.Logger.Error("Failed to update application: ", err.Error())
				http.Error(rw, `{ "msg": "something went wrong" }`, http.StatusInternalServerError)
				return
			}

			// Add member to database
			member := models.MemberDao{
				ID:       primitive.NewObjectID(),                   // doc id
				UserID:   *app.Applicant,                            // user uuid
				ClubID:   &oid,                                      // club id
				Role:     string(models.MemberMember),               // user role
				JoinedAt: primitive.NewDateTimeFromTime(time.Now()), // joined date
			}
			_, err = c.InsertMember(ctx, &member)
			if err != nil {
				c.Logger.Error(fmt.Sprintf("Failed to add member to database. Club ID: %s - UserID: %s - Error: %s", id, *app.Applicant, err.Error()))
				http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
				return
			}

			// Notify the user that they've been accepted
			if err = c.Notification.ApplicationUpdate(oid, &app); err != nil {
				c.Logger.Errorf("Failed to notify user. Club ID: %s - Error: %s", id, err.Error())
			}

			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte(`{"msg":"OK"}`))
			return
		}

		// Handle application denial
		filter := bson.M{"_id": aoid}
		change := bson.M{"$set": bson.M{"status": req.Status}}
		_, err = c.Database.ClubApplicationCol.UpdateOne(context.TODO(), filter, change)
		if err != nil {
			c.Logger.Error("failed to update application: " + err.Error())
			http.Error(rw, `{ "msg": "failed to update application" }`, http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{"msg":"OK"}`))
	}
}

// Delete club application
//
// - Validate club ID
// - Validate application ID
// - Delete club application from database
//
// Returns: OK if successful
func (c *Service) DeleteApplication() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second*15)
		defer cancel()

		uuid := r.Header.Get("UUID")

		// Validate club ID
		id := mux.Vars(r)["id"]
		oid, err := utils.ValidateObjectID(id)
		if err != nil {
			utils.HandleInvalidIDError(rw)
			c.Logger.Error("Invalid Club ID - Error: ", err.Error())
			return
		}

		// Validate application ID
		aid := mux.Vars(r)["applicationID"]
		aoid, err := utils.ValidateObjectID(aid)
		if err != nil {
			utils.HandleInvalidIDError(rw)
			c.Logger.Error("Invalid Application ID - Error: ", err.Error())
			return
		}

		// Delete club application from database
		filter := bson.M{"_id": aoid, "applicant": uuid, "club_id": oid}
		_, err = c.Database.ClubApplicationCol.DeleteOne(ctx, filter)
		if err != nil {
			c.Logger.Error(fmt.Sprintf("Failed to delete club application. ID: %s - Error: %s", id, err.Error()))
			http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{"msg": "OK"}`))
	}
}
