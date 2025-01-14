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

/*
Get Club Applications(GET)

  - Fetches and returns a list of club applications

  - Grabs club id from path

  - Must be club Admin

Returns:

	Http handler
		- Writes applications back to client
*/
func (c *Service) GetApplications() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// Grab club id from path and validate it
		id := mux.Vars(r)["id"]
		valid := utils.ValidateClubID(id)
		if !valid {
			http.Error(rw, "Bad/Invalid club id", http.StatusBadRequest)
			return
		}

		// status of applications
		status := r.URL.Query().Get("status")
		if status == "" {
			status = "pending"
		}

		// convert club id to oid
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			c.Logger.Debug(err.Error())
		}

		apps, err := aggregations.AggregateClubApplications(&oid, status, c.Database)

		if err != nil {
			if err == mongo.ErrNoDocuments {
				rw.WriteHeader(http.StatusNoContent)
				return
			}
			c.Logger.Error(fmt.Sprintf("Failed to get club applications: %s", err.Error()))
			http.Error(rw, `{ "msg": "Failed to get club applications" }`, http.StatusInternalServerError)
			return
		}

		// just in case mongo doesn't throw an error
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

/*
Create a Club Application(POST)

  - Creates a club applications

  - Grabs club id from path

  - creates club application

Returns:

	Http handler
		- Writes back application object to user
*/
func (c *Service) CreateApplication() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		uuid := r.Header.Get("UUID")

		// Grab club id from path and validate it
		id := mux.Vars(r)["id"]
		valid := utils.ValidateClubID(id)
		if !valid {
			http.Error(rw, `{ "msg": "invalid club id" }`, http.StatusBadRequest)
			return
		}

		// Convert id to object ID
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			c.Logger.Error("Failed to convert club id: ", err.Error())
			http.Error(rw, `{ "msg": "failed to convert club id" }`, http.StatusBadRequest)
			return
		}

		// check if an application already exists
		var _app models.ClubApplicationDao
		filter := bson.M{"uuid": uuid, "club_id": oid, "status": "pending"}
		err = c.Database.ClubApplicationCol.FindOne(ctx, filter).Decode(&_app)
		if err != nil {
			// If we have no existing events create a new one
			if err == mongo.ErrNoDocuments {
				timeStamp := time.Now().Unix()
				status := "pending"
				app := models.ClubApplicationDao{
					Applicant: &uuid,
					ClubID:    &oid,
					Status:    &status,
					CreatedAt: &timeStamp,
				}

				// create club application in database
				resp, err := c.Database.ClubApplicationCol.InsertOne(ctx, app)
				if err != nil {
					c.Logger.Error("failed to create application: ", err.Error())
					http.Error(rw, `{ "msg": "failed to create application" }`, http.StatusInternalServerError)
					return
				}

				response := models.CreateResponse{
					ID: resp.InsertedID.(primitive.ObjectID).Hex(),
				}
				rw.WriteHeader(http.StatusCreated)
				json.NewEncoder(rw).Encode(response)
				return
			}

			// Other database errors
			c.Logger.Error("Failed to check for application: " + err.Error())
			http.Error(rw, `{ "msg": "failed to check application" }`, http.StatusInternalServerError)
			return
		}

		// We've found a pending application
		c.Logger.Info("Club Application already exists")
		rw.WriteHeader(http.StatusCreated)
		json.NewEncoder(rw).Encode(_app)
	}
}

/*
Update a Club Applications(PUT)

  - Updates a club applications

  - Grabs club id from path

  - Grabs application id from path

  - Update the status of the specific application

  - Must be club Admin

Returns:

	Http handler
		- Writes ok back to user
*/
func (c *Service) UpdateApplication() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// Grab club id from path and validate it
		id := mux.Vars(r)["id"]
		valid := utils.ValidateClubID(id)
		if !valid {
			http.Error(rw, "invalid club id", http.StatusBadRequest)
			return
		}

		// grab club id from path
		vars := mux.Vars(r)
		// if there is no application id
		if len(vars["applicationID"]) == 0 {
			http.Error(rw, "bad application id", http.StatusBadRequest)
			return
		}

		// if we get an invalid application id
		if len(vars["applicationID"]) < 24 {
			http.Error(rw, "bad application id", http.StatusBadRequest)
			return
		}

		var req models.UpdateStatusRequest
		// decode request
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
			return
		}

		// convert club id to oid
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			c.Logger.Debug(err.Error())
		}

		// convert application id to oid
		aid := vars["applicationID"]
		aoid, err := primitive.ObjectIDFromHex(aid)
		if err != nil {
			c.Logger.Debug(err.Error())
		}

		// if the admin accepts the application
		if req.Status == "accepted" {

			// check if application exists
			var app models.ClubApplicationDao
			filter := bson.M{"_id": aoid}
			err = c.Database.ClubApplicationCol.FindOne(context.TODO(), filter).Decode(&app)
			if err != nil {
				c.Logger.Error("failed to find application: ", err.Error())
				http.Error(rw, `{ "msg": "failed to update application" }`, http.StatusInternalServerError)
				return
			}

			// if someone else already accepted it we don't want to cause issues in user data where there are duplicated club id's
			if *app.Status == "pending" {

				// update club application in database
				filter := bson.M{"_id": aoid}
				change := bson.M{"$set": bson.M{"status": req.Status}}
				_, err = c.Database.ClubApplicationCol.UpdateOne(context.TODO(), filter, change)
				if err != nil {
					c.Logger.Error("failed to update application: ", err.Error())
					http.Error(rw, `{ "msg": "failed to update application" }`, http.StatusInternalServerError)
					return
				}

				// add club id to user data
				filter = bson.M{"uuid": app.Applicant}
				change = bson.M{"$push": bson.M{"clubs": oid}}
				_, err = c.Database.UserCol.UpdateOne(context.TODO(), filter, change)
				if err != nil {
					c.Logger.Error("failed to add club to user data: ", err.Error())
					http.Error(rw, `{ "msg": "failed to update application" }`, http.StatusInternalServerError)
					return
				}

				member := models.MemberDao{ // member object to put in club
					ID:       primitive.NewObjectID(), // unique member identifier
					UUID:     *app.Applicant,          // user uuid
					Role:     "member",                // user role
					JoinedAt: time.Now().Unix(),       // joined date
				}

				// update club information by adding member in the list
				filter = bson.M{"_id": oid}
				change = bson.M{"$push": bson.M{"members": member}}
				_, err = c.Database.ClubCol.UpdateOne(context.TODO(), filter, change)
				if err != nil {
					c.Logger.Error("failed to update club: ", err.Error())
					http.Error(rw, `{ "msg": "failed to update application" }`, http.StatusInternalServerError)
					return
				}

				// find club info
				// club, err := aggregations.AggregateClub(&oid, c.Database)
				// if err != nil {
				// 	c.Logger.Error("failed to find club: ", err.Error())
				// 	http.Error(rw, `{ "msg": "failed to update application" }`, http.StatusInternalServerError)
				// 	return
				// }

				// find user device token
				// usr, err := c.SearchService.SearchUserByUUID(member.UUID)
				// if err != nil {
				// 	c.Logger.Error("failed to get user data: " + err.Error())
				// }

				// notify user they were accepted to the club
				// notification := models.Notification{
				// 	Title: fmt.Sprintf("[%s]Application", club.Name),
				// 	Body:  club.Name + "Accepted your application!",
				// 	Data:  club,
				// }

				// err = utils.AddTokenToTopic(club.ID.Hex(), usr.UUID)
				// if err != nil {
				// 	c.Logger.Error("failed add token to topic: ", err.Error())
				// }

				// if usr.DeviceTokens != nil {
				// 	for i := range *usr.DeviceTokens {
				// 		tokens := *usr.DeviceTokens
				// 		token := tokens[i]
				// 		err = utils.SendNotificationToToken(token, &notification)
				// 		if err != nil {
				// 			c.Logger.Error("failed send notification to token: ", err.Error())
				// 		}
				// 	}
				// }

				rw.WriteHeader(http.StatusOK)
				return
			}

			rw.WriteHeader(http.StatusOK)
			return
		} else {
			// update club application in database
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
			return
		}
	}
}

/*
Delete a Club Applications(DELETE)

  - removes a club application

  - Grabs club id from path

  - Grabs application id from path

  - delete club application

Returns:

	Http handler
		- Writes ok back to user
*/
func (c *Service) DeleteApplication() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// Check & Validate Auth Token
		uuid := r.Header.Get("UUID")

		// Grab club id from path and validate it
		id := mux.Vars(r)["id"]
		valid := utils.ValidateClubID(id)
		if !valid {
			http.Error(rw, "invalid club id", http.StatusBadRequest)
			return
		}
		// grab application id from path
		vars := mux.Vars(r)

		// if there is no application id
		if len(vars["applicationId"]) == 0 {
			http.Error(rw, "bad application id", http.StatusBadRequest)
			return
		}

		// if we get an invalid application id
		if len(vars["applicationId"]) < 24 {
			http.Error(rw, "bad application id", http.StatusBadRequest)
			return
		}

		// convert club id to oid
		appID := vars["application_id"]
		appOID, err := primitive.ObjectIDFromHex(appID)
		if err != nil {
			c.Logger.Debug(err.Error())
		}

		clubID := vars["id"]
		clubOID, err := primitive.ObjectIDFromHex(clubID)
		if err != nil {
			c.Logger.Debug(err.Error())
		}

		filter := bson.M{"_id": appOID, "uuid": uuid, "club_id": clubOID}

		// delete club application from database
		_, err = c.Database.ClubApplicationCol.DeleteOne(context.TODO(), filter)
		if err != nil {
			c.Logger.Error(err.Error())
			http.Error(rw, "failed to delete club application", http.StatusInternalServerError)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{ "msg": "OK" }`))
	}
}
