package service

import (
	"context"
	"io"
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
		fileName := vars["fileName"]
		fileBucket := vars["fileBucket"]

		if len(fileName) < 40 {
			http.Error(rw, "invalid file name", http.StatusBadRequest)
			return
		}

		if len(fileBucket) < 1 {
			http.Error(rw, "invalid file bucket name", http.StatusBadRequest)
			return
		}

		// any requests bigger than 1MB gets denied
		r.Body = http.MaxBytesReader(rw, r.Body, 1<<20+512) // Max request size to 1MB
		err := r.ParseMultipartForm(1 << 20)                // Max 1MB in memory
		if err != nil {
			s.Logger.Error(err.Error())
			http.Error(rw, "request too big", http.StatusBadRequest)
			return
		}

		// Get the image file from the form data
		f, err := GrabFileFromRequest(r, fileName)
		if err != nil {
			s.Logger.Error(err.Error())
			http.Error(rw, "failed to grab image from body", http.StatusInternalServerError)
			return
		}

		// Upload the image to bucket
		err = s.UploadImage(f, fileBucket, fileName)
		if err != nil {
			s.Logger.Error(err.Error())
			http.Error(rw, "failed to upload image", http.StatusInternalServerError)
			return
		}
		defer f.Close()

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
	}
}

func (s *Service) DeleteObject() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		fileName := vars["fileName"]
		fileBucket := vars["fileBucket"]

		if len(fileName) < 40 {
			http.Error(rw, "invalid file name", http.StatusBadRequest)
			return
		}

		if len(fileBucket) < 1 {
			http.Error(rw, "invalid file bucket name", http.StatusBadRequest)
			return
		}

		err := s.DeleteImage(fileBucket, fileName)
		if err != nil {
			s.Logger.Error(err.Error())
			http.Error(rw, "failed to delete image", http.StatusInternalServerError)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
	}
}

func GrabFileFromRequest(r *http.Request, name string) (*os.File, error) {
	file, _, err := r.FormFile("image")
	if err != nil {
		return nil, err
	}

	// Open a new file for writing the image data
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(f, file)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func (s *Service) UploadImage(file *os.File, bucket string, name string) error {
	contentType := "image/jpeg"
	filePath := `/app/` + name

	// Upload the file to the bucket using PutObject
	_, err := s.Client.FPutObject(context.Background(), bucket, name, filePath, minio.PutObjectOptions{ContentType: contentType})
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
