package utils

import (
	"errors"
	"net/http"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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

func HandleFindError(rw http.ResponseWriter, err error) {
	if err == mongo.ErrNoDocuments {
		http.Error(rw, `{ "msg": "resource not found" }`, http.StatusNotFound)
		return
	} else {
		http.Error(rw, `{ "msg": "failed to find resource" }`, http.StatusNotFound)
		return
	}
}

func HandleInvalidIDError(rw http.ResponseWriter) {
	rw.WriteHeader(http.StatusBadRequest)
	rw.Write([]byte(`{"msg": "invalid id in request" }`))
}

func ValidateClubObject(club *models.Club) bool {
	if club == nil {
		return false
	}
	return !club.ID.IsZero()
}
