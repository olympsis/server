package utils

import (
	"errors"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ValidateObjectID(id string) (primitive.ObjectID, error) {
	if id == "" {
		return primitive.NilObjectID, errors.New("invalid ID")
	}

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return primitive.NilObjectID, err
	}

	return oid, nil
}

func ValidateClubObject(club *models.Club) bool {
	if club == nil {
		return false
	}
	return !club.ID.IsZero()
}
