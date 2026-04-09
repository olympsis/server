package service

import (
	"cloud.google.com/go/storage"
	vision "cloud.google.com/go/vision/v2/apiv1"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// Service handles GCP Storage uploads/deletes and Vision API image moderation
type Service struct {
	Client  *storage.Client
	VClient *vision.ImageAnnotatorClient
	Logger  *logrus.Logger
	Router  *mux.Router
}

// Response is the JSON response for upload operations
type Response struct {
	// image url
	URL *string `json:"url,omitempty"`

	// image safety score (0-5, where 5 is unsafe)
	Score int `json:"score"`

	// reason the image was rejected
	Reason *string `json:"reason,omitempty"`
}
