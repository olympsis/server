package service

import (
	"olympsis-server/database"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/*
Field Service Struct
*/
type Service struct {
	Database *database.Database

	// logrus logger to Log information about service and errors
	Log *logrus.Logger

	// mux Router to complete http requests
	Router *mux.Router
}

type Field struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	Owner     string             `json:"owner" bson:"owner"`
	Name      string             `json:"name" bson:"name"`
	Notes     string             `json:"notes" bson:"notes"`
	Sports    []string           `json:"sports" bson:"sports"`
	Images    []string           `json:"images" bson:"images"`
	Location  GeoJSON            `json:"location" bson:"location"`
	City      string             `json:"city" bson:"city"`
	State     string             `json:"state" bson:"state"`
	Country   string             `json:"country" bson:"country"`
	Ownership string             `json:"ownership" bson:"ownership"`
}

type GeoJSON struct {
	Type        string    `json:"type" bson:"type"`
	Coordinates []float64 `json:"coordinates" bson:"coordinates"`
}

type FieldsResponse struct {
	TotalFields int     `json:"totalFields"`
	Fields      []Field `json:"fields"`
}
