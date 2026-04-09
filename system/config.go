package system

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"olympsis-server/database"
	"olympsis-server/server"
	"olympsis-server/utils"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/olympsis/models"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/v2/bson"
)

/*
Config Service Struct
*/
type Service struct {
	Database *database.Database // database for read/write operations
	Logger   *logrus.Logger     // logger for logging errors
	Router   *mux.Router        // router for handling incoming requests
}

func NewSystemService(i *server.ServerInterface) *Service {
	return &Service{
		Logger:   i.Logger,
		Router:   i.Router,
		Database: i.Database,
	}
}

func (s *Service) GetConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ctx := context.TODO()

		tags, err := s.FindTags(ctx)
		if err != nil {
			http.Error(w, `{"msg":"failed to get tags for app config."}`, http.StatusInternalServerError)
			return
		}

		sports, err := s.FindSports(ctx)
		if err != nil {
			http.Error(w, `{"msg":"failed to get sports for app config."}`, http.StatusInternalServerError)
			return
		}

		config := models.SystemConfig{
			Tags:   *tags,
			Sports: *sports,
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(config)
	}
}

// generateMapKitJWT creates a signed ES256 JWT for Apple's MapKit token API
// using the shared utility in utils/mapkit.go.
func (s *Service) generateMapKitJWT() (string, error) {
	config := utils.MapKitConfig{
		KeyFilePath: os.Getenv("MAPKIT_FILE_PATH"),
		KeyID:       os.Getenv("MAPKIT_KEY_ID"),
		TeamID:      os.Getenv("APPLE_TEAM_ID"),
	}
	if config.KeyFilePath == "" {
		return "", fmt.Errorf("MAPKIT_FILE_PATH environment variable is not set")
	}
	if config.KeyID == "" {
		return "", fmt.Errorf("MAPKIT_KEY_ID environment variable is not set")
	}
	if config.TeamID == "" {
		return "", fmt.Errorf("APPLE_TEAM_ID environment variable is not set")
	}
	return utils.GenerateMapKitJWT(config)
}

func (s *Service) GetMapkitServerToken() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var bearerToken string

		mode := os.Getenv("MODE")
		if mode != "PRODUCTION" {
			// In dev mode, generate the JWT from the local .p8 key file
			generated, err := s.generateMapKitJWT()
			if err != nil {
				s.Logger.Errorf("[Sys] failed to generate MapKit JWT: %v", err)
				http.Error(w, `{"msg":"failed to generate mapkit token"}`, http.StatusInternalServerError)
				return
			}
			bearerToken = generated
		} else {
			// In production, use the pre-configured MAPKIT_TOKEN
			bearerToken = os.Getenv("MAPKIT_TOKEN")
			if bearerToken == "" {
				s.Logger.Error("MAPKIT_TOKEN environment variable is not set")
				http.Error(w, `{"msg":"server configuration error"}`, http.StatusInternalServerError)
				return
			}
		}

		maxRetries := 3
		var lastErr error

		for attempt := 0; attempt < maxRetries; attempt++ {
			req, err := http.NewRequest("GET", "https://maps-api.apple.com/v1/token", nil)
			if err != nil {
				s.Logger.Errorf("[Sys] failed to create mapkit token request: %v", err)
				http.Error(w, `{"msg":"failed to create request"}`, http.StatusInternalServerError)
				return
			}
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", bearerToken))

			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				lastErr = err
				s.Logger.Errorf("[Sys] mapkit token request failed (attempt %d/%d): %v", attempt+1, maxRetries, err)
				time.Sleep(time.Duration(math.Pow(2, float64(attempt))) * time.Second)
				continue
			}

			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				lastErr = err
				s.Logger.Errorf("[Sys] failed to read mapkit token response (attempt %d/%d): %v", attempt+1, maxRetries, err)
				time.Sleep(time.Duration(math.Pow(2, float64(attempt))) * time.Second)
				continue
			}

			// Retry on 500 errors
			if resp.StatusCode == http.StatusInternalServerError {
				lastErr = fmt.Errorf("apple API returned 500")
				s.Logger.Errorf("[Sys] mapkit token API returned 500 (attempt %d/%d)", attempt+1, maxRetries)
				time.Sleep(time.Duration(math.Pow(2, float64(attempt))) * time.Second)
				continue
			}

			// For any other non-200 status, return immediately (no retry)
			if resp.StatusCode != http.StatusOK {
				s.Logger.Errorf("[Sys] mapkit token API returned status %d: %s", resp.StatusCode, string(body))
				http.Error(w, fmt.Sprintf(`{"msg":"apple API error: %d"}`, resp.StatusCode), resp.StatusCode)
				return
			}

			// Parse the response to extract the access token and expiry
			var appleResp struct {
				AccessToken      string `json:"accessToken"`
				ExpiresInSeconds int    `json:"expiresInSeconds"`
			}
			if err := json.Unmarshal(body, &appleResp); err != nil {
				s.Logger.Errorf("[Sys] failed to parse mapkit token response: %v", err)
				http.Error(w, `{"msg":"failed to parse token response"}`, http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"token":            appleResp.AccessToken,
				"expiresInSeconds": appleResp.ExpiresInSeconds,
			})
			return
		}

		// All retries exhausted
		s.Logger.Errorf("[Sys] mapkit token request failed after %d retries: %v", maxRetries, lastErr)
		http.Error(w, `{"msg":"failed to get mapkit server token"}`, http.StatusInternalServerError)
	}
}

func (s *Service) FindTags(ctx context.Context) (*[]models.Tag, error) {
	var tags []models.Tag
	cursor, err := s.Database.TagsCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}

	for cursor.Next(context.TODO()) {
		var tag models.Tag
		err := cursor.Decode(&tag)
		if err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return &tags, nil
}

func (s *Service) FindSports(ctx context.Context) (*[]models.Sport, error) {
	var sports []models.Sport
	cursor, err := s.Database.SportsCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}

	for cursor.Next(context.TODO()) {
		var sport models.Sport
		err := cursor.Decode(&sport)
		if err != nil {
			return nil, err
		}
		sports = append(sports, sport)
	}
	return &sports, nil
}
