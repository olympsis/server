package service

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/certificate"
	"github.com/sideshow/apns2/payload"
	"github.com/sirupsen/logrus"
)

/*
Create new field service struct
*/
func NewNotificationService(l *logrus.Logger, r *mux.Router) *Service {
	return &Service{Logger: l, Router: r}
}

/*
Create apns client from p12 file
*/
func (p *Service) CreateNewClient() {
	cert, err := certificate.FromP12File("./pushnote/files/cert.p12", "")
	if err != nil {
		p.Logger.Fatal("token error:", err)
	}

	p.Client = apns2.NewClient(cert).Development()
}

func (p *Service) SendPushNotification() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		var req NotificationRequest

		// decode request
		json.NewDecoder(r.Body).Decode(&req)

		// loop through tokens and send request
		for index := range req.Tokens {
			p.PushNote(req.Title, req.Body, req.Tokens[index])
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
	}
}

func (p *Service) PushNote(t string, b string, tk string) {
	notification := &apns2.Notification{}
	notification.DeviceToken = tk
	notification.Topic = "com.coronislabs.Olympsis"
	notification.Payload = payload.NewPayload().AlertTitle(t).AlertBody(b).Badge(1)
	notification.Priority = 5

	res, err := p.Client.Push(notification)
	if err != nil {
		p.Logger.Error("There was an error", err)
		return
	}

	if res.Sent() {
		p.Logger.Debug("Sent:", res.ApnsID)
	} else {
		p.Logger.Debug("Not Sent: %v %v %v\n", res.StatusCode, res.ApnsID, res.Reason)
	}
}
