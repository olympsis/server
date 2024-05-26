package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"olympsis-server/aggregations"
	"olympsis-server/database"
	"olympsis-server/utils"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/olympsis/models"
	"github.com/olympsis/search"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Service struct {
	Database      *database.Database
	Logger        *logrus.Logger
	Router        *mux.Router
	SearchService *search.Service
}

/*
Create new Club service struct
*/
func NewClubService(l *logrus.Logger, r *mux.Router, d *database.Database, sh *search.Service) *Service {
	return &Service{Logger: l, Router: r, Database: d, SearchService: sh}
}

// Fetches all of the clubs in a given location
func (s *Service) GetClubsByLocation() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		city := r.URL.Query().Get("city")
		state := r.URL.Query().Get("state")
		country := r.URL.Query().Get("country")
		sports := r.URL.Query().Get("sports")
		if country == "" {
			http.Error(rw, `{ "msg": "you need at least a country to query with" }`, http.StatusBadRequest)
			return
		}

		filter := bson.M{}

		// Location Query
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

		// Sports Query
		if sports != "" {
			splicedSports := strings.Split(sports, ",")
			filter["sports"] = bson.M{
				"$in": splicedSports,
			}
		}

		// get all of the clubs data
		clubs, err := aggregations.AggregateClubs(bson.M{"$match": filter}, s.Database)
		if err != nil {
			s.Logger.Error("failed to find clubs: ", err.Error())
			http.Error(rw, `{ "msg": "failed to find clubs" }`, http.StatusInternalServerError)
		}

		// no content
		if len(*clubs) == 0 {
			http.Error(rw, `{ "msg": "no clubs found" }`, http.StatusNoContent)
			return
		}

		resp := models.ClubsResponse{
			TotalClubs: len(*clubs),
			Clubs:      *clubs,
		}
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(resp)
	}
}

// Get the data of a club
func (c *Service) GetClub() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		id := vars["id"]

		// check if id is valid
		isValidId := utils.ValidateClubID(id)
		if !isValidId {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{"msg": "bad club id found in request." }`))
			return
		}

		// convert string -> oid
		oid, _ := primitive.ObjectIDFromHex(id)
		_, ctx := context.WithTimeout(context.Background(), 30*time.Second)
		defer ctx()

		// find club data in database
		club, err := aggregations.AggregateClub(&oid, c.Database)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				http.Error(rw, `{ "msg": "club not found" }`, http.StatusNotFound)
				return
			} else {
				c.Logger.Error("failed to find club", err.Error())
				http.Error(rw, `{ "msg": "failed to find club" }`, http.StatusNotFound)
				return
			}
		}

		// if no error is returned and no club is returned
		if club.ID.IsZero() {
			http.Error(rw, `{ "msg": "club not found" }`, http.StatusNotFound)
			return
		}

		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(club)

	}
}

// Creates a new club
func (c *Service) CreateClub() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		uuid := r.Header.Get("UUID")

		// decode request
		var req models.ClubDao
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			c.Logger.Error("Failed to decode request ", err.Error())
			http.Error(rw, "Failed to decode request", http.StatusBadRequest)
			return
		}

		timeStamp := time.Now().Unix()
		member := models.MemberDao{
			ID:       primitive.NewObjectID(),
			UUID:     uuid,
			Role:     "owner",
			JoinedAt: timeStamp,
		}
		members := []models.MemberDao{member}

		club := models.ClubDao{
			Name:        req.Name,
			Description: req.Description,
			Sports:      req.Sports,
			City:        req.City,
			State:       req.State,
			Country:     req.Country,
			Logo:        req.Logo,
			Banner:      req.Banner,
			Visibility:  req.Visibility,
			Members:     &members,
			BlackList:   req.BlackList,
			Rules:       req.Rules,
			IsVerified:  req.IsVerified,
			CreatedAt:   &timeStamp,
		}

		// create club in database
		id, err := c.InsertClub(context.TODO(), &club)
		if err != nil {
			c.Logger.Error("Failed to create club: ", err.Error())
			http.Error(rw, "Failed to create club", http.StatusInternalServerError)
			return
		}

		// update user id to contain the new club
		filter := bson.M{"uuid": uuid}
		update := bson.M{"$push": bson.M{"clubs": id}}
		_, err = c.Database.UserCol.UpdateOne(context.TODO(), filter, update)
		if err != nil {
			c.Logger.Error(fmt.Sprintf("Failed to update user: %s", err.Error()))
		}

		// create notification topics
		clubTopic := id.Hex()
		clubAdminTopic := id.Hex() + "_admin"

		// create topics and subscribe owner to it
		err = utils.CreateNotificationTopic(clubTopic)
		if err != nil {
			c.Logger.Error("Failed to create club topic: ", err.Error())
		}

		err = utils.CreateNotificationTopic(clubAdminTopic)
		if err != nil {
			c.Logger.Error("Failed to create club admin topic: ", err.Error())
		}

		err = utils.AddTokenToTopic(clubTopic, uuid)
		if err != nil {
			c.Logger.Error("Failed to add token to club topic: ", err.Error())
		}

		err = utils.AddTokenToTopic(clubAdminTopic, uuid)
		if err != nil {
			c.Logger.Error("Failed to add token club admin topic: ", err.Error())
		}

		rw.WriteHeader(http.StatusCreated)
		json.NewEncoder(rw).Encode(models.CreateResponse{ID: id.Hex()})
	}
}

// Updates club data
func (c *Service) ModifyClub() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// Grab club id from path and validate it
		id := mux.Vars(r)["id"]
		valid := utils.ValidateClubID(id)
		if !valid {
			http.Error(rw, `{ "msg": "invalid club id" }`, http.StatusBadRequest)
			return
		}

		var req models.ClubDao

		// decode request
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			c.Logger.Error("Failed to decode body: ", err.Error())
			http.Error(rw, `{ "msg": "Failed to decode body" }`, http.StatusBadRequest)
			return
		}

		// Conversions and new variables
		oid, _ := primitive.ObjectIDFromHex(id)
		filter := bson.M{"_id": oid}
		change := bson.M{}
		update := bson.M{"$set": change}

		// Changes map
		if req.Name != nil {
			change["name"] = req.Name
		}
		if req.Description != nil {
			change["description"] = req.Description
		}
		if req.Sports != nil {
			change["sports"] = req.Sports
		}
		if req.City != nil {
			change["city"] = req.City
		}
		if req.State != nil {
			change["state"] = req.State
		}
		if req.Country != nil {
			change["country"] = req.Country
		}
		if req.Logo != nil {
			change["logo"] = req.Logo
		}
		if req.Banner != nil {
			change["banner"] = req.Banner
		}
		if req.Visibility != nil {
			change["visibility"] = req.Visibility
		}
		if req.BlackList != nil && len(*req.BlackList) > 0 {
			change["blacklist"] = req.BlackList
		}
		if req.Rules != nil && len(*req.Rules) > 0 {
			change["rules"] = req.Rules
		}

		// update club data in database
		err = c.UpdateClub(context.Background(), filter, update)
		if err != nil {
			c.Logger.Error("Failed to update club: ", err.Error())
			http.Error(rw, `{ "msg": "Failed to update club" }`, http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusOK)
	}
}

// Deletes a club
func (c *Service) DeleteClub() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// Grab club id from path and validate it
		id := mux.Vars(r)["id"]
		valid := utils.ValidateClubID(id)
		if !valid {
			http.Error(rw, `{ "msg": "invalid club id" }`, http.StatusBadRequest)
			return
		}

		// convert club id to oid
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			c.Logger.Error("Failed to convert club id: ", err.Error())
			http.Error(rw, `{ "msg": "bad club id" }`, http.StatusBadRequest)
			return
		}

		// check if club exists
		club, err := c.FindClub(context.TODO(), bson.M{"_id": oid})
		if err != nil {
			if err == mongo.ErrNoDocuments {
				c.Logger.Error("club not found: ", err.Error())
				http.Error(rw, `{ "msg": "club not found" }`, http.StatusNotFound)
				return
			}
		}

		// delete topics
		clubTopic := id
		clubAdminTopic := id + "_admin"

		err = utils.DeleteNotificationTopic(clubTopic)
		if err != nil {
			c.Logger.Error("failed to delete topic", err.Error())
		}

		err = utils.DeleteNotificationTopic(clubAdminTopic)
		if err != nil {
			c.Logger.Error("failed to delete topic", err.Error())
		}

		members := *club.Members

		// delete club from users data
		for i := range members {
			filter := bson.M{"uuid": members[i].UUID}
			update := bson.M{"$pull": bson.M{"clubs": oid}}
			c.Database.UserCol.UpdateOne(context.Background(), filter, update)
		}

		// delete club
		filter := bson.M{"_id": oid}
		err = c.RemoveClub(context.TODO(), filter)
		if err != nil {
			c.Logger.Debug("failed to delete club: ", err.Error())
			http.Error(rw, `{ "msg": "failed to delete club"}`, http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusOK)
	}
}

/*
Change Member Rank (PUT)

  - Updates a member's rank

  - Grab parameters and update

Returns:

	Http handler
		- Writes OK back to client if successful
*/
func (c *Service) ChangeMemberRank() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// grab club id from path
		vars := mux.Vars(r)

		// Grab club id from path and validate it
		id := mux.Vars(r)["id"]
		valid := utils.ValidateClubID(id)
		if !valid {
			http.Error(rw, "invalid club id", http.StatusBadRequest)
			return
		}

		// if there is no member id
		if len(vars["memberID"]) == 0 || len(vars["memberID"]) < 24 {
			http.Error(rw, "bad member id found", http.StatusBadRequest)
			return
		}

		// convert club id to oid
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			c.Logger.Debug(err.Error())
		}

		// convert club id to oid
		memID := vars["memberID"]
		memOID, err := primitive.ObjectIDFromHex(memID)
		if err != nil {
			c.Logger.Debug(err.Error())
		}

		// json request
		var req models.ChangeRoleRequest
		json.NewDecoder(r.Body).Decode(&req)

		// fetch club data to get member position in array
		var club models.ClubDao
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
		members := *club.Members
		for i := 0; i < len(members); i++ {
			if members[i].ID == memOID {
				index = i
			}
		}

		member := models.MemberDao{
			ID:       memOID,
			UUID:     members[index].UUID,
			Role:     req.Role,
			JoinedAt: members[index].JoinedAt,
		}

		// remove member and add new member with new rank
		filter := bson.M{"_id": oid}
		changes := bson.M{"$pull": bson.M{"members": bson.M{"_id": memOID}}}
		_, err = c.Database.ClubCol.UpdateOne(context.Background(), filter, changes)
		if err != nil {
			http.Error(rw, "failed to update member", http.StatusInternalServerError)
			return
		}

		// remove member and add new member with new rank
		changes = bson.M{"$push": bson.M{"members": member}}
		_, err = c.Database.ClubCol.UpdateOne(context.Background(), filter, changes)
		if err != nil {
			http.Error(rw, "failed to update member", http.StatusInternalServerError)
			return
		}

		// grab user data for device token for notifications
		usr, err := c.SearchService.SearchUserByUUID(member.UUID)
		if err != nil {
			c.Logger.Error("failed to get user data: " + err.Error())
		}

		text := ""
		if req.Role == "owner" {
			text = "You've been promoted to Owner"
		} else if req.Role == "admin" {
			text = "You've been promoted to Admin"
		} else if req.Role == "moderator" {
			text = "You've been promoted to Moderator"
		} else {
			text = "You've been demoted"
		}

		// if user was member then add them to the admin topic
		if members[index].Role == "member" {
			err = utils.AddTokenToTopic(id+"_admin", usr.UUID)
			if err != nil {
				c.Logger.Error("failed to add token to topic: ", err.Error())
			}
		}

		// notify user that they had their rank changed
		notification := models.Notification{
			Title: *club.Name,
			Body:  text,
		}

		utils.SendNotificationToToken(usr.DeviceToken, &notification)

		// fetch updated club data
		err = c.Database.ClubCol.FindOne(context.Background(), bson.M{"_id": oid}).Decode(&club)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				http.Error(rw, "club not found", http.StatusNotFound)
				return
			}
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(club)
	}
}

/*
Kick Member (PUT)

  - Removes member from club

  - Grab parameters and update

Returns:

	Http handler
		- Writes OK back to client if successful
*/
func (c *Service) KickMember() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// grab club id from path
		vars := mux.Vars(r)

		// Grab club id from path and validate it
		id := mux.Vars(r)["id"]
		valid := utils.ValidateClubID(id)
		if !valid {
			http.Error(rw, "invalid club id", http.StatusBadRequest)
			return
		}

		// if there is no club id
		if len(vars["memberID"]) == 0 {
			http.Error(rw, "bad member id found", http.StatusBadRequest)
			return
		}

		// if we get an invalid id
		if len(vars["memberID"]) < 24 {
			http.Error(rw, "bad member id found", http.StatusBadRequest)
			return
		}

		// convert club id to object id
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			c.Logger.Debug(err.Error())
		}

		// convert member id to objectid
		mid := vars["memberID"]
		memOID, err := primitive.ObjectIDFromHex(mid)
		if err != nil {
			c.Logger.Debug(err.Error())
		}

		// fetch club
		filter := bson.M{"_id": oid}
		var club models.ClubDao
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
		members := *club.Members
		for i := 0; i < len(members); i++ {
			if members[i].ID == memOID {
				index = i
			}
		}

		// remove club from user data
		filter = bson.M{"uuid": members[index].UUID}
		update := bson.M{"$pull": bson.M{"clubs": oid}}
		_, err = c.Database.UserCol.UpdateOne(context.Background(), filter, update)
		if err != nil {
			c.Logger.Error(err.Error())
			http.Error(rw, "failed to update user", http.StatusInternalServerError)
			return
		}

		// fetch user token
		usr, err := c.SearchService.SearchUserByUUID(members[index].UUID)
		if err != nil {
			c.Logger.Error(err.Error())
		}

		// remove member from club
		filter = bson.M{"_id": oid}
		update = bson.M{"$pull": bson.M{"members": bson.M{"_id": memOID}}}
		_, err = c.Database.ClubCol.UpdateOne(context.Background(), filter, update)
		if err != nil {
			c.Logger.Error(err.Error())
			http.Error(rw, "failed to update club", http.StatusInternalServerError)
			return
		}

		// notify user then remove them from the topics
		notification := models.Notification{
			Title: *club.Name,
			Body:  fmt.Sprintf(`You've been kicked out of %s`, *club.Name),
		}

		utils.SendNotificationToToken(usr.DeviceToken, &notification)
		err = utils.RemoveTokenFromTopic(id, usr.UUID)
		if err != nil {
			c.Logger.Error("failed to remove token from topic: ", err.Error())
		}

		err = utils.RemoveTokenFromTopic(id+"_admin", usr.UUID)
		if err != nil {
			c.Logger.Error("failed to remove token from topic: ", err.Error())
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
	}
}

/*
Leave Club (PUT)

  - Leave club

  - Grab parameters and update

Returns:

	Http handler
		- Writes OK back to client if successful
*/
func (c *Service) LeaveClub() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		uuid := r.Header.Get("UUID")

		// Grab club id from path and validate it
		id := mux.Vars(r)["id"]
		valid := utils.ValidateClubID(id)
		if !valid {
			http.Error(rw, "invalid club id", http.StatusBadRequest)
			return
		}

		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			c.Logger.Debug(err.Error())
		}
		filter := bson.M{"_id": oid}
		update := bson.M{"$pull": bson.M{"members": bson.M{"uuid": uuid}}}

		var club models.Club
		err = c.Database.ClubCol.FindOne(context.Background(), filter).Decode(&club)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				c.Logger.Error(err)
				rw.WriteHeader(http.StatusNotFound)
				rw.Write([]byte(`{ "msg": "club does not exist" }`))
				return
			}
		}

		// remove member from club
		_, err = c.Database.ClubCol.UpdateOne(context.Background(), filter, update)
		if err != nil {
			c.Logger.Error(err.Error())
			http.Error(rw, "failed to update club", http.StatusInternalServerError)
			return
		}

		// remove club id from user data
		filter = bson.M{"uuid": uuid}
		update = bson.M{"$pull": bson.M{"clubs": oid}}
		_, err = c.Database.UserCol.UpdateOne(context.TODO(), filter, update)
		if err != nil {
			c.Logger.Error(err.Error())
			http.Error(rw, "failed to update user", http.StatusInternalServerError)
			return
		}

		// remove from topics
		err = utils.RemoveTokenFromTopic(club.ID.Hex(), uuid)
		if err != nil {
			c.Logger.Error("failed to remove token from topic: ", err.Error())
		}

		err = utils.RemoveTokenFromTopic(club.ID.Hex()+"_admin", uuid)
		if err != nil {
			c.Logger.Error("failed to remove token from topic: ", err.Error())
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`OK`))
	}
}

// CLUB POST ENDPOINTS

func (s *Service) PinClubPost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Grab club id from path and validate it
		id := mux.Vars(r)["id"]
		valid := utils.ValidateClubID(id)
		if !valid {
			http.Error(w, "invalid club id", http.StatusBadRequest)
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

func (s *Service) UnpinClubPost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Grab club id from path and validate it
		id := mux.Vars(r)["id"]
		valid := utils.ValidateClubID(id)
		if !valid {
			http.Error(w, "invalid club id", http.StatusBadRequest)
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
