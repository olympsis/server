package service

import (
	"context"
	"errors"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
)

func NewStorageService(l *logrus.Logger, r *mux.Router) *Service {
	return &Service{Logger: l, Router: r}
}

func (s *Service) ConnectToClient() {
	endpoint := os.Getenv("STORAGE_ADDR")
	accessKey := os.Getenv("STORAGE_ACCESS_KEY")
	secretKey := os.Getenv("STORAGE_SECRET_KEY")
	useSSL := false

	// Initialize minio client object.
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL})
	if err != nil {
		s.Logger.Fatalln(err)
	} else {
		s.Client = minioClient
	}
}

func (s *Service) UploadObject() http.HandlerFunc {
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

		// Upload the image to bucket
		err = s.UploadImage(r, fileBucket, fileName)
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

		err = s.DeleteImage(fileBucket, fileName)
		if err != nil {
			s.Logger.Error(err.Error())
			http.Error(rw, "failed to delete image", http.StatusInternalServerError)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
	}
}

func (s *Service) UploadImage(req *http.Request, bucket string, name string) error {
	// Upload the file to the bucket using PutObject
	_, err := s.Client.PutObject(context.Background(), bucket, name, req.Body, req.ContentLength, minio.PutObjectOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) DeleteImage(bucket string, name string) error {
	// Delete object from bucket
	err := s.Client.RemoveObject(context.Background(), bucket, name, minio.RemoveObjectOptions{})
	if err != nil {
		return err
	}
	return nil
}

func GrabFileName(h *http.Header) (string, error) {
	// Get the filename from the request header.
	fileName := h.Get("X-Filename")
	if fileName == "" {
		return "", errors.New("missing X-Filename header")
	} else {
		return fileName, nil
	}
}
