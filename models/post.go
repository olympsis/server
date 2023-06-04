package models

import "go.mongodb.org/mongo-driver/bson/primitive"

/*
Post
  - Post objects for feed
*/
type Post struct {
	ID        primitive.ObjectID `json:"id,omitempty" bson:"_id"`
	ClubID    primitive.ObjectID `json:"club_id" bson:"club_id"`
	Poster    string             `json:"poster,omitempty" bson:"poster,omitempty"`
	Body      string             `json:"body" bson:"body"`
	EventID   primitive.ObjectID `json:"event_id,omitempty" bson:"event_id,omitempty"`
	Data      *PostData          `json:"data,omitempty" bson:"data,omitempty"`
	Images    []string           `json:"images" bson:"images"`
	Likes     []Like             `json:"likes,omitempty" bson:"likes,omitempty"`
	Comments  []Comment          `json:"comments,omitempty" bson:"comments,omitempty"`
	CreatedAt int64              `json:"created_at" bson:"created_at"`
}

type PostData struct {
	Poster *UserData `json:"poster,omitempty"`
	Event  *Event    `json:"event,omitempty"`
	User   *UserData `json:"user,omitempty"`
}

/*
Comment
  - Comments on post
*/
type Comment struct {
	ID        primitive.ObjectID `json:"id,omitempty" bson:"_id"`
	UUID      string             `json:"uuid" bson:"uuid"`
	Data      *UserData          `json:"data,omitempty" bson:"data,omitempty"`
	Text      string             `json:"text" bson:"text"`
	CreatedAt int64              `json:"created_at" bson:"created_at"`
}

/*
Posts Response
  - array of posts
*/
type PostsResponse struct {
	TotalPosts int    `json:"total_posts"`
	Posts      []Post `json:"posts"`
}
