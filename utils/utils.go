package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/olympsis/models"
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
	claims["exp"] = time.Now().Add(30 * 24 * time.Hour).Unix() // 30 days

	ts, err := token.SignedString(key)

	if err != nil {
		return "", err
	}

	return ts, nil
}

func ValidateAuthToken(s string) (string, float64, float64, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(s, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("SECRET")), nil
	})

	if err != nil {
		return "", 0, 0, err
	} else {
		uuid, ok := claims["sub"].(string)
		if !ok {
			return "", 0, 0, errors.New("sub claim not found")
		}
		createdAt, ok := claims["iat"].(float64)
		if !ok {
			return "", 0, 0, errors.New("iat claim not found")
		}
		expiresAt, ok := claims["exp"].(float64)
		if !ok {
			return "", 0, 0, errors.New("exp claim not found")
		} else {
			now := time.Now().Unix()
			if expiresAt < float64(now) {
				return "", 0, 0, errors.New("token is expired")
			}
		}

		return uuid, createdAt, expiresAt, nil
	}
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

// Contact the notification service and send a notification to a token belonging to a device
func SendNotificationToToken(token string, notification *models.Notification) error {

	// convert notification to json
	data, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	// craft http request
	url := os.Getenv("NOTIF_URL")
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/notifications/%s", url, token), bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	// Set the request headers
	req.Header.Set("Content-Type", "application/json")

	// Create a new HTTP client and send the request
	client := &http.Client{}
	go func() {
		_, err := client.Do(req)
		if err != nil {
			fmt.Sprintln("Notification error: " + err.Error())
		}
	}()

	return nil
}

// Contact the notification service and send a notification to a topic
func SendNotificationToTopic(notification *models.Notification) error {

	// convert notification to json
	data, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	// craft http request
	url := os.Getenv("NOTIF_URL")
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/notifications", url), bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	// Set the request headers
	req.Header.Set("Content-Type", "application/json")

	// Create a new HTTP client and send the request
	client := &http.Client{}
	go func() {
		_, err := client.Do(req)
		if err != nil {
			fmt.Sprintln("Notification error: " + err.Error())
		}
	}()

	return nil
}

// Contact the notification service and create a new topic
func CreateNotificationTopic(name string) error {
	// data, err := json.Marshal(models.Topic{Name: name})
	// if err != nil {
	// 	return err
	// }

	// // craft http request
	// url := os.Getenv("NOTIF_URL")
	// req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/notifications/topics", url), bytes.NewBuffer(data))
	// if err != nil {
	// 	return err
	// }

	// // Set the request headers
	// req.Header.Set("Content-Type", "application/json")

	// // Create a new HTTP client and send the request
	// client := &http.Client{}
	// resp, err := client.Do(req)
	// if err != nil {
	// 	return err
	// }

	// if resp.StatusCode != 200 {
	// 	return errors.New("status code not ok")
	// }

	return nil
}

// Contact the notification service and delete a topic
func DeleteNotificationTopic(name string) error {
	// // craft http request
	// url := os.Getenv("NOTIF_URL")
	// req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/v1/notifications/topics/%s", url, name), &bytes.Buffer{})
	// if err != nil {
	// 	return err
	// }

	// // Set the request headers
	// req.Header.Set("Content-Type", "application/json")

	// // Create a new HTTP client and send the request
	// client := &http.Client{}
	// resp, err := client.Do(req)
	// if err != nil {
	// 	return err
	// }

	// if resp.StatusCode != 200 {
	// 	return errors.New("status code not ok")
	// }

	return nil
}

// Contact the notification service and add a user to a topic
func AddTokenToTopic(topic string, user string) error {
	// data, err := json.Marshal(models.ModifyTopic{User: user})
	// if err != nil {
	// 	return err
	// }

	// // craft http request
	// url := os.Getenv("NOTIF_URL")
	// req, err := http.NewRequest("PUT", fmt.Sprintf("%s/v1/notifications/topics/%s/add", url, topic), bytes.NewBuffer(data))
	// if err != nil {
	// 	return err
	// }

	// // Set the request headers
	// req.Header.Set("Content-Type", "application/json")

	// // Create a new HTTP client and send the request
	// client := &http.Client{}
	// resp, err := client.Do(req)
	// if err != nil {
	// 	return err
	// }

	// if resp.StatusCode != 200 {
	// 	return errors.New("status code not ok")
	// }

	return nil
}

// Contact the notification service and remove a user from a topic
func RemoveTokenFromTopic(topic string, user string) error {
	// data, err := json.Marshal(models.ModifyTopic{User: user})
	// if err != nil {
	// 	return err
	// }

	// craft http request
	// url := os.Getenv("NOTIF_URL")
	// req, err := http.NewRequest("PUT", fmt.Sprintf("%s/v1/notifications/topics/%s/remove", url, topic), bytes.NewBuffer(data))
	// if err != nil {
	// 	return err
	// }

	// // Set the request headers
	// req.Header.Set("Content-Type", "application/json")

	// // Create a new HTTP client and send the request
	// client := &http.Client{}
	// resp, err := client.Do(req)
	// if err != nil {
	// 	return err
	// }

	// if resp.StatusCode != 200 {
	// 	return errors.New("status code not ok")
	// }

	return nil
}
