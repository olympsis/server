package report

import (
	"olympsis-server/database"
	"olympsis-server/middleware"
	"olympsis-server/report/service"

	"github.com/gorilla/mux"
	"github.com/olympsis/notif"
	"github.com/sirupsen/logrus"
)

type ReportAPI struct {
	Logger  *logrus.Logger   // logger for logging errors
	Router  *mux.Router      // router for handling requests
	Service *service.Service // service for handing requests to
}

func NewReportAPI(l *logrus.Logger, r *mux.Router, d *database.Database, n *notif.Service) *ReportAPI {
	return &ReportAPI{
		Logger:  l,
		Router:  r,
		Service: service.NewReportService(d, l, r, n),
	}
}

func (e *ReportAPI) Setup() {

	/*
		BUG REPORTS
	*/

	// get bug reports
	e.Router.Handle("/report/bugs",
		middleware.Chain(
			e.Service.ReadBugReports(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("GET")

	// create a bug report
	e.Router.Handle("/report/bugs",
		middleware.Chain(
			e.Service.CreateBugReport(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("POST")

	// change a bug report
	e.Router.Handle("/report/bugs/{id}",
		middleware.Chain(
			e.Service.UpdateBugReport(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("PUT")

	// delete a bug report
	e.Router.Handle("/report/bugs/{id}",
		middleware.Chain(
			e.Service.DeleteBugReport(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("DELETE")
}
