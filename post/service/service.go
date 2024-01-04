package service

import (
	"context"
	"encoding/json"
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
		group := r.URL.Query().Get("groupID")
		parent := r.URL.Query().Get("parentID")

		if group == "" {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "please add a group id to your request" }`))
			return
		}

		groupID, err := primitive.ObjectIDFromHex(group)
		if err != nil {
			p.Logger.Error(err.Error())
			http.Error(rw, `{ "msg": "bad group id" }`, http.StatusBadRequest)
			return
		}
		groupIDS := bson.A{
			groupID,
		}

		if parent != "" {
			parentID, _ := primitive.ObjectIDFromHex(parent)
			groupIDS = append(groupIDS, parentID)
		}

		filter := bson.M{
			"group_id": bson.M{
				"$in": groupIDS,
			},
		}

		var posts []models.Post
		cur, err := p.Database.PostCol.Find(context.TODO(), filter)

		if err != nil {
			if err == mongo.ErrNoDocuments {
				http.Error(rw, `{ "msg": "posts not found" }`, http.StatusNoContent)
				return
			}
		}

		if cur == nil {
			http.Error(rw, `{ "msg": "posts not found" }`, http.StatusNoContent)
			return
		}

		for cur.Next(context.TODO()) {
			var post models.Post
			err := cur.Decode(&post)
			if err != nil {
				p.Logger.Error(err)
			} else {
				if post.Type == "announcement" {

					// grab org data
					var org models.Organization
					filter = bson.M{"_id": post.GroupID}
					err = p.Database.OrgCol.FindOne(context.Background(), filter).Decode(&org)
					if err != nil {
						p.Logger.Error(err.Error())
					}
					post.Data = &models.PostData{
						Organization: &org,
					}

					// in the org posts we would want to show the poster
					if group == org.ID.Hex() {
						user, err := p.SearchService.SearchUserByUUID(post.Poster)
						if err != nil {
							p.Logger.Error(err)
						}
						post.Data.Poster = &user
					}

				} else if post.Type == "post" {

					user, err := p.SearchService.SearchUserByUUID(post.Poster)
					if err != nil {
						p.Logger.Error(err)
					}
					data := models.PostData{
						Poster: &user,
					}
					post.Data = &data

				}

				// grab user data for comments
				for i := 0; i < len(post.Comments); i++ {
					usrData, err := p.SearchService.SearchUserByUUID(post.Comments[i].UUID)
					if err != nil {
						p.Logger.Error(err.Error())
					}
					post.Comments[i].Data = &usrData
				}

				posts = append(posts, post)
			}

		}

		if len(posts) == 0 {
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusNoContent)
			return
		}

		resp := models.PostsResponse{
			TotalPosts: len(posts),
			Posts:      posts,
		}

		rw.Header().Set("Content-Type", "application/json")
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
		if len(vars["id"]) < 24 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "no post id found in request." }`))
			return
		}
		id := vars["id"]
		oid, _ := primitive.ObjectIDFromHex(id)
		_, ctx := context.WithTimeout(context.Background(), 30*time.Second)
		defer ctx()

		// find post  in database
		var post models.Post
		filter := bson.D{primitive.E{Key: "_id", Value: oid}}
		err := p.Database.PostCol.FindOne(context.TODO(), filter).Decode(&post)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				rw.WriteHeader(http.StatusNotFound)
				return
			}
		}

		if post.Type == "announcement" {

			// grab org data
			var org models.Organization
			filter := bson.M{"_id": post.GroupID}
			err = p.Database.OrgCol.FindOne(context.Background(), filter).Decode(&org)
			if err != nil {
				p.Logger.Error(err.Error())
				return
			}

			post.Data = &models.PostData{
				Organization: &org,
			}

		} else if post.Type == "post" {

			user, err := p.SearchService.SearchUserByUUID(post.Poster)
			if err != nil {
				p.Logger.Error(err.Error())
			}

			data := models.PostData{
				Poster: &user,
			}
			post.Data = &data

		}

		// get comments data
		for i := 0; i < len(post.Comments); i++ {
			usrData, err := p.SearchService.SearchUserByUUID(post.Comments[i].UUID)
			if err != nil {
				p.Logger.Error(err.Error())
			}
			post.Comments[i].Data = &usrData
		}

		rw.Header().Set("Content-Type", "application/json")
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
		var req models.Post
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			p.Logger.Error(err.Error())
			http.Error(rw, `{ "msg": "bad request"}`, http.StatusBadRequest)
			return
		}

		// add aditional data to post model
		timeStamp := time.Now().Unix()
		req.CreatedAt = timeStamp
		post := models.Post{
			ID:           primitive.NewObjectID(),
			Type:         req.Type,
			Poster:       uuid,
			GroupID:      req.GroupID,
			EventID:      req.EventID,
			Body:         req.Body,
			Images:       req.Images,
			CreatedAt:    timeStamp,
			ExternalLink: req.ExternalLink,
		}

		// create post in database
		_, err = p.Database.PostCol.InsertOne(context.TODO(), post)
		if err != nil {
			p.Logger.Error(err.Error())
			http.Error(rw, `{ "msg": "failed to create post"}`, http.StatusInternalServerError)
			return
		}

		// create post notif topic
		p.NotifService.CreateTopic(post.ID.Hex())

		if req.Type == "announcement" {

			// grab org data
			var org models.Organization
			filter := bson.M{"_id": post.GroupID}
			err = p.Database.OrgCol.FindOne(context.Background(), filter).Decode(&org)
			if err != nil {
				p.Logger.Error(err.Error())
				rw.WriteHeader(http.StatusCreated)
				json.NewEncoder(rw).Encode(post)
				return
			}

			post.Data = &models.PostData{
				Organization: &org,
			}

			for i := 0; i < len(org.Members); i++ {
				p.NotifService.AddTokenToTopic(post.ID.Hex(), org.Members[i].UUID)
			}

			// find child clubs
			cur, err := p.Database.ClubCol.Find(context.TODO(), bson.M{"parent_id": org.ID})
			if err != nil {
				p.Logger.Error("No children found")
				rw.WriteHeader(http.StatusCreated)
				json.NewEncoder(rw).Encode(post)
			}

			// send a notification to all of them
			for cur.Next(context.TODO()) {
				var club models.Club
				err := cur.Decode(&club)
				if err != nil {
					p.Logger.Error(err)
				}
				// send notification to club members
				note := notif.Notification{
					Title: org.Name,
					Body:  "New announcement!",
					Topic: club.ID.Hex(),
				}
				p.NotifService.SendNotificationToTopic(&note)
			}

		} else if req.Type == "post" {

			// grab club info
			var club models.Club
			filter := bson.M{"_id": post.GroupID}
			err = p.Database.ClubCol.FindOne(context.Background(), filter).Decode(&club)
			if err != nil {
				p.Logger.Error(err.Error())
				rw.WriteHeader(http.StatusCreated)
				json.NewEncoder(rw).Encode(post)
				return
			}

			// grab user info
			user, err := p.SearchService.SearchUserByUUID(uuid)
			if err != nil {
				p.Logger.Error(err.Error())
				rw.WriteHeader(http.StatusCreated)
				json.NewEncoder(rw).Encode(post)
				return
			}

			post.Data = &models.PostData{
				Poster: &user,
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
		json.NewEncoder(rw).Encode(post)
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
			p.Logger.Error("No club ID")
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "no club ID found in request." }`))
			return
		}

		// decode request
		var req models.Post
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			p.Logger.Error(err.Error())
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
			return
		}

		oid, _ := primitive.ObjectIDFromHex(id)
		filter := bson.M{"_id": oid}
		change := bson.M{}
		update := bson.M{"$set": change}

		if req.Body != "" {
			change["body"] = req.Body
		}
		if len(req.Images) > 0 {
			change["images"] = req.Images
		}
		if req.ExternalLink != "" {
			change["external_link"] = req.ExternalLink
		}

		// update post
		_, err = p.Database.PostCol.UpdateOne(context.TODO(), filter, update)
		if err != nil {
			p.Logger.Error(err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
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

		// if there is no post id
		if len(id) < 24 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "bad post id" }`))
			return
		}

		// convert post id to oid

		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			p.Logger.Debug(err.Error())
		}

		filter := bson.M{"_id": oid}
		_, err = p.Database.PostCol.DeleteOne(context.TODO(), filter)
		if err != nil {
			p.Logger.Debug(err.Error())
			rw.WriteHeader(http.StatusInternalServerError)
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
		// grab id from path
		vars := mux.Vars(r)
		if len(vars["id"]) < 24 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "bad post id" }`))
			return
		}
		id := vars["id"]

		var req models.Like
		// decode request
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			p.Logger.Error(err.Error())
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		req.ID = primitive.NewObjectID()
		req.CreatedAt = time.Now().Unix()

		oid, _ := primitive.ObjectIDFromHex(id)
		filter := bson.M{"_id": oid}
		change := bson.M{"$push": bson.M{"likes": req}}

		_, err = p.Database.PostCol.UpdateOne(context.TODO(), filter, change)
		if err != nil {
			p.Logger.Error(err.Error())
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(req)
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
		if len(vars["id"]) < 24 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "no post id found in request." }`))
			return
		}
		id := vars["id"]

		lId := vars["likeID"]

		oid, _ := primitive.ObjectIDFromHex(id)
		loid, _ := primitive.ObjectIDFromHex(lId)

		match := bson.M{"_id": oid}
		change := bson.M{"$pull": bson.M{"likes": bson.M{"_id": loid}}}

		_, err := p.Database.PostCol.UpdateOne(context.TODO(), match, change)
		if err != nil {
			p.Logger.Error(err.Error())
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`OK`))
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
		// grab id from path
		vars := mux.Vars(r)
		if len(vars["id"]) < 24 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "no post id found in request." }`))
			return
		}
		id := vars["id"]

		var req models.Comment
		// decode request
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			p.Logger.Error(err.Error())
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		req.ID = primitive.NewObjectID()
		req.CreatedAt = time.Now().Unix()

		oid, _ := primitive.ObjectIDFromHex(id)
		filter := bson.M{"_id": oid}
		change := bson.M{"$push": bson.M{"comments": req}}

		_, err = p.Database.PostCol.UpdateOne(context.TODO(), filter, change)
		if err != nil {
			p.Logger.Error(err.Error())
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(req)
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
		if len(vars["id"]) < 24 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "bad comment ID" }`))
			return
		}
		id := vars["id"]

		cId := vars["commentID"]

		oid, _ := primitive.ObjectIDFromHex(id)
		coid, _ := primitive.ObjectIDFromHex(cId)

		match := bson.M{"_id": oid}
		change := bson.M{"$pull": bson.M{"comments": bson.M{"_id": coid}}}

		_, err := p.Database.PostCol.UpdateOne(context.TODO(), match, change)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`OK`))
	}
}
