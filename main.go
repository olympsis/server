package main

import (
	"context"
	"net/http"
	"olympsis-server/announcement"
	"olympsis-server/auth"
	"olympsis-server/club"
	"olympsis-server/database"
	"olympsis-server/event"
	"olympsis-server/event/service"
	"olympsis-server/health"
	"olympsis-server/locales"
	mapsnapshots "olympsis-server/map-snapshots"
	"olympsis-server/notifications"
	"olympsis-server/organization"
	"olympsis-server/post"
	"olympsis-server/redis"
	"olympsis-server/report"
	"olympsis-server/server"
	"olympsis-server/system"
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
	"github.com/stripe/stripe-go/v82"
	"google.golang.org/api/option"
)

func main() {
	// Set up logger
	l := logrus.New()

	// Set up Mux router
	r := mux.NewRouter()

	// Set up server configuration
	config := utils.GetServerConfig()

	// Set up database
	d := database.NewDatabase(l)
	d.EstablishConnection(&config)

	// Set up redis

	cache := redis.NewRedisClient("", "", 0)
	cacheDB := redis.NewRedisDatabase(&cache, l)
	if err := cache.Ping(context.Background()); err != nil {
		l.Fatalf("Error setting up redis client. Error: %s", err.Err().Error())
		os.Exit(1)
	}

	// Set up Firebase authentication
	opt := option.WithCredentialsFile(config.FirebaseFilePath)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		l.Fatalf("Error starting Firebase app: %s\n", err)
		os.Exit(1)
	}
	client, err := app.Auth(context.TODO())
	if err != nil {
		l.Fatalf("Error getting Firebase Auth client: %v\n", err)
		os.Exit(1)
	}

	// Set up Notification Service
	notif := notifications.NewNotificationService(l)

	// Set up search service
	sh := search.NewSearchService(l, d.AuthCol, d.UserCol)

	// Set up stripe API
	sc := stripe.NewClient(config.StripeToken)

	// Pass references to the server interface
	serverInterface := &server.ServerInterface{
		Logger:   l,
		Router:   r,
		Database: d, // db wrapper

		Stripe: sc,     // stripe
		Auth:   client, // firebase
		Search: sh,     // search

		Notification:        utils.NewNotificationInterface(config.NotifServiceURL, l),
		NotificationService: notif,
	}

	// Set up API
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
	systemAPI := system.NewConfigApi(serverInterface)

	// Initialize APIs
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
	systemAPI.Ready()

	// Set up event polling
	eventPolling := service.NewEventPollingService(d, l, &cacheDB, notif)
	go func() {
		eventPolling.Start(context.Background())
	}()

	// Set up server configuration
	s := &http.Server{
		Addr:         `:` + config.Port,
		Handler:      r,
		IdleTimeout:  60 * time.Second,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Start server
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
