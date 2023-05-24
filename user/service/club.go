package service

import (
	"context"
	"encoding/json"
	"net/http"
	"olympsis-server/models"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func (u *Service) UpdateClubInvite() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// grab invite id from path
		vars := mux.Vars(r)

		// if there is no invite id
		if len(vars["id"]) == 0 || len(vars["id"]) < 24 {
			http.Error(rw, "Bad Club ID", http.StatusBadRequest)
			return
		}

		// convert club invite id to oid
		id := vars["id"]
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			u.Log.Error(err.Error())
		}

		// we're decoding a request here but the only thing we need is `status` so we can change it from pending to accepted or declined
		var req models.ClubInvite
		json.NewDecoder(r.Body).Decode(&req)

		// set the status to the value found in the request
		filter := bson.M{"_id": oid}
		changes := bson.M{"$set": bson.M{"status": req.Status}}
		_, err = u.Database.ClubInvCol.UpdateOne(context.TODO(), filter, changes)
		if err != nil {
			u.Log.Error(err.Error())
		}

		if req.Status == "accepted" {
			// update club members
			var inv models.ClubInvite
			err = u.Database.ClubInvCol.FindOne(context.Background(), filter).Decode(&inv)
			if err != nil {
				u.Log.Error(err.Error())
				http.Error(rw, "Not Found", http.StatusNotFound)
			}

			timestamp := time.Now().Unix()

			// update user data
			filter = bson.M{"uuid": inv.UUID}
			update := bson.M{"$push": bson.M{"clubs": inv.ClubId}}
			_, err = u.Database.ClubCol.UpdateOne(context.Background(), filter, update)
			if err != nil {
				u.Log.Error(err.Error())
			}

			member := models.Member{
				ID:       primitive.NewObjectID(),
				UUID:     inv.UUID,
				Role:     "member",
				JoinedAt: timestamp,
			}

			oid, err := primitive.ObjectIDFromHex(inv.ClubId)
			if err != nil {
				u.Log.Error(err.Error())
			}

			// update club members
			filter := bson.M{"_id": oid}
			update = bson.M{"$push": bson.M{"members": member}}
			_, err = u.Database.ClubCol.UpdateOne(context.Background(), filter, update)
			if err != nil {
				u.Log.Error(err.Error())
			}

			var club models.Club
			err = u.Database.ClubCol.FindOne(context.Background(), filter).Decode(&club)
			if err != nil {
				u.Log.Error(err.Error())
			}

			// return club object
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusOK)
			json.NewEncoder(rw).Encode(club)
		}
	}
}

func (u *Service) GetClubInvites() http.HandlerFunc {
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

		filter := bson.M{"uuid": uuid, "status": "pending"}
		var invs []models.ClubInvite
		cur, err := u.Database.ClubInvCol.Find(context.TODO(), filter)

		if err != nil {
			if err == mongo.ErrNoDocuments {
				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(http.StatusNoContent)
				return
			}
		}

		// fetch invites
		for cur.Next(context.TODO()) {
			var cInv models.ClubInvite
			err := cur.Decode(&cInv)
			if err != nil {
				u.Log.Error(err)
			}

			var club models.Club
			oid, err := primitive.ObjectIDFromHex(cInv.ClubId)
			if err != nil {
				u.Log.Error(err.Error())
			}
			filter := bson.M{"_id": oid}
			err = u.Database.ClubCol.FindOne(context.Background(), filter).Decode(&club)
			if err != nil {
				u.Log.Error(err)
			}
		}

		// fetch requestor info

		if len(invs) == 0 {
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusNoContent)
			return
		}

		resp := models.ClubInvites{
			TotalInvites: len(invs),
			Invites:      invs,
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(resp)
	}
}
