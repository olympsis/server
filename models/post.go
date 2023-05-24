package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Post struct {
	ID        primitive.ObjectID `json:"id,omitempty" bson:"_id"`
	ClubId    primitive.ObjectID `json:"club_id" bson:"club_id"`
	Poster    string             `json:"poster" bson:"poster"`
	Title     string             `json:"title" bson:"title"`
	Body      string             `json:"body" bson:"body"`
	EventId   primitive.ObjectID `json:"event_id,omitempty" bson:"event_id,omitempty"`
	Images    []string           `json:"images" bson:"images"`
	Likes     []Like             `json:"likes" bson:"likes"`
	Comments  []Comment          `json:"comments" bson:"comments"`
	CreatedAt int64              `json:"created_at" bson:"created_at"`
}

type Comment struct {
	ID        primitive.ObjectID `json:"id,omitempty" bson:"_id"`
	UUID      string             `json:"uuid" bson:"uuid"`
	Text      string             `json:"text" bson:"text"`
	CreatedAt int64              `json:"created_at" bson:"created_at"`
}

type PostsResponse struct {
	TotalPosts int    `json:"total_posts"`
	Posts      []Post `json:"posts"`
}
