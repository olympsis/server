package utils

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/sirupsen/logrus"
)

type StorageInterface struct {
	ServiceURL string
	Status     string
	Client     http.Client
	Logger     *logrus.Logger

	MapKitToken string
}

func NewStorageInterface(u string, token string, logger *logrus.Logger) *StorageInterface {
	return &StorageInterface{
		ServiceURL:  u,
		Status:      "good",
		Client:      http.Client{},
		Logger:      logger,
		MapKitToken: token,
	}
}

func (s *StorageInterface) GetMapSnapshot(token string, name string) ([]byte, error) {
	var image []byte

	// Check storage for image
	image, err := s.GetSnapshotFromStorage(name)
	if err != nil {
		// We assume we don't have it. Fetch image from mapkit
		return s.GetMapKitSnapshot(token, name)
	}

	return image, nil
}

func (s *StorageInterface) GetMapKitSnapshot(token string, name string) ([]byte, error) {
	// Craft http request
	encodedLocation := url.QueryEscape(name)
	req, err := http.NewRequest("GET", fmt.Sprintf("https://snapshot.apple-mapkit.com/api/v1/snapshot?center=%s&token=%s", encodedLocation, s.MapKitToken), nil)
	if err != nil {
		return nil, err
	}

	// Add Accept header to explicitly request image data
	req.Header.Set("Accept", "image/png")
	req.Header.Set("Origin", "https://api.olympsis.com")
	req.Header.Set("Referer", "https://api.olympsis.com")

	// Make http request
	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check content type to ensure we're getting an image
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		// Read the body for error information
		body, _ := io.ReadAll(resp.Body)
		s.Logger.Errorf("URL Request: %s", req.URL)
		return nil, fmt.Errorf("unexpected content type: %s, body: %s", contentType, string(body))
	}

	if resp.StatusCode != http.StatusOK {
		s.Logger.Errorf("URL Request: %s", req.URL)
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read the image data from the response
	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		s.Logger.Errorf("URL Request: %s", req.URL)
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Start a goroutine to upload the image to storage
	go func() {
		if err := s.UploadMapKitSnapshotToStorage(token, name, imageData); err != nil {
			s.Logger.Errorf("Failed to upload image to storage: %v", err)
		}
	}()

	return imageData, nil
}

func (s *StorageInterface) GetSnapshotFromStorage(name string) ([]byte, error) {
	// Construct the URL
	url := fmt.Sprintf("https://storage.googleapis.com/olympsis-mapkit-snapshots/%s", name)

	// Create new request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "image/png")

	// Make the request
	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("storage returned non-200 status code: %d", resp.StatusCode)
	}

	// Read the image data
	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return imageData, nil
}

func (s *StorageInterface) UploadMapKitSnapshotToStorage(token string, name string, data []byte) error {
	// Create the request to the storage service
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s/v1/storage/olympsis-mapkit-snapshots", s.ServiceURL), bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set necessary headers
	req.Header.Set("Content-Type", "image/jpeg")
	req.Header.Set("X-Filename", name)

	// Make the request
	resp, err := s.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("storage service returned non-200 status code: %d", resp.StatusCode)
	}

	return nil
}
