package main

import (
	"context"
	"net/http"
	"olympsis-server/auth"
	"olympsis-server/club"
	"olympsis-server/database"
	"olympsis-server/event"
	"olympsis-server/field"
	"olympsis-server/organization"
	"olympsis-server/post"
	"olympsis-server/report"
	"olympsis-server/user"
	"os"
	"os/signal"
	"syscall"
	"time"

	firebase "firebase.google.com/go/v4"

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

	// database
	d := database.NewDatabase(l)
	d.EstablishConnection()

	opt := option.WithCredentialsFile("./files/firebase-credentials.json")
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

	authAPI := auth.NewAuthAPI(l, r, d, client)
	userAPI := user.NewUserAPI(l, r, d)
	fieldAPI := field.NewFieldAPI(l, r, d)
	clubAPI := club.NewClubAPI(l, r, d, sh)
	postAPI := post.NewPostAPI(l, r, d, sh)
	eventAPI := event.NewEventAPI(l, r, d)
	organizationAPI := organization.NewOrganizationAPI(l, r, d, sh)
	reportAPI := report.NewReportAPI(l, r, d)

	authAPI.Ready(client)
	userAPI.Ready(client)
	fieldAPI.Ready()
	clubAPI.Ready(client)
	postAPI.Ready(client)
	eventAPI.Ready(client)
	organizationAPI.Ready(client)
	reportAPI.Setup(client)

	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}

	// server config
	s := &http.Server{
		Addr:         `:` + port,
		Handler:      r,
		IdleTimeout:  60 * time.Second,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// start server
	go func() {
		l.Info(`starting olympsis server at...` + port)
		err := s.ListenAndServe()

		if err != nil {
			l.Info("error starting server: ", err)
			os.Exit(1)
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
