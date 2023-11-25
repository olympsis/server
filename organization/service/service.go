package service

import (
	"context"
	"encoding/json"
	"net/http"
	"olympsis-server/database"
	"sync"
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

		// decode request
		var req models.Organization
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
			return
		}

		// new organization model
		organization := models.Organization{
			ID:           primitive.NewObjectID(),
			Name:         req.Name,
			Description:  req.Description,
			Sport:        req.Sport,
			City:         req.City,
			State:        req.State,
			Country:      req.Country,
			ImageURL:     req.ImageURL,
			ImageGallery: req.ImageGallery,
			Members:      req.Members,
			CreatedAt:    req.CreatedAt,
		}
		organization.Members[0].ID = primitive.NewObjectID()

		// insert organization into database
		err = e.InsertAnOrganization(context.Background(), &organization)
		if err != nil {
			e.Logger.Error(err.Error())
			http.Error(rw, `{ "msg": "Failed to create organization" }`, http.StatusInternalServerError)
			return
		}

		// subscribe to notifications
		e.NotifService.CreateTopic(organization.ID.Hex())

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
		var org models.Organization
		OID, _ := primitive.ObjectIDFromHex(id)
		filter := bson.D{primitive.E{Key: "_id", Value: OID}}
		err := e.FindAnOrganization(context.TODO(), filter, &org)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				http.Error(rw, `{ "msg": "organization not found" }`, http.StatusNotFound)
				return
			}
		}

		// check to see if object is empty
		if org.Name == "" {
			http.Error(rw, `{ "msg": "organization not found" }`, http.StatusNotFound)
			return
		}

		var wg sync.WaitGroup
		var clubs []models.Club

		// fetch organization members
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ptp := range org.Members {
				data, err := e.SearchService.SearchUserByUUID(org.Members[ptp].UUID)
				if err != nil {
					e.Logger.Error(err.Error())
				}
				org.Members[ptp].Data = &data
			}
		}()

		// fetch children clubs
		wg.Add(1)
		go func() {
			defer wg.Done()
			cursor, err := e.Database.ClubCol.Find(context.Background(), bson.M{"parent_id": org.ID})
			if err != nil {
				e.Logger.Error(err.Error())
				return
			}

			for cursor.Next(context.TODO()) {
				var club models.Club
				err := cursor.Decode(&club)
				if err != nil {
					e.Logger.Error(err.Error())
					return
				}
				clubs = append(clubs, club)
			}
		}()

		wg.Wait()

		if len(clubs) != 0 {
			org.Data = &models.OrganizationData{
				Children: &clubs,
			}
		}

		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(&org)
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
		var org models.Organization
		err = e.UpdateAnOrganization(context.Background(), filter, updates, &org)
		if err != nil {
			e.Logger.Error(err)
			http.Error(rw, "failed to update organization", http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(org)
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
		err := e.DeleteAnOrganization(context.Background(), filter)
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
Create a new organization
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
Get an organization
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
Get a list of organizations
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
Update an organization
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
