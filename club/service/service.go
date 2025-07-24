package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"olympsis-server/aggregations"
	"olympsis-server/database"
	"olympsis-server/server"
	"olympsis-server/utils"
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
	Notification  *utils.NotificationInterface
}

/*
Create new Club service struct
*/
func NewClubService(i *server.ServerInterface) *Service {
	return &Service{
		Logger:        i.Logger,
		Router:        i.Router,
		Database:      i.Database,
		SearchService: i.Search,
		Notification:  i.Notification,
	}
}

// Fetches all of the clubs in a given location
func (s *Service) GetClubs() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		params, err := parseQueryParams(r)
		if err != nil {
			http.Error(rw, `{"msg": "bad request"}`, http.StatusBadRequest)
			s.Logger.Error("Bad request. Error: ", err.Error())
			return
		}

		filter := bson.M{}

		// Location Query - Only add if we have enough info
		if params.Country != "" {
			filter["country"] = params.Country

			if params.State != "" {
				filter["state"] = params.State

				if params.City != "" {
					filter["city"] = params.City
				}
			}
		}

		// Sports Query
		if len(params.Sports) > 0 {
			filter["sports"] = bson.M{
				"$in": params.Sports,
			}
		}

		// Default values for radius if needed
		radiusValue := float64(16000) // Default radius in meters
		if params.Radius != nil {
			radiusValue = *params.Radius
		}

		// Get all of the clubs data
		clubs, err := aggregations.AggregateClubs(
			filter,          // Regular filter for country/state/city/sports
			params.Location, // GeoJSON location if provided
			radiusValue,     // Radius (with default)
			params.Limit,    // Use the limit from params
			params.Skip,     // Use the skip from params
			s.Database,
		)

		if err != nil {
			s.Logger.Error("Failed to find clubs: ", err.Error())
			http.Error(rw, `{ "msg": "failed to find clubs" }`, http.StatusInternalServerError)
			return
		}

		// No content
		if len(*clubs) == 0 {
			s.Logger.Error("No clubs found matching criteria")
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

		// Validate club ID
		id := mux.Vars(r)["id"]
		oid, err := utils.ValidateObjectID(id)
		if err != nil {
			utils.HandleInvalidIDError(rw)
			c.Logger.Error("Invalid Club ID - Error: ", err.Error())
			return
		}
		_, ctx := context.WithTimeout(context.Background(), 30*time.Second)
		defer ctx()

		// Find club in database
		club, err := aggregations.AggregateClub(oid, c.Database)
		if err != nil {
			utils.HandleFindError(rw, err)
			c.Logger.Error(fmt.Sprintf("Failed to find club. ID: %s - Error: %s", id, err.Error()))
			return
		}

		// If club object is malformed
		if utils.ValidateClubObject(club) {
			http.Error(rw, `{ "msg": "something went wrong" }`, http.StatusInternalServerError)
			c.Logger.Error(fmt.Sprintf("Club Object is malformed. ID: %s", id))
			return
		}

		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(club)
	}
}

// Creates a new club
func (c *Service) CreateClub() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		ctx, cancel := context.WithTimeout(r.Context(), time.Second*15)
		defer cancel()

		uuid := r.Header.Get("UUID")

		// Decode request
		var req models.ClubDao
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			c.Logger.Error("Failed to decode request ", err.Error())
			http.Error(rw, "Failed to decode request", http.StatusBadRequest)
			return
		}

		timeStamp := primitive.NewDateTimeFromTime(time.Now())
		verification := false

		club := models.ClubDao{
			Name:        req.Name,
			Description: req.Description,
			Tags:        req.Tags,
			Sports:      req.Sports,
			City:        req.City,
			State:       req.State,
			Country:     req.Country,
			Location:    req.Location,
			Logo:        req.Logo,
			Banner:      req.Banner,
			Visibility:  req.Visibility,
			BlackList:   req.BlackList,
			Rules:       req.Rules,
			IsVerified:  &verification,
			CreatedAt:   &timeStamp,
		}

		// Create club in database
		id, err := c.InsertClub(context.TODO(), &club)
		if err != nil {
			c.Logger.Error("Failed to create club: ", err.Error())
			http.Error(rw, "Failed to create club", http.StatusInternalServerError)
			return
		}

		// Insert owner into members collection
		member := models.MemberDao{
			ID:       primitive.NewObjectID(),
			UserID:   uuid,
			Role:     "owner",
			ClubID:   id,
			JoinedAt: timeStamp,
		}
		_, err = c.InsertMember(ctx, &member)
		if err != nil {
			c.Logger.Error("Failed to create member owner. Error: ", err.Error())

			_, err = c.InsertMember(ctx, &member)
			if err != nil {
				c.Logger.Error("Failed to create member owner x2. Error: ", err.Error())
				http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
				return
			}
		}

		// create notification topics
		topicName := id.Hex()
		adminName := id.Hex() + "_admin"
		clubTopic := models.NotificationTopicDao{
			Name:  &topicName,
			Users: &[]string{uuid},
		}
		adminTopic := models.NotificationTopicDao{
			Name:  &adminName,
			Users: &[]string{uuid},
		}

		err = c.Notification.CreateTopic(r.Header.Get("Authorization"), clubTopic)
		if err != nil {
			c.Logger.Error("Failed to create club topic: ", err.Error())
		}

		err = c.Notification.CreateTopic(r.Header.Get("Authorization"), adminTopic)
		if err != nil {
			c.Logger.Error("Failed to create club admin topic: ", err.Error())
		}

		rw.WriteHeader(http.StatusCreated)
		rw.Write(fmt.Appendf(nil, `{"id": "%s"}`, id.Hex()))
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
		rw.Write([]byte(`{"msg": "OK"}`))
	}
}

// Deletes a club
func (c *Service) DeleteClub() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		ctx, cancel := context.WithTimeout(r.Context(), time.Second*15)
		defer cancel()

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

		// Delete club
		err = c.RemoveClub(ctx, bson.M{"_id": oid})
		if err != nil {
			c.Logger.Error("Failed to delete club. Error: ", err.Error())
			http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			return
		}

		// Delete club members
		err = c.DeleteMembers(ctx, bson.M{"club_id": oid})
		if err != nil {
			c.Logger.Error("Failed to delete club members. Error: ", err.Error())
			http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			return
		}

		// delete topics
		clubTopic := id
		clubAdminTopic := id + "_admin"

		err = c.Notification.DeleteTopic(r.Header.Get("Authorization"), clubTopic)
		if err != nil {
			c.Logger.Error("failed to delete topic", err.Error())
		}

		err = c.Notification.DeleteTopic(r.Header.Get("Authorization"), clubAdminTopic)
		if err != nil {
			c.Logger.Error("failed to delete topic", err.Error())
		}

		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{ "msg": "OK" }`))
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

		ctx, cancel := context.WithTimeout(r.Context(), time.Second*15)
		defer cancel()

		// grab club id from path
		vars := mux.Vars(r)

		// Grab club id from path and validate it
		id := mux.Vars(r)["id"]
		valid := utils.ValidateClubID(id)
		if !valid {
			http.Error(rw, `{"msg": "invalid club id"}`, http.StatusBadRequest)
			return
		}

		// if there is no member id
		if len(vars["memberID"]) == 0 || len(vars["memberID"]) < 24 {
			http.Error(rw, `{"msg": "invalid member id"}`, http.StatusBadRequest)
			return
		}

		// convert club id to oid
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			c.Logger.Error("Failed to convert club id to ObjectID. Error: ", err.Error())
			http.Error(rw, `{"msg": "something went wrong"}`, http.StatusBadRequest)
			return
		}

		// convert club id to oid
		memID := vars["memberID"]
		memOID, err := primitive.ObjectIDFromHex(memID)
		if err != nil {
			c.Logger.Error("Failed to convert member id to ObjectID. Error: ", err.Error())
			http.Error(rw, `{"msg": "something went wrong"}`, http.StatusBadRequest)
		}

		// Decode request
		var req models.ChangeRoleRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Find member
		member, err := c.FindMember(ctx, bson.M{"_id": memOID})
		if err != nil {
			if err == mongo.ErrNoDocuments {
				http.Error(rw, `{"msg": "member not found"}`, http.StatusNotFound)
				return
			}

			c.Logger.Error("Failed to find member. Error: ", err.Error())
			http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			return
		}

		// Update member
		err = c.UpdateMember(ctx, bson.M{"_id": memOID}, bson.M{"role": req.Role})
		if err != nil {
			c.Logger.Error("Failed to update member. Error: ", err.Error())
			http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			return
		}

		var noteText string
		switch req.Role {
		case "owner":
			noteText = "You've been promoted to Owner"
		case "admin":
			noteText = "You've been promoted to Admin"
		default:
			noteText = "You've been demoted to Member"
		}

		// if user was member then add them to the admin topic
		if member.Role == "member" {
			err = c.Notification.ModifyTopic(r.Header.Get("Authorization"), id+"_admin", models.NotificationTopicUpdateRequest{
				Action: "subscribe",
				Users:  []string{member.UserID},
			})
			if err != nil {
				c.Logger.Error("failed to add user to topic: ", err.Error())
			}
		}

		club, err := c.FindClub(ctx, bson.M{"_id": oid})
		if err != nil {
			c.Logger.Error("Failed to find club. Error: ", err.Error())
			http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			return
		}

		// notify user that they had their rank changed
		note := models.PushNotification{
			Title: *club.Name,
			Body:  noteText,
			Data: map[string]interface{}{
				"club_id": id,
			},
		}
		err = c.Notification.SendNotification(r.Header.Get("Authorization"), models.NotificationPushRequest{
			Users:        &[]string{member.UserID},
			Notification: note,
		})
		if err != nil {
			c.Logger.Error("Failed to add user to admin topic")
		}

		// fetch updated club data
		err = c.Database.ClubCol.FindOne(context.Background(), bson.M{"_id": oid}).Decode(&club)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				http.Error(rw, "club not found", http.StatusNotFound)
				return
			}
		}

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

		ctx, cancel := context.WithTimeout(r.Context(), time.Second*15)
		defer cancel()

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

		// Find club
		club, err := c.FindClub(ctx, bson.M{"_id": oid})
		if err != nil {
			if err == mongo.ErrNoDocuments {
				c.Logger.Error("Failed to find club. It does not exist.")
				http.Error(rw, `{"msg": "club not found"}`, http.StatusNotFound)
				return
			}

			c.Logger.Error("Failed to find club. Error: ", err.Error())
			http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			return
		}

		// Find member
		member, err := c.FindMember(ctx, bson.M{"_id": memOID})
		if err != nil {
			if err == mongo.ErrNoDocuments {
				c.Logger.Error("Failed to find member. It does not exist.")
				http.Error(rw, `{"msg": "member not found"}`, http.StatusNotFound)
				return
			}

			c.Logger.Error("Failed to find member. Error: ", err.Error())
			http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			return
		}

		// Remove user from members collection
		err = c.DeleteMember(ctx, bson.M{"_id": memOID})
		if err != nil {
			c.Logger.Error("Failed to find remove member. Error: ", err.Error())
			http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			return
		}

		// Notify the user that they have been kicked out of the club
		note := models.PushNotification{
			Title: *club.Name,
			Body:  fmt.Sprintf(`You've been kicked out of %s`, *club.Name),
		}
		err = c.Notification.SendNotification(r.Header.Get("Authorization"), models.NotificationPushRequest{
			Users:        &[]string{member.UserID},
			Notification: note,
		})
		if err != nil {
			c.Logger.Error("failed to send user a notification: ", err.Error())
		}

		clubTopic := club.ID.Hex()
		adminTopic := club.ID.Hex() + "_admin"
		req := models.NotificationTopicUpdateRequest{
			Action: "unsubscribe",
			Users:  []string{member.UserID},
		}
		err = c.Notification.ModifyTopic(r.Header.Get("Authorization"), clubTopic, req)
		if err != nil {
			c.Logger.Error("failed to remove token from topic: ", err.Error())
		}

		err = c.Notification.ModifyTopic(r.Header.Get("Authorization"), adminTopic, req)
		if err != nil {
			c.Logger.Error("failed to remove token from admin topic: ", err.Error())
		}

		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{"msg": "OK"}`))
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

		ctx, cancel := context.WithTimeout(r.Context(), time.Second*15)
		defer cancel()

		uuid := r.Header.Get("UUID")

		// Grab club id from path and validate it
		id := mux.Vars(r)["id"]
		valid := utils.ValidateClubID(id)
		if !valid {
			http.Error(rw, `{"msg": "invalid club id"}`, http.StatusBadRequest)
			return
		}

		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			c.Logger.Error("Failed to encode club id to ObjectID. Error: ", err.Error())
			http.Error(rw, `{"msg": "failed to decode club id"}`, http.StatusBadRequest)
			return
		}

		// Remove member
		err = c.DeleteMember(ctx, bson.M{"user_id": uuid, "club_id": oid})
		if err != nil {
			c.Logger.Error("Failed to delete member. Error: ", err.Error())
			http.Error(rw, `{"mgs": "something went wrong"}`, http.StatusInternalServerError)
			return
		}

		// remove from topics
		topicName := id
		adminName := id + "_admin"
		request := models.NotificationTopicUpdateRequest{
			Action: "unsubscribe",
			Users:  []string{uuid},
		}
		err = c.Notification.ModifyTopic(r.Header.Get("Authorization"), topicName, request)
		if err != nil {
			c.Logger.Error("failed to remove token from topic: ", err.Error())
		}

		err = c.Notification.ModifyTopic(r.Header.Get("Authorization"), adminName, request)
		if err != nil {
			c.Logger.Error("failed to remove token from topic: ", err.Error())
		}

		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{ "msg": "OK" }`))
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
			w.Write([]byte(`{"msg": "OK"}`))
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
			w.Write([]byte(`{"msg": "OK"}`))
			return
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}
