package report

import (
	"olympsis-server/middleware"
	"olympsis-server/report/service"
	"olympsis-server/server"

	"firebase.google.com/go/auth"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type ReportAPI struct {
	Logger  *logrus.Logger   // logger for logging errors
	Router  *mux.Router      // router for handling requests
	Service *service.Service // service for handing requests to
}

func NewReportAPI(i *server.ServerInterface) *ReportAPI {
	return &ReportAPI{
		Logger:  i.Logger,
		Router:  i.Router,
		Service: service.NewReportService(i),
	}
}

func (e *ReportAPI) Setup(firebase *auth.Client) {

	/*
		BUG REPORTS
	*/

	// get bug reports
	e.Router.Handle("/v1/report/bugs",
		middleware.Chain(
			e.Service.ReadBugReports(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("GET", "OPTIONS")

	// create a bug report
	e.Router.Handle("/v1/report/bugs",
		middleware.Chain(
			e.Service.CreateBugReport(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("POST", "OPTIONS")

	// change a bug report
	e.Router.Handle("/v1/report/bugs/{id}",
		middleware.Chain(
			e.Service.UpdateBugReport(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("PUT", "OPTIONS")

	// delete a bug report
	e.Router.Handle("/v1/report/bugs/{id}",
		middleware.Chain(
			e.Service.DeleteBugReport(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("DELETE", "OPTIONS")

	/*
		POST REPORTS
	*/

	// get post reports
	e.Router.Handle("/v1/report/posts",
		middleware.Chain(
			e.Service.ReadPostReports(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("GET", "OPTIONS")

	// create a post report
	e.Router.Handle("/v1/report/posts",
		middleware.Chain(
			e.Service.CreatePostReport(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("POST", "OPTIONS")

	// change a post report
	e.Router.Handle("/v1/report/posts/{id}",
		middleware.Chain(
			e.Service.UpdatePostReport(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("PUT", "OPTIONS")

	// delete a post report
	e.Router.Handle("/v1/report/posts/{id}",
		middleware.Chain(
			e.Service.DeletePostReport(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("DELETE", "OPTIONS")

	/*
		MEMBER REPORTS
	*/

	// get post reports
	e.Router.Handle("/v1/report/members",
		middleware.Chain(
			e.Service.ReadMemberReports(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("GET", "OPTIONS")

	// create a post report
	e.Router.Handle("/v1/report/members",
		middleware.Chain(
			e.Service.CreateMemberReport(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("POST", "OPTIONS")

	// change a post report
	e.Router.Handle("/v1/report/members/{id}",
		middleware.Chain(
			e.Service.UpdateMemberReport(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("PUT", "OPTIONS")

	// delete a post report
	e.Router.Handle("/v1/report/members/{id}",
		middleware.Chain(
			e.Service.DeleteMemberReport(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("DELETE", "OPTIONS")

	/*
		FIELD REPORTS
	*/

	// get bug reports
	e.Router.Handle("/v1/report/fields",
		middleware.Chain(
			e.Service.ReadFieldReports(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("GET", "OPTIONS")

	// create a bug report
	e.Router.Handle("/v1/report/fields",
		middleware.Chain(
			e.Service.CreateFieldReport(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("POST", "OPTIONS")

	// change a bug report
	e.Router.Handle("/v1/report/fields/{id}",
		middleware.Chain(
			e.Service.UpdateFieldReport(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("PUT", "OPTIONS")

	// delete a bug report
	e.Router.Handle("/v1/report/fields/{id}",
		middleware.Chain(
			e.Service.DeleteFieldReport(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("DELETE", "OPTIONS")

	/*
		EVENT REPORTS
	*/

	// get bug reports
	e.Router.Handle("/v1/report/events",
		middleware.Chain(
			e.Service.ReadEventReports(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("GET", "OPTIONS")

	// create a bug report
	e.Router.Handle("/v1/report/events",
		middleware.Chain(
			e.Service.CreateEventReport(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("POST", "OPTIONS")

	// change a bug report
	e.Router.Handle("/v1/report/events/{id}",
		middleware.Chain(
			e.Service.UpdateEventReport(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("PUT", "OPTIONS")

	// delete a bug report
	e.Router.Handle("/v1/report/events/{id}",
		middleware.Chain(
			e.Service.DeleteEventReport(),
			middleware.Logging(),
			middleware.UserMiddleware(firebase),
			middleware.CORS(),
		),
	).Methods("DELETE", "OPTIONS")
}
