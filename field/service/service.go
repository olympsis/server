package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"olympsis-server/database"
	"os"
	"strconv"

	"github.com/golang-jwt/jwt"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

/*
Create new field service struct
*/
func NewFieldService(l *logrus.Logger, r *mux.Router, d *database.Database) *Service {
	return &Service{Log: l, Router: r, Database: d}
}

/*
Create Field Data (POST)

  - Creates new field for olympsis

  - Grab request body

  - Create field data in user databse

Returns:

	Http handler
		- Writes object back to client
*/
func (f *Service) InsertAField() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		var req Field

		// decode request
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			f.Log.Error(err.Error())
			http.Error(rw, "bad body", http.StatusBadRequest)
			return
		}

		field := Field{
			ID:        primitive.NewObjectID(),
			Owner:     req.Owner,
			Name:      req.Name,
			Notes:     req.Notes,
			Sports:    req.Sports,
			Images:    req.Images,
			Location:  req.Location,
			City:      req.City,
			State:     req.State,
			Country:   req.Country,
			Ownership: req.Ownership,
		}

		// create auth user in database
		err = f.InsertField(context.Background(), &field)
		if err != nil {
			f.Log.Error(err.Error())
			http.Error(rw, "insertion failed", http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusCreated)
		json.NewEncoder(rw).Encode(field)
	}

}

/*
Get Fields  (Get)

  - Grab params and filter fields

  - Grabs field data from database

Returns:

	Http handler
		- Writes list of fields back to client
*/
func (f *Service) GetFields() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		longitude, _ := strconv.ParseFloat(r.URL.Query().Get("longitude"), 64)
		latitude, _ := strconv.ParseFloat(r.URL.Query().Get("latitude"), 64)
		radius, _ := strconv.ParseFloat(r.URL.Query().Get("radius"), 64)

		if longitude == 0 || latitude == 0 {
			http.Error(rw, "you need longitude/latitude", http.StatusBadRequest)
			return

		}

		var fields []Field
		loc := GeoJSON{
			Type:        "Point",
			Coordinates: []float64{longitude, latitude},
		}

		filter := bson.D{
			{Key: "location",
				Value: bson.D{
					{Key: "$near", Value: bson.D{
						{Key: "$geometry", Value: loc},
						{Key: "$maxDistance", Value: radius},
					}},
				}},
		}

		err := f.FindFields(context.Background(), filter, &fields)
		if err != nil {
			f.Log.Error(err.Error())
			http.Error(rw, "failed to search fields", http.StatusInternalServerError)
			return
		}

		if len(fields) == 0 {
			http.Error(rw, "no fields found", http.StatusNoContent)
			return
		}

		resp := FieldsResponse{
			TotalFields: len(fields),
			Fields:      fields,
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(resp)
	}
}

/*
Get Field Data (Get)
-	Grab uuid from params
-	Grabs field data from database

Returns:

	Http handler
		- Writes user data back to client
*/
func (f *Service) GetAField() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// grab club id from path
		vars := mux.Vars(r)
		if len(vars["id"]) == 0 || len(vars["id"]) < 24 {
			http.Error(rw, "bad field id", http.StatusBadRequest)
			return
		}

		id := vars["id"]

		// find field data in database
		var field Field
		oid, _ := primitive.ObjectIDFromHex(id)
		filter := bson.D{primitive.E{Key: "_id", Value: oid}}
		err := f.FindField(context.Background(), filter, &field)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				http.Error(rw, "field not found", http.StatusNotFound)
				return
			}
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(field)
	}
}

/*
Update Field Data (PUT)

  - Updates field data

  - Grab parameters and update

Returns:

	Http handler
		- Writes updated field back to client
*/
func (f *Service) UpdateAField() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		var req Field

		// decode request
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(rw, "bad body", http.StatusBadRequest)
			return
		}

		// grab club id from path
		vars := mux.Vars(r)
		if len(vars["id"]) == 0 || len(vars["id"]) < 24 {
			http.Error(rw, "bad field id", http.StatusBadRequest)
			return
		}

		id := vars["id"]
		oid, _ := primitive.ObjectIDFromHex(id)
		filter := bson.D{primitive.E{Key: "_id", Value: oid}}
		changes := bson.M{}
		updates := bson.M{"$set": changes}

		if req.Owner != "" {
			changes["owner"] = req.Owner
		}
		if req.Name != "" {
			changes["name"] = req.Name
		}
		if req.Notes != "" {
			changes["notes"] = req.Notes
		}
		if len(req.Sports) != 0 {
			changes["sports"] = req.Sports
		}
		if len(req.Images) != 0 {
			changes["images"] = req.Images
		}
		if req.Location.Type != "" {
			changes["location"] = req.Location
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
		if req.Ownership != "" {
			changes["ownership"] = req.Ownership
		}

		var field Field
		err = f.UpdateField(context.Background(), filter, updates, &field)
		if err != nil {
			f.Log.Error(err.Error())
			http.Error(rw, "failed to update field", http.StatusInternalServerError)
		}

		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(field)
	}
}

/*
Delete Field Data (Delete)

  - Updates field data

  - Grab parameters and update

Returns:

	Http handler
		- Writes token back to client
*/
func (f *Service) DeleteAField() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// grab club id from path
		vars := mux.Vars(r)
		if len(vars["id"]) == 0 || len(vars["id"]) < 24 {
			http.Error(rw, "bad field id", http.StatusBadRequest)
			return
		}

		id := vars["id"]
		oid, _ := primitive.ObjectIDFromHex(id)

		filter := bson.D{primitive.E{Key: "_id", Value: oid}}
		err := f.DeleteField(context.Background(), filter)
		if err != nil {
			f.Log.Debug(err.Error())
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`OK`))
	}
}

/*
Decode an Authentication Token
  - Decodes auth token
  - uses go jwt

Args:

	token - string of token

Returns:

	claims - jwt claims
	error -  if there is an error return error else nil
*/
func (f *Service) DecodeToken(token string) (jwt.MapClaims, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("KEY")), nil
	})

	if err != nil {
		return nil, err
	} else {
		return claims, nil
	}
}

/*
Grab Token from request
Args:

	r - http request

Returns:

	string - token
	error -  if there is an error return error else nil
*/
func (f *Service) GrabToken(r *http.Request) (string, error) {
	bearerToken := r.Header.Get("Authorization")

	if bearerToken == "" {
		return "", errors.New("no token found")
	}

	return bearerToken, nil
}
