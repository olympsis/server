package service

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"olympsis-server/database"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

/*
Create new Club service struct

  - Create and Returns a pointer to a new club service struct
*/
func NewClubService(l *logrus.Logger, r *mux.Router, d *database.Database) *Service {
	return &Service{Logger: l, Router: r, Database: d}
}

/*
Get Clubs (GET)

  - Fetches and returns a list of clubs

  - Grab query params

  - Filter and Search Clubs

    Returns:
    Http handler

  - Writes list of club objects back to client
*/
func (c *Service) GetClubs() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		city := r.URL.Query().Get("city")
		state := r.URL.Query().Get("state")
		country := r.URL.Query().Get("country")

		if country == "" {
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "you need at least a country to query with." }`))
			return
		}

		filter := bson.M{}
		if state == "" {
			filter["country"] = country
		} else if city != "" {
			filter["country"] = country
			filter["state"] = state
			filter["city"] = city
		} else {
			filter["country"] = country
			filter["state"] = state
		}

		var clubs []Club
		cur, err := c.Database.ClubCol.Find(context.TODO(), filter)

		if err != nil {
			if err == mongo.ErrNoDocuments {
				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(http.StatusNoContent)
				return
			}
		}

		for cur.Next(context.TODO()) {
			var club Club
			err := cur.Decode(&club)
			if err != nil {
				c.Logger.Error(err)
			}
			// fetch user data
			for i := 0; i < len(club.Members); i++ {
				usr := c.FetchUser(*r, club.Members[i].UUID)
				club.Members[i].Data = &usr
			}
			clubs = append(clubs, club)
		}

		if len(clubs) == 0 {
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusNoContent)
			return
		}

		resp := ClubsResponse{
			TotalClubs: len(clubs),
			Clubs:      clubs,
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(resp)
	}
}

/*
Get a Club (GET)

  - Fetches and returns a club object

  - Grab path values

    Returns:
    Http handler

  - Writes a club object back to client
*/
func (c *Service) GetClub() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		if len(vars["id"]) < 24 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "No club id found in request." }`))
			return
		}
		id := vars["id"]
		OID, _ := primitive.ObjectIDFromHex(id)
		_, ctx := context.WithTimeout(context.Background(), 30*time.Second)
		defer ctx()

		// find club data in database
		var club Club
		filter := bson.D{primitive.E{Key: "_id", Value: OID}}
		err := c.Database.ClubCol.FindOne(context.TODO(), filter).Decode(&club)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				rw.WriteHeader(http.StatusNotFound)
				rw.Write([]byte(`{ "msg": "Club does not exist" }`))
				return
			}
		}

		// fetch user data
		for i := 0; i < len(club.Members); i++ {
			usr := c.FetchUser(*r, club.Members[i].UUID)
			club.Members[i].Data = &usr
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(club)

	}
}

/*
Create Club Data (POST)

  - Creates new club for olympsis

  - Grab request body

  - Create club data in user databse

    Returns:
    Http handler

  - Writes object back to client
*/
func (c *Service) CreateClub() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		var req Club

		_, ctx := context.WithTimeout(context.Background(), 30*time.Second)
		defer ctx()

		// decode request
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
			return
		}

		bearerToken := r.Header.Get("Authorization")
		tokenSplit := strings.Split(bearerToken, "Bearer ")
		token := tokenSplit[1]
		uuid, _, _, err := c.ValidateAndParseJWTToken(token)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(`{ "msg": "unable to parse token." }`))
			return
		}

		timeStamp := time.Now().Unix()
		member := Member{
			ID:       primitive.NewObjectID(),
			UUID:     uuid,
			Role:     "admin",
			JoinedAt: timeStamp,
		}

		club := Club{
			ID:          primitive.NewObjectID(),
			Name:        req.Name,
			Description: req.Description,
			IsPrivate:   req.IsPrivate,
			Sport:       req.Sport,
			City:        req.City,
			State:       req.State,
			Country:     req.Country,
			ImageURL:    req.ImageURL,
			Members:     []Member{member},
			Rules:       req.Rules,
			CreatedAt:   timeStamp,
		}

		// create club in database
		_, err = c.Database.ClubCol.InsertOne(context.TODO(), club)
		if err != nil {
			c.Logger.Error(err)
		}

		// update user id to contain the new club
		filter := bson.M{"uuid": uuid}
		update := bson.M{"$push": bson.M{"clubs": club.ID}}
		_, err = c.Database.UserCol.UpdateOne(context.TODO(), filter, update)
		if err != nil {
			c.Logger.Error(err)
		}

		// subscribe to club notifications topic
		usr := c.FetchUser(*r, uuid)
		ok, err := c.SubscribeToClubTopic(*r, club.ID.Hex(), []string{usr.DeviceToken})
		if !ok || err != nil {
			c.Logger.Error("Failed to subscribe user: " + uuid + "to club topic. Club: " + club.ID.Hex())
		}

		// subscribe to club admin notifications topic
		ok, err = c.SubscribeToClubTopic(*r, club.ID.Hex()+"-admin", []string{usr.DeviceToken})
		if !ok || err != nil {
			c.Logger.Error("Failed to subscribe user: " + uuid + "to club topic. Club: " + club.ID.Hex())
		}

		rw.WriteHeader(http.StatusCreated)
		json.NewEncoder(rw).Encode(club)
	}
}

/*
Update Club Data (POST)

  - Grab Club Id from path

  - Update club data

  - Grab request body

  - updated club data in databse

  - Must be club Admin

    Returns:
    Http handler

  - Writes object back to client
*/
func (c *Service) UpdateClub() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		bearerToken := r.Header.Get("Authorization")
		tokenSplit := strings.Split(bearerToken, "Bearer ")
		token := tokenSplit[1]
		uuid, _, _, err := c.ValidateAndParseJWTToken(token)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(`{ "msg": "Uuable to parse token." }`))
			return
		}
		// grab club id from path
		vars := mux.Vars(r)
		if len(vars["id"]) == 0 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "no club Id found in request." }`))
			return
		}

		if len(vars["id"]) < 24 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "bad club Id found in request." }`))
			return
		}

		id := vars["id"]
		if err != nil {
			c.Logger.Debug(err.Error())
		}

		isFound, err := c.CheckIfUserIsAdmin(uuid, id)
		if err != nil {
			c.Logger.Error(err)
			rw.WriteHeader(http.StatusUnauthorized)
			rw.Write([]byte(`{ "msg": "unable to verify if user is club admin." }`))
			return
		}
		if !isFound {
			rw.WriteHeader(http.StatusForbidden)
			return
		}
		var req Club

		// decode request
		err = json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
			return
		}

		oid, _ := primitive.ObjectIDFromHex(id)
		filter := bson.M{"_id": oid}
		change := bson.M{}
		update := bson.M{"$set": change}

		if req.Name != "" {
			change["name"] = req.Name
		}
		if req.Description != "" {
			change["description"] = req.Description
		}
		if req.Sport != "" {
			change["sport"] = req.Sport
		}
		if req.City != "" {
			change["city"] = req.City
		}
		if req.State != "" {
			change["state"] = req.State
		}
		if req.Country != "" {
			change["country"] = req.Country
		}
		if req.ImageURL != "" {
			change["imageURL"] = req.ImageURL
		}
		if len(req.Rules) > 0 {
			change["rules"] = req.Rules
		}

		// update club user in database
		_, err = c.Database.ClubCol.UpdateOne(context.TODO(), filter, update)
		if err != nil {
			c.Logger.Error(err)
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
			return
		}

		var club Club
		err = c.Database.ClubCol.FindOne(context.Background(), filter).Decode(&club)
		if err != nil {
			c.Logger.Error(err)
			rw.WriteHeader(http.StatusNotFound)
			rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(club)
	}
}

/*
Delete Club Data (DELETE)

  - Deletes club data object

  - Grab parameters and update

Returns:

	Http handler
		- Writes OK back to client if successful
*/
func (c *Service) DeleteClub() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		bearerToken := r.Header.Get("Authorization")
		tokenSplit := strings.Split(bearerToken, "Bearer ")
		token := tokenSplit[1]
		uuid, _, _, err := c.ValidateAndParseJWTToken(token)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(`{ "msg": "unable to parse token." }`))
			return
		}

		// grab club id from path
		vars := mux.Vars(r)

		// if there is no club id
		if len(vars["id"]) == 0 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "no club Id found in request." }`))
			return
		}

		// if we get an invalid id
		if len(vars["id"]) < 24 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "bad club Id found in request." }`))
			return
		}

		// convert club id to oid
		id := vars["id"]
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			c.Logger.Debug(err.Error())
		}

		// check if club exists
		// it it doesnt exist return 404
		var _club Club
		err = c.Database.ClubCol.FindOne(context.Background(), bson.M{"_id": oid}).Decode(&_club)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				c.Logger.Error(err)
				rw.WriteHeader(http.StatusNotFound)
				rw.Write([]byte(`{ "msg": "club does not exist" }`))
				return
			}
		}

		// check if user is the club admin to make those changes.
		isFound, err := c.CheckIfUserIsAdmin(uuid, id)
		if err != nil {
			c.Logger.Error(err)
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(`{ "msg": "unable to verify if user is club admin." }`))
			return
		}
		if !isFound {
			rw.WriteHeader(http.StatusForbidden)
			return
		}

		filter := bson.M{"_id": oid}

		// we need to unregister users from topic before deleting room
		var club Club
		err = c.Database.ClubCol.FindOne(context.TODO(), filter).Decode(&club)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(http.StatusNotFound)
				return
			}
		}

		// fetch their device tokens
		var tokens []string
		for i := 0; i < len(club.Members); i++ {
			usr := c.FetchUser(*r, club.Members[i].UUID)
			tokens = append(tokens, usr.DeviceToken)

			// remove club id from user
			ufilter := bson.M{"uuid": club.Members[i].UUID}
			update := bson.M{"$pull": bson.M{"clubs": club.ID}}
			_, err = c.Database.UserCol.UpdateOne(context.Background(), ufilter, update)
			if err != nil {
				c.Logger.Error(err)
			}
		}

		// unsubscribe users and admins
		ok, err := c.UnsubscribeFromClubTopic(*r, club.ID.Hex(), tokens)
		if !ok || err != nil {
			c.Logger.Error("Failed to unsubscribe users from club topic. Club: " + club.ID.Hex())
		}
		ok, err = c.UnsubscribeFromClubTopic(*r, club.ID.Hex()+"-admin", tokens)
		if !ok || err != nil {
			c.Logger.Error("Failed to unsubscribe admins from club topic. Club: " + club.ID.Hex())
		}

		// delete club
		_, err = c.Database.ClubCol.DeleteOne(context.TODO(), filter)
		if err != nil {
			c.Logger.Debug(err.Error())
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`OK`))
	}
}

func (c *Service) ChangeMemberRank() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// grab club id from path
		vars := mux.Vars(r)

		// if there is no club id
		if len(vars["id"]) == 0 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "no club Id found in request." }`))
			return
		}

		// if we get an invalid id
		if len(vars["id"]) < 24 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "bad club Id found in request." }`))
			return
		}

		// if there is no member id
		if len(vars["memberId"]) == 0 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "no club Id found in request." }`))
			return
		}

		// if we get an member id
		if len(vars["memberId"]) < 24 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "bad club Id found in request." }`))
			return
		}

		// convert club id to oid
		id := vars["id"]
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			c.Logger.Debug(err.Error())
		}

		// convert club id to oid
		memID := vars["memberId"]
		memOID, err := primitive.ObjectIDFromHex(memID)
		if err != nil {
			c.Logger.Debug(err.Error())
		}

		// json request
		var req ChangeRoleRequest
		json.NewDecoder(r.Body).Decode(&req)

		// fetch club data to get member position in array
		var club Club
		err = c.Database.ClubCol.FindOne(context.Background(), bson.M{"_id": oid}).Decode(&club)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(http.StatusNotFound)
				return
			}
		}

		// fetch user index in club data
		index := 0
		for i := 0; i < len(club.Members); i++ {
			if club.Members[i].ID == memOID {
				index = i
			}
		}

		member := Member{
			ID:       memOID,
			UUID:     club.Members[index].UUID,
			Role:     req.Role,
			JoinedAt: club.Members[index].JoinedAt,
		}

		// remove member and add new member with new rank
		filter := bson.M{"_id": oid}
		changes := bson.M{"$pull": bson.M{"members": bson.M{"_id": memOID}}}
		_, err = c.Database.ClubCol.UpdateOne(context.Background(), filter, changes)
		if err != nil {
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(`{ "msg": "failed to update member" }`))
			return
		}

		// remove member and add new member with new rank
		changes = bson.M{"$push": bson.M{"members": member}}
		_, err = c.Database.ClubCol.UpdateOne(context.Background(), filter, changes)
		if err != nil {
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(`{ "msg": "failed to update member" }`))
			return
		}

		// grab user data for device token for notifications
		usr := c.FetchUser(*r, club.Members[index].UUID)
		text := ""
		if req.Role == "admin" {
			text = "You've been promoted to Admin"
			c.SubscribeToClubTopic(*r, club.ID.Hex()+"-admin", []string{usr.DeviceToken})
		} else if req.Role == "member" {
			text = "You've been demoted"
			c.UnsubscribeFromClubTopic(*r, club.ID.Hex()+"-admin", []string{usr.DeviceToken})
		}

		// send notification to user
		c.SendNotificationToDevice(*r, club.Name, text, usr.DeviceToken)

		// fetch updated club data
		err = c.Database.ClubCol.FindOne(context.Background(), bson.M{"_id": oid}).Decode(&club)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(http.StatusNotFound)
				rw.Write([]byte(`{ "msg": "club not found" }`))
				return
			}
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(club)
	}
}

func (c *Service) KickMember() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// grab club id from path
		vars := mux.Vars(r)

		// if there is no club id
		if len(vars["id"]) == 0 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "no club Id found in request." }`))
			return
		}

		// if we get an invalid id
		if len(vars["id"]) < 24 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "bad club Id found in request." }`))
			return
		}

		// if there is no club id
		if len(vars["memberId"]) == 0 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "no member Id found in request." }`))
			return
		}

		// if we get an invalid id
		if len(vars["memberId"]) < 24 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "bad member Id found in request." }`))
			return
		}

		// convert club id to object id
		id := vars["id"]
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			c.Logger.Debug(err.Error())
		}

		// convert member id to objectid
		mid := vars["memberId"]
		memOID, err := primitive.ObjectIDFromHex(mid)
		if err != nil {
			c.Logger.Debug(err.Error())
		}

		// fetch club
		filter := bson.M{"_id": oid}
		var club Club
		err = c.Database.ClubCol.FindOne(context.Background(), filter).Decode(&club)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				c.Logger.Error(err)
				rw.WriteHeader(http.StatusNotFound)
				rw.Write([]byte(`{ "msg": "club does not exist" }`))
				return
			}
		}

		// fetch user index in club data
		index := 0
		for i := 0; i < len(club.Members); i++ {
			if club.Members[i].ID == memOID {
				index = i
			}
		}

		// remove club from user data
		filter = bson.M{"uuid": club.Members[index].UUID}
		update := bson.M{"$pull": bson.M{"clubs": oid}}
		_, err = c.Database.UserCol.UpdateOne(context.Background(), filter, update)
		if err != nil {
			c.Logger.Error(err)
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(`{ "msg": "failed to update user" }`))
			return
		}

		// fetch user token
		usr := c.FetchUser(*r, club.Members[index].UUID)

		// remove member from club
		filter = bson.M{"_id": oid}
		update = bson.M{"$pull": bson.M{"members": bson.M{"_id": memOID}}}
		_, err = c.Database.ClubCol.UpdateOne(context.Background(), filter, update)
		if err != nil {
			c.Logger.Error(err)
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(`{ "msg": "failed to update club" }`))
			return
		}

		// notify user
		c.SendNotificationToDevice(*r, club.Name, "You've been kicked out of "+club.Name, usr.DeviceToken)

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
	}
}

func (c *Service) LeaveClub() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		bearerToken := r.Header.Get("Authorization")
		tokenSplit := strings.Split(bearerToken, "Bearer ")
		token := tokenSplit[1]
		uuid, _, _, err := c.ValidateAndParseJWTToken(token)
		if err != nil {
			c.Logger.Error(err)
			return
		}

		// grab club id from path
		vars := mux.Vars(r)

		// if there is no club id
		if len(vars["id"]) == 0 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "no club Id found in request." }`))
			return
		}

		// if we get an invalid id
		if len(vars["id"]) < 24 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "bad club Id found in request." }`))
			return
		}

		// convert club id to oid
		id := vars["id"]
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			c.Logger.Debug(err.Error())
		}

		filter := bson.M{"_id": oid}
		update := bson.M{"$pull": bson.M{"members": bson.M{"uuid": uuid}}}

		// remove member from club
		_, err = c.Database.ClubCol.UpdateOne(context.Background(), filter, update)
		if err != nil {
			c.Logger.Error(err)
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(`{ "msg": "failed to update club" }`))
			return
		}

		// remove club id from user data
		filter = bson.M{"uuid": uuid}
		update = bson.M{"$pull": bson.M{"clubs": oid}}
		_, err = c.Database.UserCol.UpdateOne(context.TODO(), filter, update)
		if err != nil {
			c.Logger.Error(err)
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`OK`))
	}
}

/* APPLICATIONS */

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
		bearerToken := r.Header.Get("Authorization")
		tokenSplit := strings.Split(bearerToken, "Bearer ")
		token := tokenSplit[1]
		uuid, _, _, err := c.ValidateAndParseJWTToken(token)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(`{ "msg": "unable to parse token." }`))
			return
		}

		// grab club id from path
		vars := mux.Vars(r)

		// if there is no club id
		if len(vars["id"]) == 0 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "no club Id found in request." }`))
			return
		}

		// if we get an invalid id
		if len(vars["id"]) < 24 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "bad club Id found in request." }`))
			return
		}

		// convert club id to oid
		id := vars["id"]
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			c.Logger.Debug(err.Error())
		}

		// check if user is the club admin to make those changes.
		isFound, err := c.CheckIfUserIsAdmin(uuid, id)
		if err != nil {
			c.Logger.Error(err)
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(`{ "msg": "unable to verify if user is club admin." }`))
			return
		}
		if !isFound {
			rw.WriteHeader(http.StatusForbidden)
			return
		}
		_, ctx := context.WithTimeout(context.Background(), 30*time.Second)
		defer ctx()

		filter := bson.M{"clubId": oid}
		var apps []ClubApplication
		cur, err := c.Database.ClubApplicationCol.Find(context.TODO(), filter)

		if err != nil {
			if err == mongo.ErrNoDocuments {
				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(http.StatusNoContent)
				return
			}
		}

		for cur.Next(context.TODO()) {
			var app ClubApplication
			err := cur.Decode(&app)
			if err != nil {
				c.Logger.Error(err)
			}
			u := c.FetchUser(*r, app.UUID)
			app.Data = &u
			apps = append(apps, app)
		}

		// just in case mongo doesnt throw an error
		if len(apps) == 0 {
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusNoContent)
			return
		}

		resp := ClubApplicationsResponse{
			TotalApplications: len(apps),
			Applications:      apps,
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
		bearerToken := r.Header.Get("Authorization")
		tokenSplit := strings.Split(bearerToken, "Bearer ")
		token := tokenSplit[1]
		uuid, _, _, err := c.ValidateAndParseJWTToken(token)
		if err != nil {
			c.Logger.Error(err)
			return
		}

		// grab club id from path
		vars := mux.Vars(r)

		// if there is no club id
		if len(vars["id"]) == 0 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "no club Id found in request." }`))
			return
		}

		// if we get an invalid id
		if len(vars["id"]) < 24 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "bad club Id found in request." }`))
			return
		}

		// convert club id to oid
		id := vars["id"]
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			c.Logger.Debug(err.Error())
		}

		// check if an application already exists
		var _app ClubApplication
		filter := bson.M{"uuid": uuid, "clubId": oid}
		err = c.Database.ClubApplicationCol.FindOne(context.Background(), filter).Decode(&_app)
		if err != nil {
			if err != mongo.ErrNoDocuments {
				// if found return the application
				rw.WriteHeader(http.StatusCreated)
				json.NewEncoder(rw).Encode(_app)
				return
			}
		}

		timeStamp := time.Now().Unix()
		app := ClubApplication{
			ID:        primitive.NewObjectID(),
			UUID:      uuid,
			ClubId:    oid,
			Status:    "pending",
			CreatedAt: timeStamp,
		}

		// create club application in database
		_, err = c.Database.ClubApplicationCol.InsertOne(context.Background(), app)
		if err != nil {
			c.Logger.Error(err)
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
			return
		}

		c.SendNotificationToTopic(*r, "Club Application", "You have a new club application", oid.Hex()+"-admin")

		rw.WriteHeader(http.StatusCreated)
		json.NewEncoder(rw).Encode(app)
	}
}

/*
Update a Club Applications(PUT)

  - Updates a club applications

  - Grabs club id from path

  - Grabs application id from path

  - Update the satus of the specific application

  - Must be club Admin

Returns:

	Http handler
		- Writes ok back to user
*/
func (c *Service) UpdateApplication() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		bearerToken := r.Header.Get("Authorization")
		tokenSplit := strings.Split(bearerToken, "Bearer ")
		token := tokenSplit[1]
		uuid, _, _, err := c.ValidateAndParseJWTToken(token)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(`{ "msg": "unable to parse token." }`))
			return
		}

		// grab club id from path
		vars := mux.Vars(r)

		// if there is no club id
		if len(vars["id"]) == 0 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "no club Id found in request." }`))
			return
		}
		// if there is no application id
		if len(vars["applicationId"]) == 0 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "no application Id found in request." }`))
			return
		}

		// if we get an invalid club id
		if len(vars["id"]) < 24 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "bad club Id found in request." }`))
			return
		}
		// if we get an invalid application id
		if len(vars["applicationId"]) < 24 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "bad application Id found in request." }`))
			return
		}

		var req ApplicationUpdateRequest
		// decode request
		err = json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
			return
		}

		// convert club id to oid
		id := vars["id"]
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			c.Logger.Debug(err.Error())
		}

		// convert application id to oid
		aid := vars["applicationId"]
		aoid, err := primitive.ObjectIDFromHex(aid)
		if err != nil {
			c.Logger.Debug(err.Error())
		}

		// check if user is the club admin to make those changes.
		isFound, err := c.CheckIfUserIsAdmin(uuid, id)
		if err != nil {
			c.Logger.Error(err)
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(`{ "msg": "unable to verify if user is club admin." }`))
			return
		}
		if !isFound {
			rw.WriteHeader(http.StatusForbidden)
			return
		}

		// if the admin accepts the application
		if req.Status == "accepted" {

			// check if application exists
			var app ClubApplication
			filter := bson.M{"_id": aoid}
			err = c.Database.ClubApplicationCol.FindOne(context.TODO(), filter).Decode(&app)
			if err != nil {
				c.Logger.Error(err)
				rw.WriteHeader(http.StatusInternalServerError)
				rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
				return
			}

			// if someone else already accepted it we dont want to cause issues in user data where there are duplicated club id's
			if app.Status == "pending" {

				// update club application in database
				filter := bson.M{"_id": aoid}
				change := bson.M{"$set": bson.M{"status": req.Status}}
				_, err = c.Database.ClubApplicationCol.UpdateOne(context.TODO(), filter, change)
				if err != nil {
					c.Logger.Error(err)
					rw.WriteHeader(http.StatusInternalServerError)
					rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
					return
				}

				// add club id to user data
				filter = bson.M{"uuid": app.UUID}
				change = bson.M{"$push": bson.M{"clubs": oid}}
				_, err = c.Database.UserCol.UpdateOne(context.TODO(), filter, change)
				if err != nil {
					c.Logger.Error(err)
					rw.WriteHeader(http.StatusInternalServerError)
					rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
					return
				}

				member := Member{ // member object to put in club
					ID:       primitive.NewObjectID(), // unique member identifier
					UUID:     app.UUID,                // user uuid
					Role:     "member",                // user role
					JoinedAt: time.Now().Unix(),       // joined date
				}

				// update club information by adding member in the list
				filter = bson.M{"_id": oid}
				change = bson.M{"$push": bson.M{"members": member}}
				_, err = c.Database.ClubCol.UpdateOne(context.TODO(), filter, change)
				if err != nil {
					c.Logger.Error(err)
					rw.WriteHeader(http.StatusInternalServerError)
					rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
					return
				}

				// find club info
				var club Club
				err = c.Database.ClubCol.FindOne(context.Background(), filter).Decode(&club)
				if err != nil {
					c.Logger.Error(err)
					rw.WriteHeader(http.StatusInternalServerError)
					rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
					return
				}

				// find user device token
				usr := c.FetchUser(*r, app.UUID)

				// subscribe to club topic
				c.SubscribeToClubTopic(*r, id, []string{usr.DeviceToken})

				// notify user they were accepted to the club
				c.SendNotificationToDevice(*r, "Club Application", club.Name+" accepted your application.", usr.DeviceToken)

				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(http.StatusOK)
				return
			}

			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusOK)
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
		bearerToken := r.Header.Get("Authorization")
		tokenSplit := strings.Split(bearerToken, "Bearer ")
		token := tokenSplit[1]
		uuid, _, _, err := c.ValidateAndParseJWTToken(token)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(`{ "msg": "unable to parse token." }`))
			return
		}

		// grab application id from path
		vars := mux.Vars(r)

		// if there is no application id
		if len(vars["id"]) == 0 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "no club Id found in request." }`))
			return
		}

		// if we get an invalid application id
		if len(vars["id"]) < 24 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "bad club Id found in request." }`))
			return
		}

		// if there is no application id
		if len(vars["applicationId"]) == 0 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "no application Id found in request." }`))
			return
		}

		// if we get an invalid application id
		if len(vars["applicationId"]) < 24 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "bad application Id found in request." }`))
			return
		}

		// convert club id to oid
		appID := vars["applicationId"]
		appOID, err := primitive.ObjectIDFromHex(appID)
		if err != nil {
			c.Logger.Debug(err.Error())
		}

		clubID := vars["id"]
		clubOID, err := primitive.ObjectIDFromHex(clubID)
		if err != nil {
			c.Logger.Debug(err.Error())
		}

		filter := bson.M{"_id": appOID, "uuid": uuid, "clubId": clubOID}

		// update club application in database
		_, err = c.Database.ClubApplicationCol.DeleteOne(context.TODO(), filter)
		if err != nil {
			c.Logger.WithFields(logrus.Fields{
				"handler": "Delete Club Application Failed",
			}).Error("Failed to delete club application in the database")
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`OK`))
	}
}

// TODO: will work on this later
func (c *Service) CreateInvitation() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		bearerToken := r.Header.Get("Authorization")
		tokenSplit := strings.Split(bearerToken, "Bearer ")
		token := tokenSplit[1]
		_, _, _, err := c.ValidateAndParseJWTToken(token)
		if err != nil {
			c.Logger.Error(err)
			return
		}

		var req ClubInvitation

		_, ctx := context.WithTimeout(context.Background(), 30*time.Second)
		defer ctx()

		// decode request
		err = json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
			return
		}

		timeStamp := time.Now().Unix()
		inv := ClubInvitation{
			ID:        primitive.NewObjectID(),
			UUID:      req.UUID,
			ClubId:    req.ClubId,
			Status:    "pending",
			CreatedAt: timeStamp,
		}

		// create club invitation in database
		_, err = c.Database.ClubInvCol.InsertOne(context.TODO(), inv)
		if err != nil {
			c.Logger.Error(err)
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
			return
		}

		//c.sendNotification(*r, "You have a new club invite", "You've been invited to join a club", "", []string{})

		rw.WriteHeader(http.StatusCreated)
		json.NewEncoder(rw).Encode(inv)
	}
}

func (c *Service) SendNotificationToTopic(r http.Request, t string, b string, tpc string) (bool, error) {
	bearerToken := r.Header.Get("Authorization")
	tokenSplit := strings.Split(bearerToken, "Bearer ")
	token := tokenSplit[1]
	client := &http.Client{}

	request := NotificationRequest{
		Title: t,
		Body:  b,
		Topic: tpc,
	}

	data, err := json.Marshal(request)
	if err != nil {
		c.Logger.Error(err.Error())
		return false, err
	}

	req, err := http.NewRequest("POST", "http://pushnote.olympsis.internal/v1/pushnote/topic", bytes.NewBuffer(data))
	if err != nil {
		c.Logger.Error(err.Error())
		return false, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		c.Logger.Error(err.Error())
		return false, err
	}

	defer resp.Body.Close()
	return true, nil
}

func (c *Service) SendNotificationToDevice(r http.Request, t string, b string, tk string) (bool, error) {
	bearerToken := r.Header.Get("Authorization")
	tokenSplit := strings.Split(bearerToken, "Bearer ")
	token := tokenSplit[1]
	client := &http.Client{}

	request := NotificationRequest{
		Title:  t,
		Body:   b,
		Tokens: []string{tk},
	}

	data, err := json.Marshal(request)
	if err != nil {
		c.Logger.Error(err.Error())
		return false, err
	}

	req, err := http.NewRequest("POST", "http://pushnote.olympsis.internal/v1/pushnote/device", bytes.NewBuffer(data))
	if err != nil {
		c.Logger.Error(err.Error())
		return false, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		c.Logger.Error(err.Error())
		return false, err
	}

	defer resp.Body.Close()
	return true, nil
}

func (c *Service) SubscribeToClubTopic(r http.Request, tpc string, tks []string) (bool, error) {
	bearerToken := r.Header.Get("Authorization")
	tokenSplit := strings.Split(bearerToken, "Bearer ")
	token := tokenSplit[1]
	client := &http.Client{}

	request := NotificationRequest{
		Topic:  tpc,
		Tokens: tks,
	}

	data, err := json.Marshal(request)
	if err != nil {
		c.Logger.Error(err.Error())
		return false, err
	}

	req, err := http.NewRequest("PUT", "http://pushnote.olympsis.internal/v1/pushnote/topic", bytes.NewBuffer(data))
	if err != nil {
		c.Logger.Error(err.Error())
		return false, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		c.Logger.Error(err.Error())
		return false, err
	}

	defer resp.Body.Close()
	return true, nil
}

func (c *Service) UnsubscribeFromClubTopic(r http.Request, tpc string, tks []string) (bool, error) {
	bearerToken := r.Header.Get("Authorization")
	tokenSplit := strings.Split(bearerToken, "Bearer ")
	token := tokenSplit[1]
	client := &http.Client{}

	request := NotificationRequest{
		Topic:  tpc,
		Tokens: tks,
	}

	data, err := json.Marshal(request)
	if err != nil {
		c.Logger.Error(err.Error())
		return false, err
	}

	req, err := http.NewRequest("DELETE", "http://pushnote.olympsis.internal/v1/pushnote/topic", bytes.NewBuffer(data))
	if err != nil {
		c.Logger.Error(err.Error())
		return false, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		c.Logger.Error(err.Error())
		return false, err
	}

	defer resp.Body.Close()
	return true, nil
}

/*
Validate an Parse JWT Token

  - parse jwt token

  - return values

Returns:

	uuid - string of the user id token
	createdAt - string of the session token created date
	role - role of user
	error -  if there is an error return error else nil
*/
func (c *Service) ValidateAndParseJWTToken(tokenString string) (string, string, float64, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("KEY")), nil
	})

	if err != nil {
		return "", "", 0, err
	} else {
		uuid := claims["uuid"].(string)
		provider := claims["provider"].(string)
		createdAt := claims["createdAt"].(float64)
		return uuid, provider, createdAt, nil
	}
}

func (c *Service) FetchUser(r http.Request, user string) LookUpUser {
	bearerToken := r.Header.Get("Authorization")
	tokenSplit := strings.Split(bearerToken, "Bearer ")
	token := tokenSplit[1]
	client := &http.Client{}

	req, err := http.NewRequest("GET", "http://lookup.olympsis.internal/v1/lookup/"+user, nil)
	if err != nil {
		c.Logger.Error(err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		c.Logger.Error(err)
	}

	defer resp.Body.Close()

	var lookup LookUpUser
	err = json.NewDecoder(resp.Body).Decode(&lookup)
	if err != nil {
		c.Logger.Fatal(err)
	}
	return lookup
}

/*
Check If User is Admin

  - fetches club data

  - check in members list if uuid exists

Returns:

	bool - if admin or no
	error - error
*/
func (c *Service) CheckIfUserIsAdmin(userId string, clubId string) (bool, error) {
	var club Club
	OID, _ := primitive.ObjectIDFromHex(clubId)
	filter := bson.D{primitive.E{Key: "_id", Value: OID}}
	err := c.Database.ClubCol.FindOne(context.TODO(), filter).Decode(&club)
	if err != nil {
		return false, err
	}
	found := false

	for i := 0; i < len(club.Members); i++ {
		if club.Members[i].UUID == userId {
			found = true
		}
	}
	c.Logger.Debug(club.Members)
	return found, nil
}
