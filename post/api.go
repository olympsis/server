package post

import (
	"olympsis-server/database"
	"olympsis-server/middleware"
	"olympsis-server/post/service"
	notif "olympsis-server/pushnote/service"
	search "olympsis-server/search"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type PostAPI struct {
	Logger  *logrus.Logger
	Router  *mux.Router
	Service *service.Service
}

func NewPostAPI(l *logrus.Logger, r *mux.Router, d *database.Database, n *notif.Service, sh *search.Service) *PostAPI {
	return &PostAPI{Logger: l, Router: r, Service: service.NewPostService(l, r, d, n, sh)}
}

func (p *PostAPI) Ready() {
	/*
		POSTS
	*/

	// get posts
	p.Router.Handle("/posts", middleware.Chain(
		p.Service.GetPosts(),
		middleware.Logging(),
		middleware.UserMiddleware(),
	)).Methods("GET")

	// get a post
	p.Router.Handle("/posts/{id}", middleware.Chain(
		p.Service.GetPost(),
		middleware.Logging(),
		middleware.UserMiddleware(),
	)).Methods("GET")

	// create a post
	p.Router.Handle("/posts", middleware.Chain(
		p.Service.CreatePost(),
		middleware.Logging(),
		middleware.UserMiddleware(),
	)).Methods("POST")

	// udpdate a post
	p.Router.Handle("/posts/{id}", middleware.Chain(
		p.Service.UpdatePost(),
		middleware.Logging(),
		middleware.UserMiddleware(),
	)).Methods("PUT")

	// delete a post
	p.Router.Handle("/posts/{id}", middleware.Chain(
		p.Service.DeletePost(),
		middleware.Logging(),
		middleware.UserMiddleware(),
	)).Methods("DELETE")

	/*
		POST LIKES
	*/

	// add a like
	p.Router.Handle("/posts/{id}/likes",
		middleware.Chain(
			p.Service.AddLike(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("POST")

	// remove a like
	p.Router.Handle("/posts/{id}/likes/{likeId}",
		middleware.Chain(
			p.Service.RemoveLike(),
			middleware.Logging(),
			middleware.UserMiddleware(),
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
			middleware.UserMiddleware(),
		),
	).Methods("POST")

	// remove a comment
	p.Router.Handle("/posts/{id}/comments/{commentId}",
		middleware.Chain(
			p.Service.RemoveComment(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("DELETE")

}
