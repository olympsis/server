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
	p.Service.CreateNewClient()
	p.Router.Handle("/v1/pushnote/device", p.Service.SendPushNotification()).Methods("POST")
}

func (p *PushNoteAPI) GetService() *service.Service {
	return p.Service
}
