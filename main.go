package main

import (
	"context"
	"net/http"
	"olympsis-server/auth"
	"olympsis-server/database"
	"olympsis-server/field"
	"olympsis-server/user"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
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

	// authentication service
	authAPI := auth.NewAuthAPI(l, r, d)
	userAPI := user.NewUserAPI(l, r, d)
	fieldAPI := field.NewFieldAPI(l, r, d)

	authAPI.Ready()
	userAPI.Ready()
	fieldAPI.Ready()

	port := os.Getenv("PORT")

	// server config
	s := &http.Server{
		Addr:         `:` + port,
		Handler:      r,
		IdleTimeout:  30 * time.Second,
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
