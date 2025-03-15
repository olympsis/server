package main

import (
	"context"
	"net/http"
	"olympsis-server/announcement"
	"olympsis-server/auth"
	"olympsis-server/club"
	"olympsis-server/database"
	"olympsis-server/event"
	"olympsis-server/health"
	"olympsis-server/locales"
	mapsnapshots "olympsis-server/map-snapshots"
	"olympsis-server/organization"
	"olympsis-server/post"
	"olympsis-server/report"
	"olympsis-server/server"
	"olympsis-server/user"
	"olympsis-server/utils"

	"olympsis-server/venue"
	"os"
	"os/signal"
	"syscall"
	"time"

	firebase "firebase.google.com/go"
	"github.com/gorilla/mux"
	"github.com/olympsis/search"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
)

func main() {
	// logger
	l := logrus.New()

	// mux router
	r := mux.NewRouter()

	// Server configuration
	config := utils.GetServerConfig()

	// database
	d := database.NewDatabase(l)
	d.EstablishConnection(&config)

	opt := option.WithCredentialsFile(config.FirebaseFilePath)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		l.Fatalf("error starting firebase app: %s\n", err)
		os.Exit(1)
	}
	client, err := app.Auth(context.TODO())
	if err != nil {
		l.Fatalf("error getting Auth client: %v\n", err)
		os.Exit(1)
	}

	// search service
	sh := search.NewSearchService(l, d.AuthCol, d.UserCol)

	// Pass references to apis
	serverInterface := &server.ServerInterface{
		Logger:   l,
		Router:   r,
		Database: d,

		Auth:   client,
		Search: sh,

		Notification: utils.NewNotificationInterface(config.NotifServiceURL, l),
	}

	announceAPI := announcement.NewAnnouncementAPI(serverInterface)
	authAPI := auth.NewAuthAPI(serverInterface)
	userAPI := user.NewUserAPI(serverInterface)
	fieldAPI := venue.NewVenueAPI(serverInterface)
	clubAPI := club.NewClubAPI(serverInterface)
	postAPI := post.NewPostAPI(serverInterface)
	eventAPI := event.NewEventAPI(serverInterface)
	orgAPI := organization.NewOrganizationAPI(serverInterface)
	reportAPI := report.NewReportAPI(serverInterface)
	localeAPI := locales.NewLocaleAPI(serverInterface)
	healthAPI := health.NewHealthAPI(serverInterface)
	snapShotAPI := mapsnapshots.NewMapSnapshotAPI(serverInterface, &config)

	announceAPI.Ready(client)
	authAPI.Ready(client)
	userAPI.Ready(client)
	fieldAPI.Ready()
	clubAPI.Ready(client)
	postAPI.Ready(client)
	eventAPI.Ready(client)
	orgAPI.Ready(client)
	reportAPI.Setup(client)
	localeAPI.Ready()
	healthAPI.Ready()
	snapShotAPI.Ready()

	// server config
	s := &http.Server{
		Addr:         `:` + config.Port,
		Handler:      r,
		IdleTimeout:  60 * time.Second,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// start server
	go func() {
		l.Info(`Starting olympsis server at...` + config.Port)

		switch config.Http {
		case "SECURE":
			err := s.ListenAndServeTLS(config.CertFilePath, config.KeyFilePath)
			if err != nil {
				l.Info("Error starting server: ", err)
				os.Exit(1)
			}
		default:
			err := s.ListenAndServe()
			if err != nil {
				l.Info("Error starting server: ", err)
				os.Exit(1)
			}
		}
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigs

	l.Printf("Received Termination(%s), graceful shutdown \n", sig)

	tc, c := context.WithTimeout(context.Background(), 30*time.Second)
	defer c()

	s.Shutdown(tc)
}
