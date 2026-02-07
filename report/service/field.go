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

func (s *Service) CreateFieldReport() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// decode request
		var req models.VenueReportDao
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(rw, `{ "msg": "failed to decode request" }`, http.StatusBadRequest)
			return
		}
		if req.VenueID == nil || req.VenueID == &bson.NilObjectID {
			http.Error(rw, `{ "msg": "field ID not found in body" }`, http.StatusBadRequest)
			return
		}

		// set necessary data
		id := bson.NewObjectID()
		uuid := r.Header.Get("UUID")
		status := "pending"
		timestamp := bson.NewDateTimeFromTime(time.Now())
		opts := options.InsertOne()
		req.ID = &id
		req.User = &uuid
		req.Status = &status
		req.CreatedAt = &timestamp

		// insert model into database
		err = s.FieldReport.Insert(context.Background(), &req, opts)
		if err != nil {
			http.Error(rw, `{ "msg": "failed to create report" }`, http.StatusInternalServerError)
			s.Logger.Error(fmt.Sprintf(`failed to insert report: %s`, err.Error()))
			return
		}

		rw.WriteHeader(http.StatusCreated)
		rw.Write([]byte(fmt.Sprintf(`{ "id": "%s" }`, id.Hex())))
	}
}

func (s *Service) ReadFieldReports() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		status := r.URL.Query().Get("status")

		filter := bson.M{}
		opts := options.Aggregate()

		if status != "" {
			filter["status"] = status
		}

		reports, err := s.FieldReport.Find(context.Background(), bson.M{"$match": filter}, opts)
		if err != nil {
			s.Logger.Error(fmt.Sprintf("failed to find reports: %s", err.Error()))
			http.Error(rw, `{ "msg": "failed to find reports" }`, http.StatusInternalServerError)
			return
		}

		if reports == nil || len(*reports) == 0 {
			rw.WriteHeader(http.StatusNoContent)
			return
		}

		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(reports)
	}
}

func (s *Service) UpdateFieldReport() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// grab report id
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(rw, `{ "msg": "no/bad ID found in request" }`, http.StatusBadRequest)
			return
		}

		// grab body
		var req models.PostReportDao
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(rw, `{ "msg": "failed to decode request" }`, http.StatusBadRequest)
			return
		}

		// handle updates
		oid, _ := bson.ObjectIDFromHex(id)
		filter := bson.M{
			"_id": oid,
		}
		changes := bson.M{}
		update := bson.M{
			"$set": changes,
		}
		if req.Notes != nil {
			changes["notes"] = req.Notes
		}
		if req.Status != nil {
			changes["status"] = req.Status
		}

		err = s.FieldReport.Update(context.TODO(), filter, update)
		if err != nil {
			s.Logger.Error(fmt.Sprintf(`failed to update report: %s`, err.Error()))
			http.Error(rw, `{ "msg": "failed to update report" }`, http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusOK)
	}
}

func (s *Service) DeleteFieldReport() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// grab report id
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(rw, `{ "msg": "no/bad ID found in request" }`, http.StatusBadRequest)
			return
		}

		// convert id -> object id
		oid, _ := bson.ObjectIDFromHex(id)
		filter := bson.M{
			"_id": oid,
		}

		// delete transaction
		err := s.FieldReport.Delete(context.Background(), filter)
		if err != nil {
			http.Error(rw, `{ "msg": "failed to delete report" }`, http.StatusBadRequest)
			return
		}

		rw.WriteHeader(http.StatusOK)
	}
}
