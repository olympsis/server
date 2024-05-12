package post

import (
	"olympsis-server/database"
	"olympsis-server/middleware"
	"olympsis-server/post/service"

	"firebase.google.com/go/auth"
	"github.com/gorilla/mux"
	"github.com/olympsis/search"
	"github.com/sirupsen/logrus"
)

type PostAPI struct {
	Logger  *logrus.Logger
	Router  *mux.Router
	Service *service.Service
}

func NewPostAPI(l *logrus.Logger, r *mux.Router, d *database.Database, sh *search.Service) *PostAPI {
	return &PostAPI{Logger: l, Router: r, Service: service.NewPostService(l, r, d, sh)}
}

func (p *PostAPI) Ready(firebase *auth.Client) {
	/*
		POSTS
	*/

	// get posts
	p.Router.Handle("/posts", middleware.Chain(
		p.Service.GetPosts(),
		middleware.Logging(),
		middleware.UserMiddleware(firebase),
	)).Methods("GET")

	// get a post
	p.Router.Handle("/posts/{id}", middleware.Chain(
		p.Service.GetPost(),
		middleware.Logging(),
		middleware.UserMiddleware(firebase),
	)).Methods("GET")

	// create a post
	p.Router.Handle("/posts", middleware.Chain(
		p.Service.CreatePost(),
		middleware.Logging(),
		middleware.UserMiddleware(firebase),
	)).Methods("POST")

	// update a post
	p.Router.Handle("/posts/{id}", middleware.Chain(
		p.Service.ModifyPost(),
		middleware.Logging(),
		middleware.UserMiddleware(firebase),
	)).Methods("PUT")

	// delete a post
	p.Router.Handle("/posts/{id}", middleware.Chain(
		p.Service.DeletePost(),
		middleware.Logging(),
		middleware.UserMiddleware(firebase),
	)).Methods("DELETE")

	/*
		POST LIKES
	*/

	// add a like
	p.Router.Handle("/posts/{id}/likes",
		middleware.Chain(
			p.Service.AddLike(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("POST")

	// remove a like
	p.Router.Handle("/posts/{id}/likes/{likeID}",
		middleware.Chain(
			p.Service.RemoveLike(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("DELETE")

	/*
		POST COMMENTS
	*/

	// add a comment
	p.Router.Handle("/posts/{id}/comments",
		middleware.Chain(
			p.Service.AddComment(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("POST")

	// remove a comment
	p.Router.Handle("/posts/{id}/comments/{commentID}",
		middleware.Chain(
			p.Service.RemoveComment(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
		),
	).Methods("DELETE")

}
