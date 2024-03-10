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
	PostReport   *orm.PostReportORM
	MemberReport *orm.MemberReportORM
	FieldReport  *orm.FieldReportORM
}

func NewReportService(d *database.Database, l *logrus.Logger, r *mux.Router, n *notif.Service) *Service {
	bugORM := orm.BugReportORM{
		Database: d,
		Logger:   l,
	}
	postORM := orm.PostReportORM{
		Database: d,
		Logger:   l,
	}
	memberORM := orm.MemberReportORM{
		Database: d,
		Logger:   l,
	}
	fieldORM := orm.FieldReportORM{
		Database: d,
		Logger:   l,
	}
	return &Service{
		Database:     d,
		Logger:       l,
		Router:       r,
		NotifService: n,
		BugReport:    &bugORM,
		PostReport:   &postORM,
		MemberReport: &memberORM,
		FieldReport:  &fieldORM,
	}
}
