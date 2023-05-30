package utils

import (
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetTokenFromHeader(r *http.Request) (string, error) {
	// Get the authorization header from the request
	authHeader := r.Header.Get("Authorization")

	// Check if the authorization header is present
	if authHeader == "" {
		return "", errors.New("authorization header not present")
	}

	// Return the token string
	return authHeader, nil
}

func GetClubTokenFromHeader(r *http.Request) (string, error) {
	token := r.Header.Get("X-Admin-Token")
	if token == "" {
		return "", errors.New("no club token found")
	}
	return token, nil
}

func ValidateClubID(s string) bool {
	_, err := primitive.ObjectIDFromHex(s)
	return err == nil
}

func GenerateAuthToken(u string, p string) (string, error) {
	var key = []byte(os.Getenv("KEY"))
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)

	claims["iss"] = "https://api.olympsis.com"
	claims["sub"] = u
	claims["pod"] = p
	claims["iat"] = time.Now().Unix()

	ts, err := token.SignedString(key)

	if err != nil {
		return "", err
	}

	return ts, nil
}

func GenerateClubToken(i string, r string, u string) (string, error) {
	var key = []byte(os.Getenv("KEY"))
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)

	claims["iss"] = i
	claims["sub"] = u
	claims["role"] = r

	ts, err := token.SignedString(key)

	if err != nil {
		return "", err
	}

	return ts, nil
}

func ValidateAuthToken(s string) (string, string, float64, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(s, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("KEY")), nil
	})

	if err != nil {
		return "", "", 0, err
	} else {
		uuid, ok := claims["sub"].(string)
		if !ok {
			return "", "", 0, errors.New("sub claim not found")
		}
		provider, ok := claims["pod"].(string)
		if !ok {
			return "", "", 0, errors.New("pod claim not found")
		}
		createdAt, ok := claims["iat"].(float64)
		if !ok {
			return "", "", 0, errors.New("iat claim not found")
		}
		return uuid, provider, createdAt, nil
	}
}

func ValidateClubToken(s string, u string) (string, string, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(s, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("KEY")), nil
	})

	if err != nil {
		return "", "", err
	} else {
		id := claims["iss"].(string)
		uuid := claims["sub"].(string)
		role := claims["role"].(string)

		if uuid != u {
			return "", "", errors.New("uuid does not match")
		}

		return id, role, nil
	}
}
