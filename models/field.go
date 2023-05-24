package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Field struct {
	ID          primitive.ObjectID `json:"id,omitempty" bson:"_id"`
	Name        string             `json:"name" bson:"name"`
	Owner       Ownership          `json:"owner" bson:"owner"`
	Description string             `json:"description" bson:"description"`
	Sports      []string           `json:"sports" bson:"sports"`
	Images      []string           `json:"images" bson:"images"`
	Location    GeoJSON            `json:"location" bson:"location"`
	City        string             `json:"city" bson:"city"`
	State       string             `json:"state" bson:"state"`
	Country     string             `json:"country" bson:"country"`
}

type GeoJSON struct {
	Type        string    `json:"type" bson:"type"`
	Coordinates []float64 `json:"coordinates" bson:"coordinates"`
}

type Ownership struct {
	Name string `json:"name" bson:"name"`
	Type string `json:"type" bson:"type"`
}

type FieldsResponse struct {
	TotalFields int     `json:"total_fields"`
	Fields      []Field `json:"fields"`
}
