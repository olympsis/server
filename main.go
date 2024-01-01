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
	"olympsis-server/user"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/olympsis/notif"
	"github.com/olympsis/search"
	"github.com/sirupsen/logrus"
)

func main() {
	// logger
	l := logrus.New()

	// mux router
	r := mux.NewRouter()

	// database
	d := database.NewDatabase(l)
	d.EstablishConnection()

	// notifications service
	k := os.Getenv("KEYID")
	t := os.Getenv("TEAMID")
	f := "./files/AuthKey_JN25FUC9X2.p8"
	n := notif.NewNotificationService(l, d.NotifCol, d.UserCol)
	err := n.CreateNewClient(k, t, f)
	if err != nil {
		panic(err.Error())
	}

	// search service
	sh := search.NewSearchService(l, d.AuthCol, d.UserCol)

	authAPI := auth.NewAuthAPI(l, r, d)
	userAPI := user.NewUserAPI(l, r, d, n)
	fieldAPI := field.NewFieldAPI(l, r, d)
	clubAPI := club.NewClubAPI(l, r, d, n, sh)
	postAPI := post.NewPostAPI(l, r, d, n, sh)
	eventAPI := event.NewEventAPI(l, r, d, n, sh)
	organizationAPI := organization.NewOrganizationAPI(l, r, d, n, sh)

	authAPI.Ready()
	userAPI.Ready()
	fieldAPI.Ready()
	clubAPI.Ready()
	postAPI.Ready()
	eventAPI.Ready()
	organizationAPI.Ready()

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

	l.Printf("Recieved Termination(%s), graceful shutdown \n", sig)

	tc, c := context.WithTimeout(context.Background(), 30*time.Second)

	defer c()

	s.Shutdown(tc)
}
