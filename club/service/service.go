package service

import (
	"context"
	"encoding/json"
	"net/http"
	"olympsis-server/database"
	"olympsis-server/utils"
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

type Service struct {
	Database      *database.Database
	Logger        *logrus.Logger
	Router        *mux.Router
	NotifService  *notif.Service
	SearchService *search.Service
}

/*
Create new Club service struct
*/
func NewClubService(l *logrus.Logger, r *mux.Router, d *database.Database, n *notif.Service, sh *search.Service) *Service {
	return &Service{Logger: l, Router: r, Database: d, NotifService: n, SearchService: sh}
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

		var clubs []models.Club
		err := c.FindClubs(context.TODO(), filter, &clubs)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				c.Logger.Error(err.Error())
				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(http.StatusNoContent)
				return
			} else {
				c.Logger.Error(err.Error())
				rw.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

		if len(clubs) == 0 {
			rw.WriteHeader(http.StatusNoContent)
			return
		}

		resp := models.ClubsResponse{
			TotalClubs: len(clubs),
			Clubs:      clubs,
		}

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

		uuid := r.Header.Get("UUID")

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
		var club models.Club
		filter := bson.D{primitive.E{Key: "_id", Value: oid}}
		err := c.FindClub(context.Background(), filter, &club)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				rw.WriteHeader(http.StatusNotFound)
				rw.Write([]byte(`{ "msg": "club does not exist" }`))
				return
			}
		}

		//
		if club.ID.Hex() == "000000000000000000000000" {
			rw.WriteHeader(http.StatusNotFound)
			return
		}

		var token string

		// fetch member data
		for i := 0; i < len(club.Members); i++ {
			if club.Members[i].UUID == uuid {
				if club.Members[i].Role != "member" {
					token, err = utils.GenerateClubToken(club.ID.Hex(), club.Members[i].Role, uuid)
					if err != nil {
						c.Logger.Error(err)
					}
				}
			}
			usr, err := c.SearchService.SearchUserByUUID(club.Members[i].UUID)
			if err != nil {
				club.Members[i].Data = nil
			} else {
				club.Members[i].Data = &usr
			}
		}

		resp := models.ClubResponse{
			Token: token,
			Club:  club,
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(resp)

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

		uuid := r.Header.Get("UUID")

		// decode request
		var req models.Club
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			c.Logger.Error("failed to decode request" + err.Error())
			http.Error(rw, "failed to decode request", http.StatusBadRequest)
			return
		}

		timeStamp := time.Now().Unix()
		member := models.Member{
			ID:       primitive.NewObjectID(),
			UUID:     uuid,
			Role:     "owner",
			JoinedAt: timeStamp,
		}

		club := models.Club{
			ID:          primitive.NewObjectID(),
			Name:        req.Name,
			Description: req.Description,
			Visibility:  req.Visibility,
			Sport:       req.Sport,
			City:        req.City,
			State:       req.State,
			Country:     req.Country,
			ImageURL:    req.ImageURL,
			Members:     []models.Member{member},
			Rules:       req.Rules,
			CreatedAt:   timeStamp,
		}

		// create club in database
		err = c.InsertClub(context.TODO(), &club)
		if err != nil {
			c.Logger.Error("failed to create club: " + err.Error())
			http.Error(rw, "failed to create club", http.StatusInternalServerError)
			return
		}

		// update user id to contain the new club
		filter := bson.M{"uuid": uuid}
		update := bson.M{"$push": bson.M{"clubs": club.ID}}
		_, err = c.Database.UserCol.UpdateOne(context.TODO(), filter, update)
		if err != nil {
			c.Logger.Error("failed to update user: " + err.Error())
		}

		// generate admin token
		token, err := utils.GenerateClubToken(club.ID.Hex(), "owner", uuid)
		if err != nil {
			c.Logger.Error("failed to create club" + err.Error())
			http.Error(rw, "failed to create club", http.StatusInternalServerError)
			return
		}

		// create notification topics
		clubTopic := club.ID.Hex()
		clubAdminTopic := club.ID.Hex() + "_admin"
		user, err := c.SearchService.SearchUserByUUID(uuid)
		if err != nil {
			c.Logger.Error("failed to get user(" + uuid + "): " + err.Error())
		}

		if user.DeviceToken == "" {
			// if we user is not subscribed for notifications then just create the topics
			err = c.NotifService.CreateTopic(clubTopic)
			if err != nil {
				c.Logger.Error("failed to create club topic: " + err.Error())
			}
			err = c.NotifService.CreateTopic(clubAdminTopic)
			if err != nil {
				c.Logger.Error("failed to create club admin topic: " + err.Error())
			}
		} else {
			// create topics and subscribe owner to it
			err = c.NotifService.CreateTopic(clubTopic)
			if err != nil {
				c.Logger.Error("failed to create club topic: " + err.Error())
			}
			err = c.NotifService.CreateTopic(clubAdminTopic)
			if err != nil {
				c.Logger.Error("failed to create club admin topic: " + err.Error())
			}

			err = c.NotifService.AddTokenToTopic(clubTopic, uuid)
			if err != nil {
				c.Logger.Error("failed to add token to club topic: " + err.Error())
			}
			err = c.NotifService.AddTokenToTopic(clubAdminTopic, uuid)
			if err != nil {
				c.Logger.Error("failed to add token club admin topic: " + err.Error())
			}
		}

		resp := models.CreateClubResponse{
			Token: token,
			Club:  club,
		}

		rw.WriteHeader(http.StatusCreated)
		json.NewEncoder(rw).Encode(resp)
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

		// id := r.Header.Get("clubID") // idk if i should take the id this way
		// uuid := r.Header.Get("UUID")
		// role := r.Header.Get("clubRole")

		// Grab club id from path and validate it
		id := mux.Vars(r)["id"]
		valid := utils.ValidateClubID(id)
		if !valid {
			http.Error(rw, "invalid club id", http.StatusBadRequest)
			return
		}

		var req models.Club

		// decode request
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			c.Logger.Error(err.Error())
			http.Error(rw, "failed to decode body", http.StatusBadRequest)
			return
		}

		// Conversions and new variables
		oid, _ := primitive.ObjectIDFromHex(id)
		filter := bson.M{"_id": oid}
		change := bson.M{}
		update := bson.M{"$set": change}

		// Changes map
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
			change["image_url"] = req.ImageURL
		}
		if len(req.Rules) > 0 {
			change["rules"] = req.Rules
		}

		// update club data in database
		var club models.Club
		err = c.UpdateAClub(context.Background(), filter, update)
		if err != nil {
			c.Logger.Error(err.Error())
			http.Error(rw, "internal server error", http.StatusInternalServerError)
			return
		}

		err = c.FindClub(context.TODO(), filter, &club)
		if err != nil {
			c.Logger.Error(err.Error())
			http.Error(rw, "internal server error", http.StatusInternalServerError)
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

		// uuid := r.Header.Get("UUID")
		// role := r.Header.Get("clubRole")

		// Grab club id from path and validate it
		id := mux.Vars(r)["id"]
		valid := utils.ValidateClubID(id)
		if !valid {
			http.Error(rw, "invalid club id", http.StatusBadRequest)
			return
		}

		// convert club id to oid
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			c.Logger.Debug(err.Error())
		}

		// check if club exists
		var club models.Club
		err = c.Database.ClubCol.FindOne(context.Background(), bson.M{"_id": oid}).Decode(&club)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				c.Logger.Error(err.Error())
				http.Error(rw, "club not found", http.StatusNotFound)
				return
			}
		}

		// delete topics
		clubTopic := club.ID.Hex()
		clubAdminTopic := club.ID.Hex() + "_admin"
		c.NotifService.DeleteTopic(clubTopic)
		c.NotifService.DeleteTopic(clubAdminTopic)

		// delete club from users data
		for i := 0; i < len(club.Members); i++ {
			filter := bson.M{"uuid": club.Members[i].UUID}
			update := bson.M{"$pull": bson.M{"clubs": oid}}
			c.Database.UserCol.UpdateOne(context.Background(), filter, update)
		}

		// delete club
		filter := bson.M{"_id": oid}
		_, err = c.Database.ClubCol.DeleteOne(context.TODO(), filter)
		if err != nil {
			c.Logger.Debug(err.Error())
		}

		rw.Header().Set("Content-Type", "application/json")
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
		var club models.Club
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

		member := models.Member{
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
		if club.Members[index].Role == "member" {
			c.NotifService.AddTokenToTopic(club.ID.Hex()+"_admin", usr.UUID)
		}

		// notify user that they had their rank changed
		notification := notif.Notification{
			Title: club.Name,
			Body:  text,
		}
		c.NotifService.SendNotificationToToken(&notification, usr.DeviceToken)

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
			c.Logger.Error(err.Error())
			http.Error(rw, "failed to update user", http.StatusInternalServerError)
			return
		}

		// fetch user token
		usr, err := c.SearchService.SearchUserByUUID(club.Members[index].UUID)
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
		notification := notif.Notification{
			Title: club.Name,
			Body:  "You've been kicked out of " + club.Name,
		}
		c.NotifService.SendNotificationToToken(&notification, usr.DeviceToken)
		c.NotifService.RemoveTokenFromTopic(club.ID.Hex(), usr.UUID)
		c.NotifService.RemoveTokenFromTopic(club.ID.Hex()+"_admin", usr.UUID)

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
		c.NotifService.RemoveTokenFromTopic(club.ID.Hex(), uuid)
		c.NotifService.RemoveTokenFromTopic(club.ID.Hex()+"_admin", uuid)

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

		// Grab club id from path and validate it
		id := mux.Vars(r)["id"]
		valid := utils.ValidateClubID(id)
		if !valid {
			http.Error(rw, "invalid club id", http.StatusBadRequest)
			return
		}

		// convert club id to oid
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			c.Logger.Debug(err.Error())
		}

		_, ctx := context.WithTimeout(context.Background(), 30*time.Second)
		defer ctx()

		filter := bson.M{"club_id": oid}
		var apps []models.ClubApplication
		cur, err := c.Database.ClubApplicationCol.Find(context.TODO(), filter)

		if err != nil {
			if err == mongo.ErrNoDocuments {
				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(http.StatusNoContent)
				return
			}
		}

		for cur.Next(context.TODO()) {
			var app models.ClubApplication
			err := cur.Decode(&app)
			if err != nil {
				c.Logger.Error(err)
			}
			user, err := c.SearchService.SearchUserByUUID(app.UUID)
			if err != nil {
				c.Logger.Error("failed to get user data: " + err.Error())
			}
			app.Data = &user
			apps = append(apps, app)
		}

		// just in case mongo doesnt throw an error
		if len(apps) == 0 {
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusNoContent)
			return
		}

		resp := models.ClubApplicationsResponse{
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

		// check if an application already exists
		var _app models.ClubApplication
		filter := bson.M{"uuid": uuid, "club_id": oid}
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
		app := models.ClubApplication{
			ID:        primitive.NewObjectID(),
			UUID:      uuid,
			ClubID:    oid,
			Status:    "pending",
			CreatedAt: timeStamp,
		}

		// create club application in database
		_, err = c.Database.ClubApplicationCol.InsertOne(context.Background(), app)
		if err != nil {
			c.Logger.Error(err.Error())
			http.Error(rw, "failed to create application", http.StatusInternalServerError)
			return
		}

		// notify admins
		note := notif.Notification{
			Title: "New Club Application",
			Body:  "You have a new club application",
			Topic: app.ClubID.Hex() + "_admin",
		}
		err = c.NotifService.SendNotificationToTopic(&note)
		if err != nil {
			c.Logger.Error(err.Error())
		}

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

		// Grab club id from path and validate it
		id := mux.Vars(r)["id"]
		valid := utils.ValidateClubID(id)
		if !valid {
			http.Error(rw, "invalid club id", http.StatusBadRequest)
			return
		}

		// grab club id from path
		vars := mux.Vars(r)
		// if there is no application id
		if len(vars["applicationID"]) == 0 {
			http.Error(rw, "bad application id", http.StatusBadRequest)
			return
		}

		// if we get an invalid application id
		if len(vars["applicationID"]) < 24 {
			http.Error(rw, "bad application id", http.StatusBadRequest)
			return
		}

		var req models.ApplicationUpdateRequest
		// decode request
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
			return
		}

		// convert club id to oid
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			c.Logger.Debug(err.Error())
		}

		// convert application id to oid
		aid := vars["applicationID"]
		aoid, err := primitive.ObjectIDFromHex(aid)
		if err != nil {
			c.Logger.Debug(err.Error())
		}

		// if the admin accepts the application
		if req.Status == "accepted" {

			// check if application exists
			var app models.ClubApplication
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

				member := models.Member{ // member object to put in club
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
				var club models.Club
				err = c.Database.ClubCol.FindOne(context.Background(), filter).Decode(&club)
				if err != nil {
					c.Logger.Error(err)
					rw.WriteHeader(http.StatusInternalServerError)
					rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
					return
				}

				// find user device token
				usr, err := c.SearchService.SearchUserByUUID(member.UUID)
				if err != nil {
					c.Logger.Error("failed to get user data: " + err.Error())
				}

				// notify user they were accepted to the club
				notification := notif.Notification{
					Title: "Club Application",
					Body:  club.Name + " accepted your application.",
				}
				c.NotifService.AddTokenToTopic(club.ID.Hex(), usr.UUID)
				c.NotifService.SendNotificationToToken(&notification, usr.DeviceToken)

				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(http.StatusOK)
				return
			}

			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusOK)
			return
		} else {
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
		// Check & Validate Auth Token
		uuid := r.Header.Get("UUID")

		// Grab club id from path and validate it
		id := mux.Vars(r)["id"]
		valid := utils.ValidateClubID(id)
		if !valid {
			http.Error(rw, "invalid club id", http.StatusBadRequest)
			return
		}
		// grab application id from path
		vars := mux.Vars(r)

		// if there is no application id
		if len(vars["applicationId"]) == 0 {
			http.Error(rw, "bad application id", http.StatusBadRequest)
			return
		}

		// if we get an invalid application id
		if len(vars["applicationId"]) < 24 {
			http.Error(rw, "bad application id", http.StatusBadRequest)
			return
		}

		// convert club id to oid
		appID := vars["application_id"]
		appOID, err := primitive.ObjectIDFromHex(appID)
		if err != nil {
			c.Logger.Debug(err.Error())
		}

		clubID := vars["id"]
		clubOID, err := primitive.ObjectIDFromHex(clubID)
		if err != nil {
			c.Logger.Debug(err.Error())
		}

		filter := bson.M{"_id": appOID, "uuid": uuid, "club_id": clubOID}

		// delete club application from database
		_, err = c.Database.ClubApplicationCol.DeleteOne(context.TODO(), filter)
		if err != nil {
			c.Logger.Error(err.Error())
			http.Error(rw, "failed to delete club application", http.StatusInternalServerError)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`OK`))
	}
}
