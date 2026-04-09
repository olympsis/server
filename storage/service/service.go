package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"olympsis-server/server"
	"strings"

	"cloud.google.com/go/storage"
	vision "cloud.google.com/go/vision/v2/apiv1"
	"github.com/gorilla/mux"
	"google.golang.org/api/option"
)

/*
Create new Storage service struct

  - Creates and returns a pointer to a new storage service struct
*/
func NewStorageService(i *server.ServerInterface) *Service {
	return &Service{
		Logger: i.Logger,
		Router: i.Router,
	}
}

// ConnectToClient initializes the GCP Storage and Vision API clients using the given credentials file
func (s *Service) ConnectToClient(credentialsFilePath string) error {
	// Storage Client
	client, err := storage.NewClient(context.TODO(), option.WithCredentialsFile(credentialsFilePath))
	if err != nil {
		s.Logger.Fatal("Failed to create GCP Storage client: " + err.Error())
		return err
	}
	s.Client = client

	// Computer Vision Client
	vClient, err := vision.NewImageAnnotatorClient(context.TODO(), option.WithCredentialsFile(credentialsFilePath))
	if err != nil {
		s.Logger.Fatal("Failed to create Vision API client: " + err.Error())
		return err
	}
	s.VClient = vClient
	s.Logger.Info("Connected to GCP Storage & Vision clients")

	return nil
}

/*
Upload Object (POST)

  - Validates image safety using Google Vision API
  - Uploads file to GCP Storage bucket
  - Returns the file URL and safety score

Returns:

	Http handler
*/
func (s *Service) UploadObject() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		fileBucket := vars["fileBucket"]
		if len(fileBucket) == 0 {
			http.Error(rw, `{ "msg" : "invalid file bucket name" }`, http.StatusBadRequest)
			return
		}

		// TODO: temporary bucket name mapping — remove once clients send the correct name
		fileBucket = resolveBucket(fileBucket)

		// Parse the file from the request body.
		// Supports both multipart/form-data (Android client) and raw binary uploads.
		var bodyData []byte
		contentType := r.Header.Get("Content-Type")
		if strings.HasPrefix(contentType, "multipart/form-data") {
			// Android client sends multipart form data — extract the file part
			if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB max
				s.Logger.Error("Failed to parse multipart form. Error: ", err.Error())
				http.Error(rw, `{ "msg" : "failed to parse form data" }`, http.StatusBadRequest)
				return
			}
			file, _, err := r.FormFile("file")
			if err != nil {
				s.Logger.Error("Failed to get file from form. Error: ", err.Error())
				http.Error(rw, `{ "msg" : "no file found in form data" }`, http.StatusBadRequest)
				return
			}
			defer file.Close()
			bodyData, err = io.ReadAll(file)
			if err != nil {
				s.Logger.Error("Failed to read file from form. Error: ", err.Error())
				http.Error(rw, `{ "msg" : "failed to read file" }`, http.StatusBadRequest)
				return
			}
		} else {
			// Raw binary upload (iOS client or direct API calls)
			var err error
			bodyData, err = io.ReadAll(r.Body)
			if err != nil {
				s.Logger.Error("Failed to read request body. Error: ", err.Error())
				http.Error(rw, `{ "msg" : "failed to read body" }`, http.StatusBadRequest)
				return
			}
		}

		// Get the filename from the request header
		fileName, err := GrabFileName(&r.Header)
		if err != nil {
			s.Logger.Error("No filename found in header")
			http.Error(rw, `{ "msg" : "no file name in header" }`, http.StatusBadRequest)
			return
		}

		// Validate image using Vision API
		resp, err := s.AnnotateImage(bodyData)
		if err != nil {
			s.Logger.Error("Failed to validate image. Error: ", err.Error())
			http.Error(rw, `{ "msg" : "failed to validate image" }`, http.StatusBadRequest)
			return
		}

		var response Response

		if resp == nil || len(resp.Responses) == 0 {
			s.Logger.Error("Vision API returned empty response")
			http.Error(rw, `{ "msg" : "failed to validate image" }`, http.StatusInternalServerError)
			return
		}

		// Calculate safety score
		safety := s.AggregateSafetyScore(resp.Responses[0])
		response.Score = *safety

		if *safety == 5 {
			reason := "Unsafe image"
			response.Reason = &reason

			rw.WriteHeader(http.StatusBadRequest)
			rw.Header().Set("Content-Type", "application/json")
			json.NewEncoder(rw).Encode(response)
			return
		}

		// Upload file to GCP Storage
		err = s._uploadObject(bodyData, fileBucket, fileName)
		if err != nil {
			s.Logger.Error("Failed to upload file. Error: ", err.Error())
			http.Error(rw, `{ "msg" : "failed to upload image" }`, http.StatusBadRequest)
			return
		}

		url := fileBucket + "/" + fileName
		response.URL = &url

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(response)
	}
}

/*
Delete Object (DELETE)

  - Deletes a file from the specified GCP Storage bucket

Returns:

	Http handler
*/
func (s *Service) DeleteObject() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		fileBucket := vars["fileBucket"]
		if len(fileBucket) == 0 {
			http.Error(rw, `{ "msg" : "invalid file bucket name" }`, http.StatusBadRequest)
			return
		}

		// TODO: temporary bucket name mapping — remove once clients send the correct name
		fileBucket = resolveBucket(fileBucket)

		// Get the filename from the request header
		fileName, err := GrabFileName(&r.Header)
		if err != nil {
			s.Logger.Error("Failed to delete file. No file name in header")
			http.Error(rw, `{ "msg" : "no file name in header" }`, http.StatusBadRequest)
			return
		}

		err = s._deleteObject(fileBucket, fileName)
		if err != nil {
			s.Logger.Error(fmt.Sprintf("Failed to delete image (%s). Error: (%s)", fileName, err.Error()))
			http.Error(rw, `{ "msg" : "failed to delete image" }`, http.StatusBadRequest)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
	}
}
