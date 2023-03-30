package pushnote

import (
	"olympsis-server/pushnote/service"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type PushNoteAPI struct {
	Logger  *logrus.Logger
	Router  *mux.Router
	Service *service.Service
}

func NewPushNoteAPI(l *logrus.Logger, r *mux.Router) *PushNoteAPI {
	return &PushNoteAPI{Logger: l, Router: r, Service: service.NewNotificationService(l, r)}
}

func (p *PushNoteAPI) Ready() {
	p.Router.Handle("/v1/pushnote/device", p.Service.SendPushNotification()).Methods("POST")
	p.Router.Handle("/v1/pushnote/topic", p.Service.SendPushNotificationToTopic()).Methods("POST")
	p.Router.Handle("/v1/pushnote/topic", p.Service.SubscribeToTopic()).Methods("PUT")
	p.Router.Handle("/v1/pushnote/topic", p.Service.UnSubscribeFromTopic()).Methods("DELETE")
}
