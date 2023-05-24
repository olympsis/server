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

type Service struct {
	Database *database.Database
	Logger   *logrus.Logger
	Router   *mux.Router
}

type Post struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	Owner     string             `json:"owner" bson:"owner"`
	ClubId    primitive.ObjectID `json:"clubId" bson:"clubId"`
	Body      string             `json:"body" bson:"body"`
	Images    []string           `json:"images" bson:"images"`
	Likes     []Like             `json:"likes" bson:"likes"`
	CreatedAt int64              `json:"createdAt" bson:"createdAt"`
}

type Owner struct {
	UUID     string `json:"uuid" bson:"uuid"`
	Username string `json:"username" bson:"username"`
	ImageURL string `json:"imageURL" bson:"imageURL"`
}

type Like struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	UUID      string             `json:"uuid" bson:"uuid"`
	CreatedAt int64              `json:"createdAt" bson:"createdAt"`
}

type Comment struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	UUID      string             `json:"uuid" bson:"uuid"`
	Text      string             `json:"text" bson:"text"`
	CreatedAt int64              `json:"createdAt" bson:"createdAt"`
}

type CommentThread struct {
	ID       primitive.ObjectID `json:"_d" bson:"_id"`
	Comments []Comment          `json:"comments" bson:"comments"`
}

type CommentThreadResponse struct {
	TotalComments int       `json:"totalComments"`
	Comments      []Comment `json:"comments"`
}

type PostsResponse struct {
	TotalPosts int    `json:"totalPosts"`
	Posts      []Post `json:"posts"`
}

/*
Lookup User
- contains identifiable user data that others can see
*/
type LookUpUser struct {
	FirstName string   `json:"firstName" bson:"firstName"`
	LastName  string   `json:"lastName" bson:"lastName"`
	Username  string   `json:"username" bson:"username"`
	ImageURL  string   `json:"imageURL" bson:"imageURL"`
	Clubs     []string `json:"clubs" bson:"clubs"`
	Sports    []string `json:"sports" bson:"sports"`
	Badges    []Badge  `json:"badges" bson:"badges"`
	Trophies  []Trophy `json:"trophies" bson:"trophies"`
	Friends   []Friend `json:"friends" bson:"friends"`
}

/*
Trophy
  - Trophy object
*/
type Trophy struct {
	ID          primitive.ObjectID `json:"id" bson:"_id"`
	Name        string             `json:"name" bson:"name"`
	Title       string             `json:"title" bson:"title"`
	ImageURL    string             `json:"imageURL" bson:"imageURL"`
	EventId     primitive.ObjectID `json:"eventId" bson:"eventId"`
	Description string             `json:"description" bson:"description"`
	AchievedAt  int64              `json:"achievedAt" bson:"achievedAt"`
}

/*
Badge
  - Badge object
*/
type Badge struct {
	ID          primitive.ObjectID `json:"_d" bson:"_id"`
	Name        string             `json:"name" bson:"name"`
	Title       string             `json:"title" bson:"title"`
	ImageURL    string             `json:"imageURL" bson:"imageURL"`
	EventId     primitive.ObjectID `json:"eventId" bson:"eventId"`
	Description string             `json:"description" bson:"description"`
	AchievedAt  int64              `json:"achievedAt" bson:"achievedAt"`
}

/*
Friend
  - Friend object
*/
type Friend struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	UUID      string             `json:"uuid" bson:"uuid"`
	CreatedAt int64              `json:"createdAt" bson:"createdAt"`
}

type NotificationRequest struct {
	Tokens []string `json:"tokens"`
	Title  string   `json:"title"`
	Body   string   `json:"body"`
	Topic  string   `json:"topic"`
}

type Club struct {
	ID          primitive.ObjectID `json:"id" bson:"_id"`
	Name        string             `json:"name" bson:"name"`
	Description string             `json:"description,omitempty" bson:"description,omitempty"`
	Sport       string             `json:"sport" bson:"sport"`
	City        string             `json:"city" bson:"city"`
	State       string             `json:"state" bson:"state"`
	Country     string             `json:"country" bson:"country"`
	ImageURL    string             `json:"imageURL,omitempty" bson:"imageURL,omitempty"`
	IsPrivate   bool               `json:"isPrivate" bson:"isPrivate"`
	Rules       []string           `json:"rules,omitempty" bson:"rules,omitempty"`
	CreatedAt   int64              `json:"createdAt" bson:"createdAt"`
}

/*
Create new Post service struct

  - Create and Returns a pointer to a new post service struct
*/
func NewPostService(l *logrus.Logger, r *mux.Router, d *database.Database) *Service {
	return &Service{Logger: l, Router: r, Database: d}
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

		var posts []Post
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
			var post Post
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

		resp := PostsResponse{
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
		var post Post
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
		bearerToken := r.Header.Get("Authorization")
		tokenSplit := strings.Split(bearerToken, "Bearer ")
		token := tokenSplit[1]
		uuid, _, _, err := p.ValidateAndParseJWTToken(token)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(`{ "msg": "unable to parse token." }`))
			return
		}

		var req Post

		// decode request
		err = json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
			return
		}

		timeStamp := time.Now().Unix()
		req.CreatedAt = timeStamp
		post := Post{
			ID:        primitive.NewObjectID(),
			Owner:     req.Owner,
			ClubId:    req.ClubId,
			Body:      req.Body,
			Images:    req.Images,
			Likes:     []Like{},
			CreatedAt: timeStamp,
		}

		thread := CommentThread{
			ID:       post.ID,
			Comments: []Comment{},
		}

		// create post in database
		_, err = p.Database.PostCol.InsertOne(context.TODO(), post)
		if err != nil {
			p.Logger.Error(err)
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
			return
		}

		// create comments thread in database
		_, err = p.Database.CommentsCol.InsertOne(context.TODO(), thread)
		if err != nil {
			p.Logger.Error(err)
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
			return
		}

		// grab user info for notifications
		user := p.FetchUser(*r, uuid)
		fN := user.FirstName + " " + user.LastName

		// grab club info for notifications
		var club Club
		filter := bson.M{"_id": post.ClubId}
		err = p.Database.ClubCol.FindOne(context.Background(), filter).Decode(&club)
		if err != nil {
			p.Logger.Error(err)
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": " ` + err.Error() + `" }`))
			return
		}

		// send notification to club members
		p.SendNotificationToTopic(*r, club.Name, fN+" posted in "+club.Name, club.ID.Hex())

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

		var req Post

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

		var req Like
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

/* COMMENTS */

func (p *Service) GetComments() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// grab id from path
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
		var thread CommentThread
		filter := bson.D{primitive.E{Key: "_id", Value: oid}}
		err := p.Database.CommentsCol.FindOne(context.TODO(), filter).Decode(&thread)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				rw.WriteHeader(http.StatusNotFound)
				rw.Write([]byte(`{ "msg": "post does not exist" }`))
				return
			}
		}

		var comments []Comment

		for i := range thread.Comments {
			comment := thread.Comments[i]
			comments = append(comments, comment)
		}

		if len(comments) == 0 {
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusNoContent)
			return
		}

		resp := CommentThreadResponse{
			TotalComments: len(comments),
			Comments:      comments,
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(resp)
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

		var req Comment
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

func (p *Service) FetchUser(r http.Request, user string) LookUpUser {
	bearerToken := r.Header.Get("Authorization")
	tokenSplit := strings.Split(bearerToken, "Bearer ")
	token := tokenSplit[1]
	client := &http.Client{}

	req, err := http.NewRequest("GET", "http://lookup.olympsis.internal/v1/lookup/"+user, nil)
	if err != nil {
		p.Logger.Error(err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		p.Logger.Error(err)
	}

	defer resp.Body.Close()

	var lookup LookUpUser
	err = json.NewDecoder(resp.Body).Decode(&lookup)
	if err != nil {
		p.Logger.Error(err)
	}
	return lookup
}

func (p *Service) SendNotificationToTopic(r http.Request, t string, b string, tpc string) (bool, error) {
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
		p.Logger.Error(err.Error())
		return false, err
	}

	req, err := http.NewRequest("POST", "http://pushnote.olympsis.internal/v1/pushnote/topic", bytes.NewBuffer(data))
	if err != nil {
		p.Logger.Error(err.Error())
		return false, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		p.Logger.Error(err.Error())
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
func (p *Service) ValidateAndParseJWTToken(tokenString string) (string, string, float64, error) {
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

/*
Middleware

  - Makes sure user is authenticated before taking requests

  - If there is no token or a bad token it returns the request with a unauthorized or forbidden error

Returns:

	Http handler
	- Passes the request to the next handler
*/
func (p *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		bearerToken := r.Header.Get("Authorization")
		tokenSplit := strings.Split(bearerToken, "Bearer ")

		if bearerToken == "" {
			p.Logger.Error("no auth token")
			http.Error(rw, "Unauthorized", http.StatusUnauthorized)
			return
		}

		token := tokenSplit[1]
		if token == "" {
			p.Logger.Error("no auth token")
			http.Error(rw, "Unauthorized", http.StatusUnauthorized)
			return
		}

		_, _, _, err := p.ValidateAndParseJWTToken(token)

		if err != nil {
			p.Logger.Error("bad auth token")
			http.Error(rw, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(rw, r)
	})
}

// Later we want to ping the db and if the db goes down or something is wrong with this service we want to restart it.
func (p *Service) Healthz() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte("ok"))
	}
}

func (p *Service) WhoAmi() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`
		{
			"version": "0.1.6",
			"service": "post"
		}
		`))
	}
}
