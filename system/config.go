package system

import (
	"context"
	"encoding/json"
	"net/http"
	"olympsis-server/database"
	"olympsis-server/server"

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
