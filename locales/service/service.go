package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"olympsis-server/database"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/*
Organization Service Struct
  - Database pointer
  - Logger pointer
  - Router pointer
*/
type Service struct {
	Database *database.Database
	Logger   *logrus.Logger
	Router   *mux.Router
}

/*
Creates a new instance of the locale service
*/
func NewLocaleService(l *logrus.Logger, r *mux.Router, d *database.Database) *Service {
	return &Service{Logger: l, Router: r, Database: d}
}

func (s *Service) GetCountries() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		countries, err := s.FindCountries(context.TODO(), bson.M{})
		if err != nil {
			s.Logger.Error(`Failed to get countries. Error: ` + err.Error())
			http.Error(w, fmt.Sprintf(`{ "msg": "%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(countries)
	}
}

func (s *Service) GetAdministrativeAreas() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(w, `{ "msg": "no/bad country id found in request" }`, http.StatusBadRequest)
			return
		}

		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			s.Logger.Error(fmt.Sprintf(`Failed to create object id from URL: %s`, err.Error()))
			http.Error(w, `{ "msg": "Failed to create object id from URL" }`, http.StatusBadRequest)
			return
		}

		adminAreas, err := s.FindAdministrativeAreas(context.TODO(), bson.M{"country_id": oid})
		if err != nil {
			s.Logger.Error(fmt.Sprintf(`Failed to get administrative areas. Error: %s`, err.Error()))
			http.Error(w, `{ "msg": "Failed to get administrative areas." }`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(adminAreas)
	}
}

func (s *Service) GetSubAdministrativeAreas() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(w, `{ "msg": "no/bad admin area id found in request" }`, http.StatusBadRequest)
			return
		}

		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			s.Logger.Error(fmt.Sprintf(`Failed to create object id from URL: %s`, err.Error()))
			http.Error(w, `{ "msg": "Failed to create object id from URL" }`, http.StatusBadRequest)
			return
		}

		subAdminAreas, err := s.FindSubAdministrativeAreas(context.TODO(), bson.M{"admin_area_id": oid})
		if err != nil {
			s.Logger.Error(fmt.Sprintf(`Failed to get sub administrative areas. Error: %s`, err.Error()))
			http.Error(w, `{ "msg": "Failed to get sub administrative areas." }`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(subAdminAreas)
	}
}
