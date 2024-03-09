package service

import (
	"olympsis-server/database"
	"olympsis-server/report/orm"

	"github.com/gorilla/mux"
	"github.com/olympsis/notif"
	"github.com/sirupsen/logrus"
)

type Service struct {
	Database     *database.Database
	Logger       *logrus.Logger
	Router       *mux.Router
	NotifService *notif.Service
	BugReport    *orm.BugReportORM
}

func NewReportService(d *database.Database, l *logrus.Logger, r *mux.Router, n *notif.Service) *Service {
	bugORM := orm.BugReportORM{
		Database: d,
		Logger:   l,
	}
	return &Service{
		Database:     d,
		Logger:       l,
		Router:       r,
		NotifService: n,
		BugReport:    &bugORM,
	}
}
