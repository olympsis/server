package utils

import (
	"net/http"

	"go.mongodb.org/mongo-driver/mongo"
)

func HandleFindError(rw http.ResponseWriter, err error) {
	if err == mongo.ErrNoDocuments {
		http.Error(rw, `{ "msg": "resource not found" }`, http.StatusNotFound)
		return
	} else {
		http.Error(rw, `{ "msg": "failed to find resource" }`, http.StatusInternalServerError)
		return
	}
}

func HandleInvalidIDError(rw http.ResponseWriter) {
	rw.WriteHeader(http.StatusBadRequest)
	rw.Write([]byte(`{"msg": "invalid id in request" }`))
}
