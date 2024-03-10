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

	/*
		POST REPORTS
	*/

	// get post reports
	e.Router.Handle("/report/{id}/posts",
		middleware.Chain(
			e.Service.ReadPostReports(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("GET")

	// create a post report
	e.Router.Handle("/report/{id}/posts",
		middleware.Chain(
			e.Service.CreatePostReport(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("POST")

	// change a post report
	e.Router.Handle("/report/posts/{id}",
		middleware.Chain(
			e.Service.UpdatePostReport(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("PUT")

	// delete a post report
	e.Router.Handle("/report/posts/{id}",
		middleware.Chain(
			e.Service.DeletePostReport(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("DELETE")

	/*
		MEMBER REPORTS
	*/

	// get post reports
	e.Router.Handle("/report/{id}/members",
		middleware.Chain(
			e.Service.ReadMemberReports(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("GET")

	// create a post report
	e.Router.Handle("/report/members",
		middleware.Chain(
			e.Service.CreateMemberReport(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("POST")

	// change a post report
	e.Router.Handle("/report/members/{id}",
		middleware.Chain(
			e.Service.UpdateMemberReport(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("PUT")

	// delete a post report
	e.Router.Handle("/report/members/{id}",
		middleware.Chain(
			e.Service.DeleteMemberReport(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("DELETE")

	/*
		FIELD REPORTS
	*/

	// get bug reports
	e.Router.Handle("/report/fields",
		middleware.Chain(
			e.Service.ReadFieldReports(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("GET")

	// create a bug report
	e.Router.Handle("/report/fields",
		middleware.Chain(
			e.Service.CreateFieldReport(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("POST")

	// change a bug report
	e.Router.Handle("/report/fields/{id}",
		middleware.Chain(
			e.Service.UpdateFieldReport(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("PUT")

	// delete a bug report
	e.Router.Handle("/report/fields/{id}",
		middleware.Chain(
			e.Service.DeleteFieldReport(),
			middleware.Logging(),
			middleware.UserMiddleware(),
		),
	).Methods("DELETE")
}
