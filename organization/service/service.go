package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"olympsis-server/database"
	"olympsis-server/utils"
	"time"

	"github.com/gorilla/mux"
	"github.com/olympsis/models"
	"github.com/olympsis/notif"
	"github.com/olympsis/search"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

/*
Organization Service Struct
*/
type Service struct {
	Database *database.Database

	// logrus logger to Logger information about service and errors
	Logger *logrus.Logger

	// mux Router to complete http requests
	Router *mux.Router

	// notif service
	NotifService *notif.Service

	// search service
	SearchService *search.Service
}

/*
Create new field service struct
*/
func NewOrganizationService(l *logrus.Logger, r *mux.Router, d *database.Database, n *notif.Service, sh *search.Service) *Service {
	return &Service{Logger: l, Router: r, Database: d, NotifService: n, SearchService: sh}
}

/*
	ORGANIZATION
*/

/*
Create a new organization
*/
func (e *Service) CreateOrganization() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		uuid := r.Header.Get("UUID")

		// decode request
		var req models.OrganizationDao
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
			return
		}

		timeStamp := time.Now().Unix()

		// creator of the organization
		member := models.MemberDao{
			ID:       primitive.NewObjectID(),
			UUID:     uuid,
			Role:     "manager",
			JoinedAt: timeStamp,
		}
		members := []models.MemberDao{member}
		// new organization model
		organization := models.OrganizationDao{
			Name:         req.Name,
			Description:  req.Description,
			Sport:        req.Sport,
			City:         req.City,
			State:        req.State,
			Country:      req.Country,
			ImageURL:     req.ImageURL,
			ImageGallery: req.ImageGallery,
			Members:      &members,
			CreatedAt:    &timeStamp,
		}

		// insert organization into database
		id, err := e.InsertAnOrganization(context.Background(), &organization)
		if err != nil {
			e.Logger.Error(err.Error())
			http.Error(rw, `{ "msg": "Failed to create organization" }`, http.StatusInternalServerError)
			return
		}

		// update user data
		update := bson.M{
			"$push": bson.M{
				"organizations": id,
			},
		}
		e.Database.UserCol.UpdateOne(context.Background(), bson.M{"uuid": uuid}, update)

		// subscribe to notifications
		e.NotifService.CreateTopic(id.Hex())
		e.NotifService.AddTokenToTopic(id.Hex(), uuid)

		// return created organization
		rw.WriteHeader(http.StatusCreated)
		json.NewEncoder(rw).Encode(organization)
	}
}

/*
Get an organization
*/
func (e *Service) GetOrganization() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// grab organization id from path
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "no organization id found in request" }`))
			return
		}

		// find organization data in database
		OID, _ := primitive.ObjectIDFromHex(id)
		filter := bson.D{primitive.E{Key: "_id", Value: OID}}
		org, err := e.FindAnOrganization(context.TODO(), filter)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				http.Error(rw, `{ "msg": "organization not found" }`, http.StatusNotFound)
				return
			}
		}

		// check to see if object is empty
		if org.Name == nil {
			http.Error(rw, `{ "msg": "organization not found" }`, http.StatusNotFound)
			return
		}
		// FIXME
	}
}

/*
Get a list of organizations
*/
func (e *Service) GetOrganizations() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// query param
		state := r.URL.Query().Get("state")
		country := r.URL.Query().Get("country")
		filter := bson.M{
			"state":   state,
			"country": country,
		}

		if state == "" || country == "" {
			http.Error(rw, `{ "msg": "You need a state and a country to query organizations" }`, http.StatusBadRequest)
			return
		}

		// fetch organizations
		var orgs []models.Organization
		e.FindOrganizations(context.Background(), filter, &orgs)
		if len(orgs) == 0 {
			http.Error(rw, "no organizations", http.StatusNoContent)
			return
		}

		resp := models.OrganizationsResponse{
			TotalEvents:   len(orgs),
			Organizations: orgs,
		}

		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(resp)
	}
}

/*
Update an organization
*/
func (e *Service) UpdateOrganization() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// grab event id from path
		vars := mux.Vars(r)
		if len(vars["id"]) < 24 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "no organization ID found in request." }`))
			return
		}
		id := vars["id"]

		// decode request
		var req models.Organization
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
			return
		}

		// handle changes here
		oid, _ := primitive.ObjectIDFromHex(id)
		filter := bson.M{"_id": oid}
		changes := bson.M{}
		updates := bson.M{"$set": changes}

		if req.Name != "" {
			changes["name"] = req.Name
		}
		if req.Description != "" {
			changes["description"] = req.Description
		}
		if req.City != "" {
			changes["city"] = req.City
		}
		if req.State != "" {
			changes["state"] = req.State
		}
		if req.Country != "" {
			changes["country"] = req.Country
		}
		if req.ImageURL != "" {
			changes["image_url"] = req.ImageURL
		}
		if len(req.ImageGallery) != 0 {
			changes["image_gallery"] = req.ImageGallery
		}
		if req.PinnedPostID.Hex() != "" {
			changes["pinned_post_id"] = req.PinnedPostID
		}

		// update and return updated organization
		err = e.UpdateAnOrganization(context.Background(), filter, updates)
		if err != nil {
			e.Logger.Error(err)
			http.Error(rw, "failed to update organization", http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusOK)
	}
}

/*
Delete an organization
*/
func (e *Service) DeleteOrganization() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// grab id from path
		vars := mux.Vars(r)
		if len(vars["id"]) < 24 {
			http.Error(rw, `{ "msg": "bad organization id" }`, http.StatusBadRequest)
			return
		}
		id := vars["id"]

		oid, _ := primitive.ObjectIDFromHex(id)
		filter := bson.M{"_id": oid}
		org, err := e.FindAnOrganization(context.Background(), filter)
		if err != nil {
			e.Logger.Debug(fmt.Sprintf(`failed to find org: %s`, err.Error()))
		}
		members := *org.Members

		// delete org from users data
		for i := 0; i < len(members); i++ {
			filter := bson.M{"uuid": members[i].UUID}
			update := bson.M{"$pull": bson.M{"organizations": oid}}
			e.Database.UserCol.UpdateOne(context.Background(), filter, update)
		}

		err = e.DeleteAnOrganization(context.Background(), filter)
		if err != nil {
			e.Logger.Debug(err.Error())
			http.Error(rw, `{ "msg": "failed to delete event" }`, http.StatusInternalServerError)
		}

		// delete notification topic
		e.NotifService.DeleteTopic(id)
		rw.WriteHeader(http.StatusOK)
	}
}

/*
	APPLICATION
*/

/*
Create a new application
*/
func (e *Service) CreateApplication() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// decode request
		var req models.OrganizationApplication
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
			return
		}

		// check for an existing application
		var application models.OrganizationApplication
		err = e.Database.OrgApplicationCol.FindOne(context.Background(), bson.M{"club_id": req.ClubID, "organization_id": req.OrganizationID}).Decode(&application)
		if err == nil {
			rw.WriteHeader(http.StatusOK)
			json.NewEncoder(rw).Encode(application)
			return
		}

		// insert application into database
		req.ID = primitive.NewObjectID()
		req.Status = "pending"
		req.CreatedAt = time.Now().Unix()
		err = e.InsertApplication(context.Background(), &req)
		if err != nil {
			e.Logger.Error(err.Error())
			http.Error(rw, `{ "msg": "failed to create application" }`, http.StatusInternalServerError)
			return
		}

		// notify org members
		note := notif.Notification{
			Title: "New Application",
			Body:  "You've recieved an application",
			Topic: application.OrganizationID.Hex(),
		}
		e.NotifService.SendNotificationToTopic(&note)

		rw.WriteHeader(http.StatusCreated)
		json.NewEncoder(rw).Encode(req)
	}
}

/*
Get an application
*/
func (e *Service) GetApplication() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// grab id from path
		vars := mux.Vars(r)
		if len(vars["id"]) < 24 {
			http.Error(rw, `{ "msg": "bad application id" }`, http.StatusBadRequest)
			return
		}
		oid, _ := primitive.ObjectIDFromHex(vars["id"])

		var application models.OrganizationApplication
		err := e.FindApplication(context.Background(), bson.M{"_id": oid}, &application)
		if err != nil {
			http.Error(rw, `{ "msg": "failed to get application" }`, http.StatusNotFound)
			return
		}

		// if we don't get anything
		if application.Status == "" {
			http.Error(rw, `{ "msg": "failed to find organization application" }`, http.StatusNotFound)
			return
		}

		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(application)
	}
}

/*
Get a list of applications
*/
func (e *Service) GetApplications() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// grab id from path
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "bad id found in request" }`))
			return
		}

		oid, _ := primitive.ObjectIDFromHex(id)

		var applications []models.OrganizationApplication
		err := e.FindApplications(context.Background(), bson.M{"organization_id": oid, "status": "pending"}, &applications)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				rw.WriteHeader(http.StatusNoContent)
				return
			}
		}

		if len(applications) == 0 {
			rw.WriteHeader(http.StatusNoContent)
			return
		}

		// fetch club data for each application
		for i := range applications {
			var club models.Club
			err = e.Database.ClubCol.FindOne(context.Background(), bson.M{"_id": applications[i].ClubID}).Decode(&club)
			if err != nil {
				e.Logger.Error(err.Error())
			}
			applications[i].Data = &models.OrganizationApplicationData{
				Club: &club,
			}
		}

		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(applications)
	}
}

/*
Update an application
*/
func (e *Service) UpdateApplication() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// grab id from path
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(rw, `{ "msg": "bad application id" }`, http.StatusBadRequest)
			return
		}
		oid, _ := primitive.ObjectIDFromHex(id)

		var req models.OrganizationApplication
		json.NewDecoder(r.Body).Decode(&req)

		// update the club's parent id
		if req.Status == "accepted" {
			filter := bson.M{
				"_id": req.ClubID,
			}
			updates := bson.M{
				"$set": bson.M{
					"parent_id": req.OrganizationID,
				},
			}
			e.Database.ClubCol.UpdateOne(context.Background(), filter, updates)
			// maybe notify club admins that their application was approved.
		}

		err := e.UpdateAnApplication(context.Background(), bson.M{"_id": oid}, bson.M{"$set": bson.M{"status": req.Status}}, &req)
		if err != nil {
			http.Error(rw, `{ "msg": "failed to update application" }`, http.StatusInternalServerError)
			e.Logger.Error(err.Error())
			return
		}

		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(req)
	}
}

/*
Delete an organization application
*/
func (e *Service) DeleteApplication() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// grab id from path
		vars := mux.Vars(r)
		if len(vars["id"]) < 24 {
			http.Error(rw, `{ "msg": "bad application id" }`, http.StatusBadRequest)
			return
		}
		oid, _ := primitive.ObjectIDFromHex(vars["id"])

		err := e.DeleteAnApplication(context.Background(), bson.M{"_id": oid})
		if err != nil {
			http.Error(rw, `{"msg": "failed to delete application"}`, http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusOK)
	}
}

/*
	INVITATION
*/

/*
Creates an invitation object
*/
func (e *Service) CreateInvitation() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		uuid := r.Header.Get("UUID")

		// decode request
		var req models.Invitation
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
			return
		}

		filter := bson.M{
			"recipient":  uuid,
			"subject_id": req.SubjectID,
		}

		// check to see if invitation already exists
		var invitation models.Invitation
		err = e.FindAnInvitation(context.TODO(), filter, &invitation)
		if err == nil {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(invitation)
			return
		}

		// insert application into database
		req.ID = primitive.NewObjectID()
		req.CreatedAt = time.Now().Unix()
		err = e.InsertAnInvitation(context.TODO(), &req)
		if err != nil {
			e.Logger.Error(err.Error())
			http.Error(w, `{ "msg": "failed to create invitation" }`, http.StatusInternalServerError)
			return
		}

		// fetch user data
		user, err := e.SearchService.SearchUserByUUID(req.Recipient)
		if err != nil {
			e.Logger.Error("Failed to fetch user data: " + err.Error())
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(req)
			return
		}

		// fetch organization data
		var org models.Organization
		err = e.Database.OrgCol.FindOne(context.TODO(), bson.M{"_id": req.SubjectID}).Decode(&org)
		if err != nil {
			e.Logger.Error("Failed to fetch organization data: " + err.Error())
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(req)
			return
		}

		// notify user
		note := notif.Notification{
			Title: "New Invitation",
			Body:  "You've been invited to join the " + org.Name + " organization",
			Data:  org,
		}
		e.NotifService.SendNotificationToToken(&note, user.DeviceToken)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(req)
	}
}

/*
Gets an invitation object
*/
func (e *Service) GetInvitation() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// grab id from path
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(w, `{ "msg": "bad invitation id" }`, http.StatusBadRequest)
			return
		}
		oid, _ := primitive.ObjectIDFromHex(id)

		// find invitation document
		var invitation models.Invitation
		err := e.FindAnInvitation(context.Background(), bson.M{"_id": oid}, &invitation)
		if err != nil {
			http.Error(w, `{ "msg": "invitation not found" }`, http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(invitation)
	}
}

/*
Get invitations of an organization
*/
func (e *Service) GetInvitations() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// grab id from path
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(w, `{ "msg": "bad organization id" }`, http.StatusBadRequest)
			return
		}
		oid, _ := primitive.ObjectIDFromHex(id)

		var invitations []models.Invitation
		err := e.FindInvitations(context.TODO(), bson.M{"subject_id": oid}, &invitations)
		if err != nil {
			e.Logger.Error("Failed to find invitations: " + err.Error())
			http.Error(w, `{"msg": "failed to find invitations"}`, http.StatusNoContent)
		}

		if len(invitations) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// fetch org data
		var org models.Organization
		err = e.Database.OrgCol.FindOne(context.TODO(), bson.M{"_id": oid}).Decode(&org)
		if err != nil {
			e.Logger.Error("Failed to find organization: " + err.Error())
			http.Error(w, `{"msg": "failed to get organization data"}`, http.StatusInternalServerError)
			return
		}

		// fetch club data for each application
		for i := range invitations {
			invitations[i].Data = &models.InvitationData{
				Organization: &org,
			}
		}

		response := models.InvitationsResponse{
			TotalInvitations: len(invitations),
			Invitations:      invitations,
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

/*
Update an invitation
*/
func (e *Service) UpdateInvitation() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// grab id from path
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(w, `{ "msg": "bad invitation id" }`, http.StatusBadRequest)
			return
		}
		oid, _ := primitive.ObjectIDFromHex(id)

		var req models.Invitation
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, `{"msg": "Failed to decode request"}`, http.StatusBadRequest)
			return
		}

		if req.Status == "accepted" {
			member := models.MemberDao{
				ID:       primitive.NewObjectID(),
				UUID:     req.Recipient,
				Role:     "manager",
				JoinedAt: time.Now().Unix(),
			}
			changes := bson.M{
				"$push": bson.M{"members": member},
			}
			_, err = e.Database.OrgCol.UpdateOne(context.TODO(), bson.M{"_id": req.SubjectID}, changes)
			if err != nil {
				e.Logger.Error("Failed to add user to organization: " + err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

		// We update the invitation only after we've added the user into the org
		// If this fails we want the user to be able to try again
		filter := bson.M{
			"_id": oid,
		}
		updates := bson.M{
			"$set": bson.M{
				"status": req.Status,
			},
		}
		err = e.UpdateAnInvitation(context.Background(), filter, updates, &req)
		if err != nil {
			e.Logger.Error("Failed to update invitation: " + err.Error())
			http.Error(w, `{"msg": "failed to update invitation"}`, http.StatusInternalServerError)
			return
		}

		// Update user data to have the organization
		filter = bson.M{
			"uuid": req.Recipient,
		}
		updates = bson.M{
			"$push": bson.M{
				"organizations": req.SubjectID,
			},
		}
		_, err = e.Database.UserCol.UpdateOne(context.TODO(), filter, updates)
		if err != nil {
			e.Logger.Error("Failed to update user data: " + err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// fetch user data
		usr, err := e.SearchService.SearchUserByUUID(req.Recipient)
		if err != nil {
			e.Logger.Error("Failed to find user data: " + err.Error())
		}

		// notify club admins
		note := notif.Notification{
			Title: "Invitation Status",
			Body:  usr.Username + " " + req.Status + " their invite.",
			Topic: req.SubjectID.Hex(),
		}
		err = e.NotifService.SendNotificationToTopic(&note)
		if err != nil {
			e.Logger.Error("Failed to send notification: " + err.Error())
		}
		err = e.NotifService.AddTokenToTopic(req.SubjectID.Hex(), req.Recipient)
		if err != nil {
			e.Logger.Error("Failed to add token to topic: " + err.Error())
		}
		w.WriteHeader(http.StatusOK)
	}
}

/*
Delete an invitation
*/
func (e *Service) DeleteInvitation() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// grab id from path
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(w, `{ "msg": "bad invitation id" }`, http.StatusBadRequest)
			return
		}
		oid, _ := primitive.ObjectIDFromHex(id)

		err := e.DeleteAnInvitation(context.Background(), bson.M{"_id": oid})
		if err != nil {
			e.Logger.Error("Failed to delete invitation: " + err.Error())
			http.Error(w, `{"msg": "Failed to delete invitation"}`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

// CLUB POST ENDPOINTS

func (s *Service) PinOrgPost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Grab club id from path and validate it
		id := mux.Vars(r)["id"]
		valid := utils.ValidateClubID(id)
		if !valid {
			http.Error(w, "invalid org id", http.StatusBadRequest)
			return
		}

		// grab post id from path
		postID := mux.Vars(r)["postID"]
		if len(postID) < 24 {
			http.Error(w, "bad post id", http.StatusBadRequest)
			return
		}

		// update club data to reflect new post
		ok := s.PinPost(&id, &postID)
		if ok {
			w.WriteHeader(http.StatusOK)
			return
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}

func (s *Service) UnpinOrgPost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Grab club id from path and validate it
		id := mux.Vars(r)["id"]
		valid := utils.ValidateClubID(id)
		if !valid {
			http.Error(w, "invalid org id", http.StatusBadRequest)
			return
		}

		// remove pinned post from club
		ok := s.UnpinPost(&id)
		if ok {
			w.WriteHeader(http.StatusOK)
			return
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}
