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
	"olympsis-server/middleware"
	"olympsis-server/notifications"
	"olympsis-server/organization"
	"olympsis-server/post"
	redisDB "olympsis-server/redis"
	"olympsis-server/report"
	"olympsis-server/server"
	storageAPI "olympsis-server/storage"
	"olympsis-server/system"
	"olympsis-server/user"
	"olympsis-server/utils"
	"olympsis-server/utils/secrets"

	"olympsis-server/venue"
	"os"
	"os/signal"
	"syscall"
	"time"

	firebase "firebase.google.com/go"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stripe/stripe-go/v82"
	"google.golang.org/api/option"
)

func main() {
	// Set up logger
	l := logrus.New()

	// Set up Mux router
	r := mux.NewRouter()

	// Set default Content-Type to application/json for all routes
	r.Use(middleware.JSONGlobal)

	manager := secrets.New()

	// Set up server configuration
	config := utils.GetServerConfig(manager)

	// Set up database
	d := database.NewDatabase(l)
	d.EstablishConnection(manager, &config)

	// Set up redis
	rConfig := utils.GetRedisConfig(manager)
	cache := redisDB.NewClient(rConfig.Address, &rConfig.Username, &rConfig.Password, 0)
	cacheDB := redisDB.New(&cache, l)
	if err := cache.Ping(context.Background()).Err(); err != nil {
		l.Fatalf("Error setting up redis client. Error: %s", err.Error())
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

	// Create APNS client
	apnsClient, err := utils.CreateApns2Client(config.AppleKeyID, config.AppleTeamID, config.APNSFileURl)
	if err != nil {
		l.Fatalf("Failed to create Apns2 client. Error: %s", err.Error())
		os.Exit(1)
	}

	// Set up Notification Service
	notif := notifications.New(apnsClient, l, d)

	// Set up stripe API
	sc := stripe.NewClient(config.StripeToken)

	// Pass references to the server interface
	serverInterface := &server.ServerInterface{
		Logger:   l,
		Router:   r,
		Database: d, // db wrapper

		Stripe: sc,     // stripe
		Auth:   client, // firebase

		Cache: &cacheDB, // redis

		Notification: notif, // notifications
	}

	// Set up storage service first (other modules depend on it)
	storageModule := storageAPI.NewStorageAPI(serverInterface)
	if err := storageModule.Service.ConnectToClient(config.GCPCredentialsFilePath); err != nil {
		l.Fatalf("Failed to connect storage service to GCP: %s", err.Error())
		os.Exit(1)
	}
	serverInterface.Storage = storageModule.Service

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
	systemAPI.Ready(client)
	storageModule.Ready(client)

	// Apply compression universally
	r.Use(middleware.GzipMiddleware)

	// Handling raw notification requests
	r.Handle("/v1/notifications", middleware.Chain(
		serverInterface.Notification.HandleNotificationRequest(),
		middleware.Logging(),
		middleware.CORS(),
	)).Methods("POST", "OPTIONS")

	// Set up event polling
	// eventPolling := service.NewEventPollingService(d, l, &cacheDB, notif)
	// go eventPolling.Start(context.Background())

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

	l.Infof("Received Termination(%s), graceful shutdown \n", sig)

	tc, c := context.WithTimeout(context.Background(), 30*time.Second)
	defer c()

	s.Shutdown(tc)
}
