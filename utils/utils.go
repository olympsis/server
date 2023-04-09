package utils

import (
	"errors"
	"net/http"
	"os"
	"strings"
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

	// Split the authorization header into the bearer and token parts
	authHeaderParts := strings.Split(authHeader, " ")
	if len(authHeaderParts) != 2 || authHeaderParts[0] != "Bearer" {
		return "", errors.New("invalid authorization header format")
	}

	// Return the token string
	return authHeaderParts[1], nil
}

func GetClubTokenFromHeader(r *http.Request) (string, error) {
	token := r.Header.Get("Club Admin Token")
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
		uuid := claims["uuid"].(string)
		provider := claims["provider"].(string)
		createdAt := claims["createdAt"].(float64)
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
		rank := claims["role"].(string)

		if uuid != u {
			return "", "", errors.New("uuid does not match")
		}

		return id, rank, nil
	}
}
