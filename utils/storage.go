package utils

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"olympsis-server/types"
	"strings"

	"github.com/sirupsen/logrus"
)

type StorageInterface struct {
	Client  http.Client
	Logger  *logrus.Logger
	Storage types.StorageUploader // Direct reference to upload files without external HTTP calls

	MapKitConfig MapKitConfig // Credentials for generating fresh MapKit JWTs
}

func NewStorageInterface(storage types.StorageUploader, mapkitConfig MapKitConfig, logger *logrus.Logger) *StorageInterface {
	return &StorageInterface{
		Client:       http.Client{},
		Logger:       logger,
		Storage:      storage,
		MapKitConfig: mapkitConfig,
	}
}

// GetMapSnapshot checks if the snapshot exists in storage, and if not, fetches it
// from MapKit and uploads it. Returns a URL path: mapkit-snapshots/{hash}.png
func (s *StorageInterface) GetMapSnapshot(name string) (string, error) {
	imageHash := CreateImageHash(name)

	// Check if the image already exists in storage
	exists, err := s.SnapshotExistsInStorage(imageHash)
	if err == nil && exists {
		return fmt.Sprintf("mapkit-snapshots/%s", imageHash), nil
	}

	// Image doesn't exist in storage, fetch from MapKit and upload
	return s.GetMapKitSnapshot(name)
}

// GetMapKitSnapshot fetches the snapshot from Apple MapKit, uploads it to storage
// in the background, and returns a URL path: mapkit-snapshots/{hash}.png
func (s *StorageInterface) GetMapKitSnapshot(name string) (string, error) {
	// Generate a fresh JWT for each request since MapKit tokens expire after 30 minutes
	mapkitToken, err := GenerateMapKitJWT(s.MapKitConfig)
	if err != nil {
		return "", fmt.Errorf("failed to generate MapKit JWT: %w", err)
	}

	imageHash := CreateImageHash(name)
	encodedLocation := url.QueryEscape(name)
	zoom := 15
	req, err := http.NewRequest("GET", fmt.Sprintf("https://snapshot.apple-mapkit.com/api/v1/snapshot?center=%s&token=%s&z=%d", encodedLocation, mapkitToken, zoom), nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "image/png")
	req.Header.Set("Origin", "https://api.olympsis.com")
	req.Header.Set("Referer", "https://api.olympsis.com")

	resp, err := s.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		body, _ := io.ReadAll(resp.Body)
		s.Logger.Errorf("URL Request: %s", req.URL)
		return "", fmt.Errorf("unexpected content type: %s, body: %s", contentType, string(body))
	}

	if resp.StatusCode != http.StatusOK {
		s.Logger.Errorf("URL Request: %s", req.URL)
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		s.Logger.Errorf("URL Request: %s", req.URL)
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Upload to storage directly using the storage service (no external HTTP call)
	go func() {
		if err := s.Storage.UploadToStorage(imageData, "olympsis-mapkit-snapshots", imageHash); err != nil {
			s.Logger.Errorf("Failed to upload snapshot to storage: %v", err)
		}
	}()

	return fmt.Sprintf("mapkit-snapshots/%s", imageHash), nil
}

// SnapshotExistsInStorage checks if a snapshot exists in GCS using a HEAD request
func (s *StorageInterface) SnapshotExistsInStorage(name string) (bool, error) {
	url := fmt.Sprintf("https://storage.googleapis.com/olympsis-mapkit-snapshots/%s", name)

	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.Client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}
