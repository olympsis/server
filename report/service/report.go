package service

import (
	"olympsis-server/database"
	"olympsis-server/report/orm"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type Service struct {
	Database     *database.Database
	Logger       *logrus.Logger
	Router       *mux.Router
	BugReport    *orm.BugReportORM
	PostReport   *orm.PostReportORM
	MemberReport *orm.MemberReportORM
	FieldReport  *orm.FieldReportORM
	EventReport  *orm.EventReportORM
}

func NewReportService(d *database.Database, l *logrus.Logger, r *mux.Router) *Service {
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
	eventORM := orm.EventReportORM{
		Database: d,
		Logger:   l,
	}
	return &Service{
		Database:     d,
		Logger:       l,
		Router:       r,
		BugReport:    &bugORM,
		PostReport:   &postORM,
		MemberReport: &memberORM,
		FieldReport:  &fieldORM,
		EventReport:  &eventORM,
	}
}
