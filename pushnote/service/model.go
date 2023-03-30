package service

import (
	firebase "firebase.google.com/go/v4"
	"github.com/gorilla/mux"
	"github.com/sideshow/apns2"
	"github.com/sirupsen/logrus"
)

type Service struct {
	client *apns2.Client // custom apns for later
	fApp   *firebase.App
	Logger *logrus.Logger
	Router *mux.Router
}

type Notififcation struct {
	ID          string      `json:"id"`
	DeviceToken string      `json:"deviceToken"`
	Topic       string      `json:"topic"`
	Priority    int         `json:"priority"`
	Payload     interface{} `json:"payoad"`
	Expiration  int64       `json:"expiration"`
	PushType    string      `json:"pushType"`
}

type NotificationRequest struct {
	Tokens []string `json:"tokens"`
	Title  string   `json:"title"`
	Body   string   `json:"body"`
	Topic  string   `json:"topic"`
}
