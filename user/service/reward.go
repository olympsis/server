package service

import (
	"context"
	"encoding/json"
	"net/http"
	"olympsis-server/models"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/*
Add Badge(POST)

  - Grab uuid from token

  - Add Badge to User Data

Returns:

	Http handler
*/
func (u *Service) AddBadge() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		token, err := u.GrabToken(r)
		if err != nil {
			u.Log.Error(err.Error())
			http.Error(rw, "Unauthorized", http.StatusUnauthorized)
		}

		claims, err := u.DecodeToken(token)
		if err != nil {
			u.Log.Error("Failed to Decode Token: " + err.Error())
			http.Error(rw, "Forbidden", http.StatusForbidden)
			return
		}

		uuid := claims["sub"].(string)

		// decode body
		var req models.Badge
		json.NewDecoder(r.Body).Decode(&req)

		badge := models.Badge{
			ID:          primitive.NewObjectID(),
			Name:        req.Name,
			Title:       req.Title,
			ImageURL:    req.ImageURL,
			EventId:     req.EventId,
			Description: req.Description,
			AchievedAt:  req.AchievedAt,
		}

		filter := bson.M{"uuid": uuid}
		changes := bson.M{"$push": bson.M{"badges": badge}}

		_, err = u.Database.UserCol.UpdateOne(context.TODO(), filter, changes)
		if err != nil {
			u.Log.Debug(err.Error())
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(badge)
	}
}

/*
Remove Badge(DELETE)

  - Grab uuid from token

  - Grab id from path

  - Remove badge from user data

Returns:

	Http handler
*/
func (u *Service) RemoveBadge() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		token, err := u.GrabToken(r)
		if err != nil {
			u.Log.Error(err.Error())
			http.Error(rw, "Unauthorized", http.StatusUnauthorized)
		}

		claims, err := u.DecodeToken(token)
		if err != nil {
			u.Log.Error("Failed to Decode Token: " + err.Error())
			http.Error(rw, "Forbidden", http.StatusForbidden)
			return
		}

		uuid := claims["sub"].(string)

		// grab badge id from path
		vars := mux.Vars(r)

		// if there is no badge id
		if len(vars["id"]) == 0 || len(vars["id"]) < 24 {
			http.Error(rw, "Bad Badge ID", http.StatusBadRequest)
			return
		}

		// convert badge id to oid
		id := vars["id"]
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			u.Log.Debug(err.Error())
		}

		filter := bson.M{"uuid": uuid}
		changes := bson.M{"$pull": bson.M{"badges": bson.M{"_id": oid}}}

		_, err = u.Database.UserCol.UpdateOne(context.TODO(), filter, changes)
		if err != nil {
			u.Log.Debug(err.Error())
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`OK`))
	}
}

/*
Add Badge(POST)

  - Grab uuid from token

  - Add Badge to User Data

Returns:

	Http handler
*/
func (u *Service) AddTrophy() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		token, err := u.GrabToken(r)
		if err != nil {
			u.Log.Error(err.Error())
			http.Error(rw, "Unauthorized", http.StatusUnauthorized)
		}

		claims, err := u.DecodeToken(token)
		if err != nil {
			u.Log.Error("Failed to Decode Token: " + err.Error())
			http.Error(rw, "Forbidden", http.StatusForbidden)
			return
		}

		uuid := claims["sub"].(string)

		// decode body
		var req models.Badge
		json.NewDecoder(r.Body).Decode(&req)

		trophy := models.Trophy{
			ID:          primitive.NewObjectID(),
			Name:        req.Name,
			Title:       req.Title,
			ImageURL:    req.ImageURL,
			EventId:     req.EventId,
			Description: req.Description,
			AchievedAt:  req.AchievedAt,
		}

		filter := bson.M{"uuid": uuid}
		changes := bson.M{"$push": bson.M{"trophies": trophy}}

		_, err = u.Database.UserCol.UpdateOne(context.TODO(), filter, changes)
		if err != nil {
			u.Log.Debug(err.Error())
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(trophy)
	}
}

/*
Remove Trophy(DELETE)

  - Grab uuid from token

  - Grab id from path

  - Remove trophy from user data

Returns:

	Http handler
*/
func (u *Service) RemoveTrophy() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		token, err := u.GrabToken(r)
		if err != nil {
			u.Log.Error(err.Error())
			http.Error(rw, "Unauthorized", http.StatusUnauthorized)
		}

		claims, err := u.DecodeToken(token)
		if err != nil {
			u.Log.Error("Failed to Decode Token: " + err.Error())
			http.Error(rw, "Forbidden", http.StatusForbidden)
			return
		}

		uuid := claims["sub"].(string)
		// grab club id from path
		vars := mux.Vars(r)

		// if there is no club id
		if len(vars["id"]) == 0 || len(vars["id"]) < 24 {
			http.Error(rw, "Bad Trophy ID", http.StatusBadRequest)
			return
		}

		// convert application id to oid
		id := vars["id"]
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			u.Log.Debug(err.Error())
		}

		filter := bson.M{"uuid": uuid}
		changes := bson.M{"$pull": bson.M{"trophies": bson.M{"_id": oid}}}

		_, err = u.Database.UserCol.UpdateOne(context.TODO(), filter, changes)
		if err != nil {
			u.Log.Debug(err.Error())
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`OK`))
	}
}
