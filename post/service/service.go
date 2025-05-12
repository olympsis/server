package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"olympsis-server/aggregations"
	"olympsis-server/server"
	"time"

	"github.com/gorilla/mux"
	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

/*
Create new Post service struct

  - Create and Returns a pointer to a new post service struct
*/
func NewPostService(i *server.ServerInterface) *Service {
	return &Service{
		Logger:        i.Logger,
		Router:        i.Router,
		Database:      i.Database,
		SearchService: i.Search,
		Notification:  i.Notification,
	}
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

		params, err := parsePostQueryParams(r)
		if err != nil {
			p.Logger.Error("Failed to parse request. Error: ", err.Error())
			http.Error(rw, `{"msg": "bad request"}`, http.StatusBadRequest)
		}

		filter := bson.M{}
		groupID, err := primitive.ObjectIDFromHex(params.GroupID)
		if err != nil {
			p.Logger.Error("Failed to encode group id to object id: ", err.Error())
			http.Error(rw, `{ "msg" : "bad group id found in request"}`, http.StatusBadRequest)
			return
		}

		if params.ParentID != nil && *params.ParentID != "" {
			parentID, _ := primitive.ObjectIDFromHex(*params.ParentID)
			filter["$or"] = bson.A{
				bson.M{"group_id": groupID},
				bson.M{"group_id": parentID},
			}
		} else {
			filter["group_id"] = groupID
		}

		posts, err := aggregations.AggregatePosts(filter, 20, 0, p.Database)

		if posts == nil || len(*posts) == 0 {
			http.Error(rw, `{ "msg": "no posts found" }`, http.StatusNoContent)
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
		post, err := aggregations.AggregatePost(oid, p.Database)
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

		// Context setup
		ctx, cancel := context.WithTimeout(r.Context(), time.Second*15)
		defer cancel()

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

		// Add additional data to post model
		timestamp := primitive.NewDateTimeFromTime(time.Now())
		req.CreatedAt = &timestamp
		post := models.PostDao{
			Type:         req.Type,
			Poster:       &uuid,
			GroupID:      req.GroupID,
			EventID:      req.EventID,
			Body:         req.Body,
			Images:       req.Images,
			IsSensitive:  req.IsSensitive,
			ExternalLink: req.ExternalLink,
			CreatedAt:    &timestamp,
		}

		// Create post in database
		id, err := p.InsertPost(context.TODO(), &post, nil)
		if err != nil {
			p.Logger.Error("failed to create post: ", err.Error())
			http.Error(rw, `{ "msg": "failed to create post"}`, http.StatusInternalServerError)
			return
		}
		if id == nil {
			p.Logger.Error("inserted post id is null")
			http.Error(rw, `{"msg": "failed to add post to database"}`, http.StatusInternalServerError)
			return
		}

		// Create post notification topic
		postID := id.Hex()
		topic := models.NotificationTopicDao{
			Name:  &postID,
			Users: &[]string{uuid},
		}
		err = p.Notification.CreateTopic(r.Header.Get("Authorization"), topic)
		if err != nil {
			p.Logger.Error("failed to create topic: " + err.Error())
		}

		var user models.UserData
		var note models.PushNotification

		// grab user info
		user, err = p.SearchService.SearchUserByUUID(uuid)
		if err != nil {
			p.Logger.Error("failed to fetch user data: ", err.Error())
			rw.WriteHeader(http.StatusCreated)
			rw.Write(fmt.Appendf(nil, `{"id": "%s"}`, postID))
			return
		}

		switch *req.Type {
		case "announcement":
			var memberUUIDs []string
			var org models.OrganizationDao

			// Find organization data
			filter := bson.M{"_id": post.GroupID}
			err = p.Database.OrgCol.FindOne(context.Background(), filter).Decode(&org)
			if err != nil {
				p.Logger.Error("failed to find organization: ", err.Error())
				rw.WriteHeader(http.StatusCreated)
				return
			}

			// Find organization members to add to the post notification topic
			members, err := findOrganizationMembers(&ctx, post.GroupID, p.Database)
			if err != nil {
				p.Logger.Error("Failed to find organization members. Error: ", err.Error())
			}
			for _, v := range *members {
				memberUUIDs = append(memberUUIDs, v.UserID)
			}
			err = p.Notification.ModifyTopic(r.Header.Get("Authorization"), org.ID.Hex(), models.NotificationTopicUpdateRequest{
				Action: "subscribe",
				Users:  memberUUIDs,
			})
			if err != nil {
				p.Logger.Error("Failed to add organization members to new announcement topic.")
			}

			// Find child clubs
			cur, err := p.Database.ClubCol.Find(context.TODO(), bson.M{"parent_id": post.GroupID})
			if err != nil {
				p.Logger.Error("No children found")
				rw.WriteHeader(http.StatusCreated)
				rw.Write(fmt.Appendf(nil, `{"id": "%s"}`, post.ID.Hex()))
			}

			// Send a notification to all of them
			for cur.Next(context.TODO()) {
				var club models.Club
				err := cur.Decode(&club)
				if err != nil {
					p.Logger.Error("failed to decode club: ", err.Error())
				}

				// send notification to club members
				clubID := club.ID.Hex()
				note = generateNewAnnouncementNotification(postID, *org.Name)
				p.Notification.SendNotification(r.Header.Get("Authorization"), models.NotificationPushRequest{
					Topic:        &clubID,
					Notification: note,
				})
			}
		default:
			// grab club info
			var club models.Club
			filter := bson.M{"_id": post.GroupID}
			err = p.Database.ClubCol.FindOne(context.Background(), filter).Decode(&club)
			if err != nil {
				p.Logger.Error("failed to fetch club data: ", err.Error())
				rw.WriteHeader(http.StatusCreated)
				rw.Write(fmt.Appendf(nil, `{"id": "%s"}`, postID))
			}

			clubID := club.ID.Hex()
			note = generateNewPostNotification(postID, club.Name, user.Username)
			p.Notification.SendNotification(r.Header.Get("Authorization"), models.NotificationPushRequest{
				Topic:        &clubID,
				Notification: note,
			})
		}

		rw.WriteHeader(http.StatusCreated)
		rw.Write(fmt.Appendf(nil, `{"id": "%s"}`, postID))
	}
}

/*
Update Post (POST)

  - Grab Post Id from path

  - Update post data

  - Grab request body

  - updated post data in database

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
		if req.IsSensitive != nil {
			change["is_sensitive"] = req.IsSensitive
		}

		// update post
		_, err = p.Database.PostsCollection.UpdateOne(context.TODO(), filter, update)
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
		_, err = p.Database.PostsCollection.DeleteOne(context.TODO(), filter)
		if err != nil {
			p.Logger.Debug("failed to delete post: ", err.Error())
			http.Error(rw, `{ "msg": "failed to delete post" }`, http.StatusInternalServerError)
			return
		}

		// delete notif topic
		err = p.Notification.DeleteTopic(r.Header.Get("Authorization"), id)
		if err != nil {
			p.Logger.Error("failed to delete topic: ", err.Error())
		}

		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{ "msg": "OK" }`))
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
		ctx, cancel := context.WithTimeout(r.Context(), time.Second*5)
		defer cancel()

		// grab uuid of the user who made this request
		uuid := r.Header.Get("UUID")

		// grab id from path
		vars := mux.Vars(r)
		id := vars["id"]
		if len(id) < 24 {
			http.Error(rw, `{ "msg": "bad post id" }`, http.StatusBadRequest)
			return
		}

		timestamp := primitive.NewDateTimeFromTime(time.Now())
		oid, _ := primitive.ObjectIDFromHex(id)
		like := models.ReactionDao{
			UserID:    &uuid,
			PostID:    &oid,
			CreatedAt: &timestamp,
		}

		rid, err := p.InsertReaction(ctx, &like, nil)
		if err != nil { // unexpected error
			p.Logger.Error("Failed to add reaction: ", err.Error())
			http.Error(rw, `{ "msg" : "something went wrong" }`, http.StatusInternalServerError)
			return
		}
		if rid == nil {
			p.Logger.Error("Failed to insert reaction. Inserted ID nil.")
			http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusOK)
		rw.Write(fmt.Appendf(nil, `{"id": "%s"}`, rid.Hex()))
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

		_, err := p.Database.PostsCollection.UpdateOne(context.TODO(), match, change)
		if err != nil {
			p.Logger.Error("failed to remove like: ", err.Error())
			http.Error(rw, `{ "msg" : "failed to remove like" }`, http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{ "msg": "OK" }`))
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

		ctx, cancel := context.WithTimeout(r.Context(), time.Second*5)
		defer cancel()

		// grab uuid of the user who made this request
		uuid := r.Header.Get("UUID")

		// grab id from path
		vars := mux.Vars(r)
		if len(vars["id"]) < 24 {
			http.Error(rw, `{ "msg": "no post id found in request." }`, http.StatusBadRequest)
			return
		}
		id := vars["id"]

		var req models.PostCommentDao
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			p.Logger.Error("failed to decode comment: ", err.Error())
			http.Error(rw, `{ "msg": "something went wrong" }`, http.StatusBadRequest)
			return
		}

		timestamp := primitive.NewDateTimeFromTime(time.Now())
		oid, _ := primitive.ObjectIDFromHex(id)
		req.UserID = &uuid
		req.PostID = &oid
		req.CreatedAt = &timestamp

		cid, err := p.InsertComment(ctx, &req, nil)
		if err != nil {
			p.Logger.Error("Failed to insert comment. Error: ", err.Error())
			http.Error(rw, `{ "msg": "something went wrong" }`, http.StatusInternalServerError)
			return
		}
		if cid == nil {
			p.Logger.Error("Failed to insert comment. Inserted ID nil.")
			http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusOK)
		rw.Write(fmt.Appendf(nil, `{ "id": "%s" }`, cid.Hex()))
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
func (p *Service) DeleteComment() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second*5)
		defer cancel()

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

		err := p.RemoveComment(ctx, bson.M{"_id": coid, "post_id": oid})
		if err != nil {
			p.Logger.Error("failed to remove comment. Error: ", err.Error())
			http.Error(rw, `{ "msg" : "something went wrong" }`, http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{ "msg": "OK" }`))

	}
}
