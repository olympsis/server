package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"olympsis-server/database"
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

/*
Create new Post service struct

  - Create and Returns a pointer to a new post service struct
*/
func NewPostService(l *logrus.Logger, r *mux.Router, d *database.Database, n *notif.Service, sh *search.Service) *Service {
	return &Service{Logger: l, Router: r, Database: d, NotifService: n, SearchService: sh}
}

/*
Get Posts (GET)

  - Fetches and returns a list of posts

  - Grab query params

  - Filter and Search Posts

    Returns:
    Http handler

  - Writes list of posts objects back to client
*/
func (p *Service) GetPosts() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// grab query parameters
		group := r.URL.Query().Get("groupID")
		parent := r.URL.Query().Get("parentID")
		if group == "" {
			http.Error(rw, `{ "msg" : "no group id found in request"}`, http.StatusBadRequest)
			return
		}

		// convert ids found to objectIDs
		var ids []primitive.ObjectID
		groupID, err := primitive.ObjectIDFromHex(group)
		if err != nil {
			p.Logger.Error("failed to encode group id to object id: ", err.Error())
			http.Error(rw, `{ "msg" : "bad group id found in request"}`, http.StatusBadRequest)
			return
		} else {
			ids = append(ids, groupID)
		}
		if parent != "" {
			parentID, _ := primitive.ObjectIDFromHex(parent)
			ids = append(ids, parentID)
		}

		// run aggregation pipeline
		posts, err := FindPosts(ids, p.Database, 100)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				http.Error(rw, `{ "msg": "posts not found" }`, http.StatusNoContent)
				return
			} else {
				p.Logger.Error("failed to get posts: ", err.Error())
				http.Error(rw, `{ "msg": "posts not found" }`, http.StatusInternalServerError)
			}
		}
		if posts == nil || len(*posts) == 0 {
			http.Error(rw, `{ "msg": "no posts content not found" }`, http.StatusNoContent)
			return
		}

		resp := models.PostsResponse{
			TotalPosts: len(*posts),
			Posts:      *posts,
		}
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(resp)
	}
}

/*
Get a Post (GET)

  - Fetches and returns a post object

  - Grab path values

Returns:

	Http handler

		- Writes a post object back to client
*/
func (p *Service) GetPost() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(rw, `{ "msg": "no post id found in request." }`, http.StatusBadRequest)
			return
		}

		// convert id to objectID
		oid, _ := primitive.ObjectIDFromHex(id)

		// run aggregation pipeline to fetch post
		post, err := FindPost(oid, p.Database)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				http.Error(rw, `{"msg": "post not found"}`, http.StatusNotFound)
				return
			}
			p.Logger.Error("failed to get post: ", err.Error())
			http.Error(rw, `{"msg":"failed to get post"}`, http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(post)
	}
}

/*
Create Post (POST)

  - Creates new post for club

  - Grab request body

  - Create post in database

    Returns:

Http handler

  - Writes object back to client
*/
func (p *Service) CreatePost() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// grab uuid of the user who made this request
		uuid := r.Header.Get("UUID")

		// decode request
		var req models.PostDao
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			p.Logger.Error("failed to decode request: ", err.Error())
			http.Error(rw, `{ "msg": "failed to decode request" }`, http.StatusBadRequest)
			return
		}

		// add aditional data to post model
		id := primitive.NewObjectID()
		timeStamp := time.Now().Unix()
		req.CreatedAt = &timeStamp
		post := models.PostDao{
			ID:           &id,
			Type:         req.Type,
			Poster:       &uuid,
			GroupID:      req.GroupID,
			EventID:      req.EventID,
			Body:         req.Body,
			Images:       req.Images,
			CreatedAt:    &timeStamp,
			ExternalLink: req.ExternalLink,
		}

		// create post in database
		_, err = p.Database.PostCol.InsertOne(context.TODO(), post)
		if err != nil {
			p.Logger.Error("failed to create post: ", err.Error())
			http.Error(rw, `{ "msg": "failed to create post"}`, http.StatusInternalServerError)
			return
		}

		// create post notif topic
		p.NotifService.CreateTopic(post.ID.Hex())

		if *req.Type == "announcement" {

			// grab org data
			var org models.Organization
			filter := bson.M{"_id": post.GroupID}
			err = p.Database.OrgCol.FindOne(context.Background(), filter).Decode(&org)
			if err != nil {
				p.Logger.Error("failed to find organization: ", err.Error())
				rw.WriteHeader(http.StatusCreated)
				return
			}

			for i := 0; i < len(org.Members); i++ {
				p.NotifService.AddTokenToTopic(post.ID.Hex(), org.Members[i].UUID)
			}

			// find child clubs
			cur, err := p.Database.ClubCol.Find(context.TODO(), bson.M{"parent_id": org.ID})
			if err != nil {
				p.Logger.Error("No children found")
				rw.WriteHeader(http.StatusCreated)
				rw.Write([]byte(`{"id": "` + post.ID.Hex() + `" }`))
			}

			// send a notification to all of them
			for cur.Next(context.TODO()) {
				var club models.Club
				err := cur.Decode(&club)
				if err != nil {
					p.Logger.Error("failed to decode club: ", err.Error())
				}

				// send notification to club members
				note := notif.Notification{
					Title: org.Name,
					Body:  "New announcement!",
					Topic: club.ID.Hex(),
				}
				p.NotifService.SendNotificationToTopic(&note)
			}

		} else if *req.Type == "post" {

			// grab club info
			var club models.Club
			filter := bson.M{"_id": post.GroupID}
			err = p.Database.ClubCol.FindOne(context.Background(), filter).Decode(&club)
			if err != nil {
				p.Logger.Error("failed to fetch club data: ", err.Error())
				rw.WriteHeader(http.StatusCreated)
				rw.Write([]byte(`{"id": "` + post.ID.Hex() + `" }`))
				return
			}

			// grab user info
			user, err := p.SearchService.SearchUserByUUID(uuid)
			if err != nil {
				p.Logger.Error("failed to fetch user data: ", err.Error())
				rw.WriteHeader(http.StatusCreated)
				rw.Write([]byte(`{"id": "` + post.ID.Hex() + `" }`))
				return
			}

			// send notification to club members
			note := notif.Notification{
				Title: club.Name,
				Body:  user.Username + " created a post!",
				Topic: club.ID.Hex(),
			}
			p.NotifService.SendNotificationToTopic(&note)
		}

		rw.WriteHeader(http.StatusCreated)
		rw.Write([]byte(`{"id": "` + post.ID.Hex() + `" }`))
	}
}

/*
Update Post (POST)

  - Grab Post Id from path

  - Update post data

  - Grab request body

  - updated post data in databse

Returns:

	Http handler

		- Writes object back to client
*/
func (p *Service) ModifyPost() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// grab post id from path
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(rw, `{ "msg" : "bad/no club id found in request" }`, http.StatusBadRequest)
			return
		}

		// decode request
		var req models.PostDao
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			p.Logger.Error("failed to decode request: ", err.Error())
			http.Error(rw, `{ "msg": "failed to decode request" }`, http.StatusBadRequest)
			return
		}

		oid, _ := primitive.ObjectIDFromHex(id)
		filter := bson.M{"_id": oid}
		change := bson.M{}
		update := bson.M{"$set": change}

		if req.Body != nil {
			change["body"] = req.Body
		}
		if req.EventID != nil {
			change["event_id"] = req.EventID
		}
		if req.Images != nil {
			change["images"] = req.Images
		}
		if req.ExternalLink != nil {
			change["external_link"] = req.ExternalLink
		}

		// update post
		_, err = p.Database.PostCol.UpdateOne(context.TODO(), filter, update)
		if err != nil {
			p.Logger.Error("failed to update post: ", err.Error())
			http.Error(rw, `{ "msg" : "failed to update post" }`, http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusOK)
	}
}

/*
Delete Post (DELETE)

  - Deletes post object

  - Grab id from path

  - Delete post

Returns:

	Http handler

		- Writes OK back to client if successful
*/
func (p *Service) DeletePost() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// grab club id from path
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(rw, `{ "mgs" : "bad/no post id found in request"}`, http.StatusBadRequest)
			return
		}

		// convert post id to oid
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			p.Logger.Debug("failed to convert post id to object id: ", err.Error())
			http.Error(rw, `{ "msg": "failed to convert id to object id" }`, http.StatusInternalServerError)
		}

		filter := bson.M{"_id": oid}
		_, err = p.Database.PostCol.DeleteOne(context.TODO(), filter)
		if err != nil {
			p.Logger.Debug("failed to delete post: ", err.Error())
			http.Error(rw, `{ "msg": "failed to delete post" }`, http.StatusInternalServerError)
			return
		}

		// delete notif topic
		p.NotifService.DeleteTopic(id)
		rw.WriteHeader(http.StatusOK)
	}
}

/* LIKES */
/*
Add Like (POST)

  - grab post id from path

  - decode request body

  - add like to event

Returns:

	Http handler
		- Writes back like object to client
*/
func (p *Service) AddLike() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// grab uuid of the user who made this request
		uuid := r.Header.Get("UUID")

		// grab id from path
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(rw, `{ "msg": "bad post id" }`, http.StatusBadRequest)
			return
		}

		like := models.Like{
			ID:        primitive.NewObjectID(),
			UUID:      uuid,
			CreatedAt: time.Now().Unix(),
		}

		oid, _ := primitive.ObjectIDFromHex(id)
		filter := bson.M{"_id": oid, "likes.uuid": bson.M{"$ne": uuid}}
		change := bson.M{
			"$push": bson.M{
				"likes": like,
			},
		}

		resp, err := p.Database.PostCol.UpdateOne(context.TODO(), filter, change)
		if err != nil { // unexpected error
			p.Logger.Error("failed to add like: ", err.Error())
			http.Error(rw, `{ "msg" : "failed to add like" }`, http.StatusInternalServerError)
			return
		} else if resp.ModifiedCount != 1 { // the like already exits
			rw.WriteHeader(http.StatusOK)
			return
		} else { // newly created like
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte(fmt.Sprintf(`{ "id": "%s" }`, like.ID.Hex())))
		}
	}
}

/*
Remove Like (DELETE)

  - grab post id from path

  - grab like id from path

  - pull like from event

Returns:

	Http handler
		- Writes back OK to client
*/
func (p *Service) RemoveLike() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// grab id from path
		vars := mux.Vars(r)
		id := vars["id"]
		lId := vars["likeID"]
		if len(id) < 24 {
			http.Error(rw, `{ "msg" : "bad/no post id found in request." }`, http.StatusBadRequest)
			return
		}
		if len(lId) < 24 {
			http.Error(rw, `{ "msg" : "bad/no like id found in request." }`, http.StatusBadRequest)
			return
		}

		oid, _ := primitive.ObjectIDFromHex(id)
		loid, _ := primitive.ObjectIDFromHex(lId)

		match := bson.M{"_id": oid}
		change := bson.M{"$pull": bson.M{"likes": bson.M{"_id": loid}}}

		_, err := p.Database.PostCol.UpdateOne(context.TODO(), match, change)
		if err != nil {
			p.Logger.Error("failed to remove like: ", err.Error())
			http.Error(rw, `{ "msg" : "failed to remove like" }`, http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusOK)
	}
}

/*
Add Comment (POST)

  - grab post id from path

  - decode request body

  - add comment to event

Returns:

	Http handler
		- Writes back like object to client
*/
func (p *Service) AddComment() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// grab uuid of the user who made this request
		uuid := r.Header.Get("UUID")

		// grab id from path
		vars := mux.Vars(r)
		if len(vars["id"]) < 24 {
			http.Error(rw, `{ "msg": "no post id found in request." }`, http.StatusBadRequest)
			return
		}
		id := vars["id"]

		var req models.CommentDao
		// decode request
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			p.Logger.Error("failed to decode comments: ", err.Error())
			http.Error(rw, `{ "msg": "failed to decode comments" }`, http.StatusBadRequest)
			return
		}

		newID := primitive.NewObjectID()
		timestamp := time.Now().Unix()

		req.ID = &newID
		req.UUID = &uuid
		req.CreatedAt = &timestamp

		oid, _ := primitive.ObjectIDFromHex(id)
		filter := bson.M{"_id": oid}
		change := bson.M{"$push": bson.M{"comments": req}}

		_, err = p.Database.PostCol.UpdateOne(context.TODO(), filter, change)
		if err != nil {
			p.Logger.Error("failed to update post: ", err.Error())
			http.Error(rw, `{ "msg": "failed to update post" }`, http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(fmt.Sprintf(`{ "id": "%s" }`, req.ID.Hex())))
	}
}

/*
Remove Like (DELETE)

  - grab post id from path

  - grab comment id from path

  - pull comment from event

Returns:

	Http handler
		- Writes back OK to client
*/
func (p *Service) RemoveComment() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// grab id from path
		vars := mux.Vars(r)
		id := vars["id"]
		cId := vars["commentID"]
		if len(id) < 24 {
			http.Error(rw, `{ "msg" : "bad/no post id found in request" }`, http.StatusBadRequest)
			return
		}
		if len(cId) < 24 {
			http.Error(rw, `{ "msg" : "bad/no comment id found in request" }`, http.StatusBadRequest)
			return
		}

		oid, _ := primitive.ObjectIDFromHex(id)
		coid, _ := primitive.ObjectIDFromHex(cId)

		match := bson.M{"_id": oid}
		change := bson.M{"$pull": bson.M{"comments": bson.M{"_id": coid}}}

		_, err := p.Database.PostCol.UpdateOne(context.TODO(), match, change)
		if err != nil {
			p.Logger.Error("failed to remove comment: ", err.Error())
			http.Error(rw, `{ "msg" : "failed to remove comment" }`, http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusOK)
	}
}
