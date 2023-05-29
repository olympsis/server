package post

import (
	"olympsis-server/database"
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
	p.Router.Handle("/posts", p.Service.GetPosts()).Methods("GET")
	p.Router.Handle("/posts/{id}", p.Service.GetPost()).Methods("GET")
	p.Router.Handle("/posts", p.Service.CreatePost()).Methods("POST")
	p.Router.Handle("/posts/{id}", p.Service.UpdatePost()).Methods("PUT")
	p.Router.Handle("/posts/{id}", p.Service.DeletePost()).Methods("DELETE")

	p.Router.Handle("/posts/{id}/likes", p.Service.AddLike()).Methods("POST")
	p.Router.Handle("/posts/{id}/likes/{likeId}", p.Service.RemoveLike()).Methods("DELETE")

	p.Router.Handle("/posts/{id}/comments", p.Service.AddComment()).Methods("POST")
	p.Router.Handle("/posts/{id}/comments/{commentId}", p.Service.RemoveComment()).Methods("DELETE")
}
