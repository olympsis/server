package service

import (
	"olympsis-server/database"

	"github.com/gorilla/mux"
	"github.com/olympsis/notif"
	"github.com/olympsis/search"
	"github.com/sirupsen/logrus"
)

/*
Organization Service Struct
*/
type Service struct {
	Database *database.Database

	// logrus logger to Logger information about service and errors
	Logger *logrus.Logger

	// mux Router to complete http requests
	Router *mux.Router

	// notif service
	NotifService *notif.Service

	// search service
	SearchService *search.Service
}

/*
Create new field service struct
*/
func NewOrganizationService(l *logrus.Logger, r *mux.Router, d *database.Database, n *notif.Service, sh *search.Service) *Service {
	return &Service{Logger: l, Router: r, Database: d, NotifService: n, SearchService: sh}
}
