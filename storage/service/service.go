package service

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"cloud.google.com/go/storage"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
)

func NewStorageService(l *logrus.Logger, r *mux.Router) *Service {
	return &Service{Logger: l, Router: r}
}

func (s *Service) ConnectToClient() {
	filePath := "./files/credentials.json"
	client, err := storage.NewClient(context.TODO(), option.WithCredentialsFile(filePath))
	if err != nil {
		s.Logger.Fatal("failed to create client" + err.Error())
	}
	s.Client = client
}

func (s *Service) UploadObject() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		fileBucket := vars["fileBucket"]
		if len(fileBucket) < 1 {
			http.Error(rw, "invalid file bucket name", http.StatusBadRequest)
			return
		}

		// Print the request headers
		fmt.Println("-----------------------------------------")
		for name, values := range r.Header {
			for _, value := range values {
				fmt.Printf("%s: %s\n", name, value)
			}
		}
		fmt.Println("-----------------------------------------")

		// read body data
		bodyData, err := io.ReadAll(r.Body)
		if err != nil {
			s.Logger.Error(err.Error())
			http.Error(rw, "failed to read body", http.StatusBadRequest)
			return
		}

		// Get the filename from the request header.
		fileName, err := GrabFileName(&r.Header)
		if err != nil {
			s.Logger.Error(err.Error())
			http.Error(rw, "no file name", http.StatusBadRequest)
			return
		}

		// Upload file to MinIO server
		err = s._uploadObject(bodyData, fileBucket, fileName)
		if err != nil {
			s.Logger.Error(err.Error())
			http.Error(rw, "failed to upload image", http.StatusInternalServerError)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
	}
}

func (s *Service) DeleteObject() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		fileBucket := vars["fileBucket"]

		// Get the filename from the request header.
		fileName, err := GrabFileName(&r.Header)
		if err != nil {
			s.Logger.Error(err.Error())
			http.Error(rw, "no file name", http.StatusBadRequest)
			return
		}

		if len(fileBucket) < 1 {
			http.Error(rw, "invalid file bucket name", http.StatusBadRequest)
			return
		}

		err = s._deleteObject(fileBucket, fileName)
		if err != nil {
			s.Logger.Error(err.Error())
			http.Error(rw, "failed to delete image", http.StatusInternalServerError)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
	}
}
