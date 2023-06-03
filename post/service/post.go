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
			user, err := p.SearchService.SearchUserByUUID(post.Poster)
			if err != nil {
				p.Logger.Error(err)
			}

			data := models.PostData{
				User: user,
			}
			post.Data = data
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
				return
			}
		}

		user, err := p.SearchService.SearchUserByUUID(post.Poster)
		if err != nil {
			p.Logger.Error(err.Error())
		}

		data := models.PostData{
			User: user,
		}
		post.Data = data

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
			p.Logger.Error(err.Error())
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		timeStamp := time.Now().Unix()
		req.CreatedAt = timeStamp
		post := models.Post{
			ID:        primitive.NewObjectID(),
			Poster:    uuid,
			ClubID:    req.ClubID,
			EventID:   req.EventID,
			Body:      req.Body,
			Images:    req.Images,
			CreatedAt: timeStamp,
		}

		// create post in database
		_, err = p.Database.PostCol.InsertOne(context.TODO(), post)
		if err != nil {
			p.Logger.Error(err.Error())
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		// grab user data
		user, err := p.SearchService.SearchUserByUUID(uuid)
		if err != nil {
			p.Logger.Error("failed to grab user data")
		}

		// topic for post
		p.NotifService.CreateTopic(post.ID.Hex())
		p.NotifService.AddTokenToTopic(post.ID.Hex(), user.UUID, user.DeviceToken)

		// grab club info for notifications
		var club models.Club
		filter := bson.M{"_id": post.ClubID}
		err = p.Database.ClubCol.FindOne(context.Background(), filter).Decode(&club)
		if err != nil {
			p.Logger.Error(err.Error())
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		// send notification to club members
		note := notif.Notification{
			Title: club.Name,
			Body:  user.FirstName + " Created a post!",
			Topic: club.ID.Hex(),
		}
		p.NotifService.SendNotificationToTopic(&note)

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
		id := vars["id"]
		if len(id) < 24 {
			p.Logger.Error("No club ID")
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "no club ID found in request." }`))
			return
		}

		var req models.Post

		// decode request
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

		// update post
		_, err = p.Database.PostCol.UpdateOne(context.TODO(), filter, update)
		if err != nil {
			p.Logger.Error(err)
			rw.WriteHeader(http.StatusInternalServerError)
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
		if len(vars["id"]) < 24 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "bad post ID" }`))
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
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		// delete notif topic
		p.NotifService.DeleteTopic(id)

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

		_, err = p.Database.CommentsCol.UpdateOne(context.TODO(), filter, change)
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
