package utils

import (
	"errors"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func ValidateObjectID(id string) (bson.ObjectID, error) {
	if id == "" {
		return bson.NilObjectID, errors.New("invalid ID")
	}

	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return bson.NilObjectID, err
	}

	return oid, nil
}

func ValidateClubObject(club *models.Club) bool {
	if club == nil {
		return false
	}
	return !club.ID.IsZero()
}
