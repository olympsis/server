package main

import (
	"context"
	"net/http"
	_ "net/http/pprof" // registers pprof handlers on DefaultServeMux (served on the localhost-only listener in main)
	"olympsis-server/announcement"
	"olympsis-server/auth"
	"olympsis-server/bus"
	// "olympsis-server/club" // DISABLED 2026-06-15: clubs turned off for now
	"olympsis-server/database"
	"olympsis-server/event"
	eventService "olympsis-server/event/service"
	"olympsis-server/grpcapi"
	"olympsis-server/health"
	"olympsis-server/locales"
	mapsnapshots "olympsis-server/map-snapshots"
	"olympsis-server/middleware"
	"olympsis-server/notifications"
	"olympsis-server/organization"
	"olympsis-server/post"
	"olympsis-server/push"
	redisDB "olympsis-server/redis"
	"olympsis-server/report"
	"olympsis-server/server"
	storageAPI "olympsis-server/storage"
	"olympsis-server/system"
	"olympsis-server/user"
	"olympsis-server/utils"
	"olympsis-server/utils/secrets"

	"net"
	"olympsis-server/venue"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stripe/stripe-go/v82"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
)

func main() {
	// Set up logger
	l := logrus.New()

	// Constrained-hardware runtime tuning. An explicit GOMEMLIMIT env var (e.g.
	// from the PM2 ecosystem env) takes precedence; this only sets a conservative
	// default soft heap limit when none is set, so the binary self-limits on the
	// shared 16 GB Mac mini. 1.5 GiB leaves headroom for MongoDB's cache — raise
	// GOMEMLIMIT if pprof shows the server genuinely needs more.
	if os.Getenv("GOMEMLIMIT") == "" {
		debug.SetMemoryLimit(1536 << 20) // 1.5 GiB
	}

	// Expose pprof on a localhost-only listener for memory/goroutine profiling
	// (to confirm the leak fixes hold over time). It is NOT registered on the
	// main router, so it is never reachable through KrakenD — only from the box
	// itself. Tunnel in with `ssh -L 6060:localhost:6060 <host>`, then open
	// http://localhost:6060/debug/pprof/. Override the bind address with
	// PPROF_ADDR, or set PPROF_ADDR=off to disable. No write timeout on purpose:
	// CPU/trace profiles need to stream for their full duration.
	pprofAddr := os.Getenv("PPROF_ADDR")
	if pprofAddr == "" {
		pprofAddr = "localhost:6060"
	}
	if pprofAddr != "off" {
		go func() {
			l.Infof("[Perf] Starting pprof listener on %s", pprofAddr)
			if err := http.ListenAndServe(pprofAddr, nil); err != nil {
				l.Errorf("[Perf] pprof listener stopped: %s", err.Error())
			}
		}()
	}

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
		l.Fatalf("[Database] Error setting up redis client. Error: %s", err.Error())
		os.Exit(1)
	}

	// Set up Firebase authentication
	opt := option.WithCredentialsFile(config.FirebaseFilePath)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		l.Fatalf("[Core] Error starting Firebase app: %s\n", err)
		os.Exit(1)
	}
	client, err := app.Auth(context.TODO())
	if err != nil {
		l.Fatalf("[Core] Error getting Firebase Auth client: %v\n", err)
		os.Exit(1)
	}

	// Firebase Cloud Messaging client (Android push). Reuses the same Firebase
	// app + service-account credentials as Auth, so no extra OAuth2 plumbing.
	var msgClient *messaging.Client
	if msgClient, err = app.Messaging(context.TODO()); err != nil {
		l.Fatalf("[Core] Error getting Firebase Messaging client: %v\n", err)
		os.Exit(1)
	}

	// Create APNS client
	apnsClient, err := utils.CreateApns2Client(config.AppleKeyID, config.AppleTeamID, config.APNSFileURl)
	if err != nil {
		l.Fatalf("[Carousel] Failed to create Apns2 client. Error: %s", err.Error())
		os.Exit(1)
	}

	// Set up Notification Service (legacy rich notifications: events + clubs)
	notif := notifications.New(apnsClient, l, d)

	// Set up Push Service (loc_key event notes: reminder, participant, comment),
	// delivering to iOS via APNs and Android via FCM v1.
	pushSvc := push.New(apnsClient, msgClient, l, d)

	// Set up the RabbitMQ event-bus publisher. The server is a pure publisher:
	// event.created/team.created feed invite-service, rsvp.created/comment.created
	// feed notif-service. Its lifecycle gets its own cancellable context so the
	// background reconnect goroutine stops cleanly on shutdown.
	//
	// A connect failure is intentionally NOT fatal — publishing is best effort,
	// and the publisher retries in the background — so the rest of the API stays
	// up even when the broker is down.
	busCtx, busCancel := context.WithCancel(context.Background())
	rmqConfig := utils.GetRabbitMQConfig()
	publisher := bus.New(l, rmqConfig.URL, rmqConfig.Exchange)
	if err := publisher.Connect(busCtx); err != nil {
		l.Errorf("[Bus] Failed to connect to RabbitMQ: %s", err.Error())
	}

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

		Notification: notif,   // legacy rich notifications
		Push:         pushSvc, // loc_key event push notifications

		Bus: publisher, // RabbitMQ domain-event publisher
	}

	// Set up storage service first (other modules depend on it)
	storageModule := storageAPI.NewStorageAPI(serverInterface)
	if err := storageModule.Service.ConnectToClient(config.GCPCredentialsFilePath); err != nil {
		l.Fatalf("[Storage] Failed to connect storage service to GCP: %s", err.Error())
		os.Exit(1)
	}
	serverInterface.Storage = storageModule.Service

	// Set up API
	announceAPI := announcement.NewAnnouncementAPI(serverInterface)
	authAPI := auth.NewAuthAPI(serverInterface)
	userAPI := user.NewUserAPI(serverInterface)
	fieldAPI := venue.NewVenueAPI(serverInterface)
	// clubAPI := club.NewClubAPI(serverInterface) // DISABLED 2026-06-15: clubs turned off for now
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
	// clubAPI.Ready(client) // DISABLED 2026-06-15: clubs turned off for now
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
	)).Methods("POST", "OPTIONS")

	// Set up event polling. Tie its lifecycle to a cancellable context so the
	// 5-minute ticker goroutine stops cleanly on shutdown instead of leaking.
	pollCtx, pollCancel := context.WithCancel(context.Background())
	eventPolling := eventService.NewEventPollingService(d, l, &cacheDB, pushSvc)
	go eventPolling.Start(pollCtx)
	l.Info("[E-Polling] Initialized...")

	// Internal gRPC server (EventTeamService). invite-service calls AddTeamMember
	// on it when a user accepts a TEAM invite, so the main server — the only
	// writer of eventTeams — adds them to the roster. Not exposed publicly.
	//
	// A listen failure is intentionally non-fatal (mirroring the bus publisher):
	// the rest of the API keeps serving even if the gRPC port is unavailable. Set
	// GRPC_PORT=off to disable the listener entirely.
	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50051"
	}
	var grpcServer *grpc.Server
	if grpcPort != "off" {
		grpcServer = grpcapi.NewGRPCServer(eventAPI.Service, l)
		grpcLis, err := net.Listen("tcp", ":"+grpcPort)
		if err != nil {
			l.Errorf("[gRPC] failed to listen on :%s: %s", grpcPort, err.Error())
			grpcServer = nil
		} else {
			go func() {
				l.Infof("[gRPC] EventTeamService listening on :%s", grpcPort)
				if serveErr := grpcServer.Serve(grpcLis); serveErr != nil {
					l.Errorf("[gRPC] server stopped: %s", serveErr.Error())
				}
			}()
		}
	}

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
		l.Info(`[Core] Starting olympsis server at...` + config.Port)

		switch config.Http {
		case "SECURE":
			err := s.ListenAndServeTLS(config.CertFilePath, config.KeyFilePath)
			if err != nil {
				l.Info("[Core] Error starting server: ", err)
				os.Exit(1)
			}
		default:
			err := s.ListenAndServe()
			if err != nil {
				l.Info("[Core] Error starting server: ", err)
				os.Exit(1)
			}
		}
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigs

	l.Infof("[Core] Received Termination(%s), graceful shutdown \n", sig)

	// Stop the event polling ticker goroutine before tearing down dependencies.
	pollCancel()

	// Stop accepting new gRPC calls and let in-flight AddTeamMember calls finish.
	if grpcServer != nil {
		grpcServer.GracefulStop()
	}

	tc, c := context.WithTimeout(context.Background(), 30*time.Second)
	defer c()

	if err := s.Shutdown(tc); err != nil {
		l.Errorf("[Core] Server shutdown error: %s", err.Error())
	}

	// Stop the bus reconnect goroutine and close the broker connection. Safe
	// here: the HTTP server has drained, so no handler is mid-publish.
	busCancel()
	if err := publisher.Close(); err != nil {
		l.Errorf("[Bus] shutdown error: %s", err.Error())
	}

	// Drain the notification carousel and the push dispatcher before closing
	// Mongo, since processing queued jobs still needs the database connection.
	notif.Stop()
	pushSvc.Stop()

	// Close the MongoDB connection pool.
	if err := d.Client.Disconnect(tc); err != nil {
		l.Errorf("[Database] disconnect error: %s", err.Error())
	}
}
