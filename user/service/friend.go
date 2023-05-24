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

/*
Get User Data (GET)

  - Grab uuid from params

  - Grabs user data from database

Returns:

	Http handler
		- Writes user data back to client
*/
func (u *Service) GetFriends() http.HandlerFunc {
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

		// find user data in database
		var user models.User
		filter := bson.D{primitive.E{Key: "uuid", Value: uuid}}
		err = u.FindUser(context.Background(), filter, &user)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				http.Error(rw, "Not Found", http.StatusNotFound)
				return
			}
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(user)
	}
}

/*
Get Friend Requests (GET)

  - Grab uuid from token

  - Fetch requests

Returns:

	Http handler
		- Writes an array of Friend Requests
*/
func (u *Service) GetFriendRequests() http.HandlerFunc {
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
		filter := bson.M{"requestee": uuid, "status": "pending"}
		var reqs []models.FriendRequest
		cur, err := u.Database.FriendReqCol.Find(context.TODO(), filter)

		if err != nil {
			if err == mongo.ErrNoDocuments {
				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(http.StatusNoContent)
				return
			}
		}

		// fetch requests
		for cur.Next(context.TODO()) {
			var fReq models.FriendRequest
			err := cur.Decode(&fReq)
			if err != nil {
				u.Log.Error(err)
			}
			// lookup := u.FetchUser(*r, fReq.Requestor)
			// fReq.RequestorData = lookup
			reqs = append(reqs, fReq)
		}

		// fetch requestor info

		if len(reqs) == 0 {
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusNoContent)
			return
		}

		resp := models.FriendRequests{
			TotalRequests: len(reqs),
			Requests:      reqs,
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(resp)
	}
}

/*
Create Friend Request(POST)

  - Grab uuid from token

  - Decode body

  - Create friend request in db

Returns:

	Http handler
*/
func (u *Service) CreateFriendRequest() http.HandlerFunc {
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
		var req models.FriendRequest
		json.NewDecoder(r.Body).Decode(&req)

		fReq := models.FriendRequest{
			ID:        primitive.NewObjectID(),
			Requestor: uuid,
			Requestee: req.Requestee,
			Status:    "pending",
			CreatedAt: time.Now().Unix(),
		}

		_, err = u.Database.FriendReqCol.InsertOne(context.TODO(), fReq)
		if err != nil {
			u.Log.Error(err)
			return
		}

		// fetch requestee data and device token to notify them
		var requestee models.User
		filter := bson.M{"uuid": req.Requestee}
		err = u.Database.UserCol.FindOne(context.Background(), filter).Decode(&requestee)
		if err != nil {
			u.Log.Error(err)

			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusCreated)
			json.NewEncoder(rw).Encode(fReq)

			return
		}

		// send notification
		// u.sendNotification(*r, "Friend Request", "You have a new friend request.", []string{requestee.DeviceToken})

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusCreated)
		json.NewEncoder(rw).Encode(fReq)
	}
}

/*
Update Friend Request(POST)

  - Grab uuid from token

  - grab friend request id from path

  - update friend request in db

Returns:

	Http handler
*/
func (u *Service) UpdateFriendRequest() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// grab request id from path
		vars := mux.Vars(r)

		// if there is no request id id
		if len(vars["id"]) == 0 || len(vars["id"]) < 24 {
			http.Error(rw, "Bad Request ID", http.StatusBadRequest)
			return
		}

		// convert request id to oid
		id := vars["id"]
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			u.Log.Error(err.Error())
		}

		// we're decoding a request here but the only thing we need is `status` so we can change it from pending to accepted or declined
		var req models.FriendRequest
		json.NewDecoder(r.Body).Decode(&req)

		// set the status to the value found in the request
		filter := bson.M{"_id": oid}
		changes := bson.M{"$set": bson.M{"status": req.Status}}
		_, err = u.Database.FriendReqCol.UpdateOne(context.TODO(), filter, changes)
		if err != nil {
			u.Log.Error(err.Error())
		}

		if req.Status == "accepted" {
			// now we need to update both the requestor and requestee data to reflect that they are now friends.
			// first we fetch the friend request and we use the requestor and requestee id's on there
			var request models.FriendRequest
			err = u.Database.FriendReqCol.FindOne(context.Background(), filter).Decode(&request)
			u.Log.Info(request.Requestor)
			if err != nil {
				u.Log.Error("Failed to find friend request " + err.Error())
				http.Error(rw, "Request Not Found", http.StatusNotFound)
			}

			timestamp := time.Now().Unix()

			// friend object for requestee or the person calling this function to accept the request
			filter = bson.M{"uuid": request.Requestee}
			friend := models.Friend{
				ID:        primitive.NewObjectID(),
				UUID:      request.Requestor,
				CreatedAt: timestamp,
			}

			// friend object for requestor or the person that sent the friend request
			rFilter := bson.M{"uuid": request.Requestor}
			rFriend := models.Friend{
				ID:        primitive.NewObjectID(),
				UUID:      request.Requestee,
				CreatedAt: timestamp,
			}

			// update requestee data
			update := bson.M{"$push": bson.M{"friends": friend}}
			_, err = u.Database.UserCol.UpdateOne(context.Background(), filter, update)
			if err != nil {
				u.Log.Error("Failed to update Requestee's friends: " + err.Error())
			}

			// update requestor data
			rUpdate := bson.M{"$push": bson.M{"friends": rFriend}}
			_, err = u.Database.UserCol.UpdateOne(context.Background(), rFilter, rUpdate)
			if err != nil {
				u.Log.Error("Failed to update Requestor's friends: " + err.Error())
			}

			// lookup := u.FetchUser(*r, request.Requestor)
			// friend.Data = lookup

			// return friend object
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusOK)
			json.NewEncoder(rw).Encode(friend)
		} else {
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusOK)
		}
	}
}

/*
Delete Friend Request(DELETE)

  - Grab uuid from token

  - Delete friend request in db

Returns:

	Http handler
*/
func (u *Service) DeleteFriendRequest() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		token, err := u.GrabToken(r)
		if err != nil {
			u.Log.Error(err.Error())
			http.Error(rw, "Unauthorized", http.StatusUnauthorized)
		}

		claims, err := u.DecodeToken(token)
		if err != nil {
			u.Log.Error(err.Error())
			http.Error(rw, "Forbidden", http.StatusForbidden)
			return
		}

		uuid := claims["sub"].(string)

		// grab application id from path
		vars := mux.Vars(r)

		// if there is no application id
		if len(vars["id"]) == 0 || len(vars["id"]) < 24 {
			http.Error(rw, "Bad Request ID", http.StatusBadRequest)
			return
		}

		// convert request id string to oid
		id := vars["id"]
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			u.Log.Debug(err.Error())
		}

		filter := bson.M{"_id": oid, "requestor": uuid}
		_, err = u.Database.FriendReqCol.DeleteOne(context.TODO(), filter)
		if err != nil {
			u.Log.Debug(err.Error())
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`OK`))
	}
}

/*
Remove Friend(DELETE)

  - Grab uuid from token

  - Grab id from path

  - Remove friend from user data

Returns:

	Http handler
*/
func (u *Service) RemoveFriend() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		token, err := u.GrabToken(r)
		if err != nil {
			u.Log.Error(err.Error())
			http.Error(rw, "Unauthorized", http.StatusUnauthorized)
		}

		claims, err := u.DecodeToken(token)
		if err != nil {
			u.Log.Error(err.Error())
			http.Error(rw, "Forbidden", http.StatusForbidden)
			return
		}

		uuid := claims["sub"].(string)

		// grab club id from path
		vars := mux.Vars(r)

		// if there is no club id
		if len(vars["id"]) == 0 || len(vars["id"]) < 24 {
			http.Error(rw, "Bad Friend ID", http.StatusBadRequest)
			return
		}

		// convert application id to oid
		id := vars["id"]
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			u.Log.Debug(err.Error())
		}

		filter := bson.M{"uuid": uuid}
		changes := bson.M{"$pull": bson.M{"friends": bson.M{"_id": oid}}}

		_, err = u.Database.UserCol.UpdateOne(context.TODO(), filter, changes)
		if err != nil {
			u.Log.Debug(err.Error())
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`OK`))
	}
}
