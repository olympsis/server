package service

import (
	"github.com/gorilla/mux"
	"github.com/sideshow/apns2"
	"github.com/sirupsen/logrus"
)

type Service struct {
	Client *apns2.Client
	Logger *logrus.Logger
	Router *mux.Router
}

type NotificationRequest struct {
	Tokens   []string    `json:"tokens"`
	Title    string      `json:"title"`
	Body     string      `json:"body"`
	Topic    string      `json:"topic"`
	Priority int         `json:"priority"`
	Data     interface{} `json:"data"`
}
