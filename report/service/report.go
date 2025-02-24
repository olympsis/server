package service

import (
	"olympsis-server/database"
	"olympsis-server/report/orm"
	"olympsis-server/server"
	"olympsis-server/utils"

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
	Notification *utils.NotificationInterface
}

func NewReportService(i *server.ServerInterface) *Service {
	// Create ORMs
	bugORM := orm.BugReportORM{
		Database: i.Database,
		Logger:   i.Logger,
	}
	postORM := orm.PostReportORM{
		Database: i.Database,
		Logger:   i.Logger,
	}
	memberORM := orm.MemberReportORM{
		Database: i.Database,
		Logger:   i.Logger,
	}
	fieldORM := orm.FieldReportORM{
		Database: i.Database,
		Logger:   i.Logger,
	}
	eventORM := orm.EventReportORM{
		Database: i.Database,
		Logger:   i.Logger,
	}
	
	return &Service{
		Database:     i.Database,
		Logger:       i.Logger,
		Router:       i.Router,
		BugReport:    &bugORM,
		PostReport:   &postORM,
		MemberReport: &memberORM,
		FieldReport:  &fieldORM,
		EventReport:  &eventORM,
		Notification: i.Notification,
	}
}
