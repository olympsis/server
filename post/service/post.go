package service

import (
	"context"
	"encoding/json"
	"net/http"
	"olympsis-server/database"
	"olympsis-server/models"
	notif "olympsis-server/pushnote/service"
	search "olympsis-server/search"
	"time"

	"github.com/gorilla/mux"
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
	NotifService  *notif.Service
}

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
		club := r.URL.Query().Get("clubId")

		if club == "" {
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "for now posts are exclusive to clubs" }`))
			return
		}

		coid, _ := primitive.ObjectIDFromHex(club)
		filter := bson.M{"clubId": coid}

		var posts []models.Post
		cur, err := p.Database.PostCol.Find(context.TODO(), filter)

		if err != nil {
			if err == mongo.ErrNoDocuments {
				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(http.StatusNoContent)
				return
			}
		}

		if cur == nil {
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusNoContent)
			return
		}

		for cur.Next(context.TODO()) {
			var post models.Post
			err := cur.Decode(&post)
			if err != nil {
				p.Logger.Error(err)
			}
			posts = append(posts, post)
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
				rw.Write([]byte(`{ "msg": "post does not exist" }`))
				return
			}
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
		// grab uuid
		uuid := r.Header.Get("UUID")

		var req models.Post

		// decode request
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
			return
		}

		timeStamp := time.Now().Unix()
		req.CreatedAt = timeStamp
		post := models.Post{
			ID:        primitive.NewObjectID(),
			Poster:    uuid,
			ClubId:    req.ClubId,
			Body:      req.Body,
			Images:    req.Images,
			Likes:     []models.Like{},
			CreatedAt: timeStamp,
		}

		// create post in database
		_, err = p.Database.PostCol.InsertOne(context.TODO(), post)
		if err != nil {
			p.Logger.Error(err)
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
			return
		}

		// grab user info for notifications
		// user := p.FetchUser(*r, uuid)
		// fN := user.FirstName + " " + user.LastName

		// grab club info for notifications
		var club models.Club
		filter := bson.M{"_id": post.ClubId}
		err = p.Database.ClubCol.FindOne(context.Background(), filter).Decode(&club)
		if err != nil {
			p.Logger.Error(err)
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
			return
		}

		// send notification to club members
		//p.SendNotificationToTopic(*r, club.Name, fN+" posted in "+club.Name, club.ID.Hex())

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
func (p *Service) UpdatePost() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// grab post id from path
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

		var req models.Post

		// decode request
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			p.Logger.Error(err)
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

		// update post
		_, err = p.Database.PostCol.UpdateOne(context.TODO(), filter, update)
		if err != nil {
			p.Logger.Error(err)
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`OK`))
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

		// if there is no club id
		if len(vars["id"]) == 0 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "no post id found in request." }`))
			return
		}

		// if we get an invalid id
		if len(vars["id"]) < 24 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "bad post ic found in request." }`))
			return
		}

		// convert club id to oid
		id := vars["id"]
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			p.Logger.Debug(err.Error())
		}

		filter := bson.D{primitive.E{Key: "_id", Value: oid}}
		_, err = p.Database.PostCol.DeleteOne(context.TODO(), filter)
		if err != nil {
			p.Logger.Debug(err.Error())
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`OK`))
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
			rw.Write([]byte(`{ "msg": "no post id found in request." }`))
			return
		}
		id := vars["id"]

		var req models.Like
		// decode request
		_ = json.NewDecoder(r.Body).Decode(&req)
		req.ID = primitive.NewObjectID()
		req.CreatedAt = time.Now().Unix()

		oid, _ := primitive.ObjectIDFromHex(id)
		filter := bson.M{"_id": oid}
		change := bson.M{"$push": bson.M{"likes": req}}

		_, err := p.Database.PostCol.UpdateOne(context.TODO(), filter, change)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
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

		lId := vars["likeId"]

		oid, _ := primitive.ObjectIDFromHex(id)
		loid, _ := primitive.ObjectIDFromHex(lId)

		match := bson.M{"_id": oid}
		change := bson.M{"$pull": bson.M{"likes": bson.M{"_id": loid}}}

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
		_ = json.NewDecoder(r.Body).Decode(&req)
		req.ID = primitive.NewObjectID()
		req.CreatedAt = time.Now().Unix()

		oid, _ := primitive.ObjectIDFromHex(id)
		filter := bson.M{"_id": oid}
		change := bson.M{"$push": bson.M{"comments": req}}

		_, err := p.Database.CommentsCol.UpdateOne(context.TODO(), filter, change)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
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
			rw.Write([]byte(`{ "msg": "no comment id found in request." }`))
			return
		}
		id := vars["id"]

		cId := vars["commentId"]

		oid, _ := primitive.ObjectIDFromHex(id)
		coid, _ := primitive.ObjectIDFromHex(cId)

		match := bson.M{"_id": oid}
		change := bson.M{"$pull": bson.M{"comments": bson.M{"_id": coid}}}

		_, err := p.Database.CommentsCol.UpdateOne(context.TODO(), match, change)
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
