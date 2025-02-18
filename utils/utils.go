package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"olympsis-server/database"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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

type SafeClubs struct {
	mu    sync.Mutex
	clubs map[primitive.ObjectID]*models.Club
}

func NewSafeClub() *SafeClubs {
	return &SafeClubs{
		mu:    sync.Mutex{},
		clubs: make(map[primitive.ObjectID]*models.Club),
	}
}
func (c *SafeClubs) AddClub(club *models.Club) {
	c.mu.Lock()
	c.clubs[club.ID] = club
	c.mu.Unlock()
}
func (c *SafeClubs) FindClub(id primitive.ObjectID) *models.Club {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.clubs[id]
}

type SafeOrganizations struct {
	mu            sync.Mutex
	organizations map[primitive.ObjectID]*models.Organization
}

func NewSafeOrganization() *SafeOrganizations {
	return &SafeOrganizations{
		mu:            sync.Mutex{},
		organizations: make(map[primitive.ObjectID]*models.Organization),
	}
}
func (o *SafeOrganizations) AddOrganization(org *models.Organization) {
	o.mu.Lock()
	o.organizations[org.ID] = org
	o.mu.Unlock()
}
func (o *SafeOrganizations) FindOrganization(id primitive.ObjectID) *models.Organization {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.organizations[id]
}

type SafeUsers struct {
	mu      sync.Mutex
	members map[string]*models.UserData
}

func NewSafeUsers() *SafeUsers {
	return &SafeUsers{
		mu:      sync.Mutex{},
		members: make(map[string]*models.UserData),
	}
}
func (m *SafeUsers) AddUser(usr *models.UserData) {
	m.mu.Lock()
	m.members[usr.UUID] = usr
	m.mu.Unlock()
}
func (m *SafeUsers) FindUser(uuid string) *models.UserData {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.members[uuid]
}

type SafeFields struct {
	mu     sync.Mutex
	fields map[primitive.ObjectID]*models.Venue
}

func NewSafeFields() *SafeFields {
	return &SafeFields{
		mu:     sync.Mutex{},
		fields: make(map[primitive.ObjectID]*models.Venue),
	}
}
func (m *SafeFields) AddField(field *models.Venue) {
	m.mu.Lock()
	m.fields[field.ID] = field
	m.mu.Unlock()
}
func (m *SafeFields) FindField(id primitive.ObjectID) *models.Venue {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.fields[id]
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

func FindUser(uuid string, database *database.Database) (*models.UserData, error) {

	ctx := context.Background()

	filter := bson.M{
		"$match": bson.M{
			"uuid": uuid,
		},
	}

	authLookup := bson.M{
		"$lookup": bson.M{
			"from":         "auth",
			"localField":   "uuid",
			"foreignField": "uuid",
			"as":           "_auth",
		},
	}

	authAddFields := bson.M{
		"$addFields": bson.M{
			"first_name": bson.M{
				"$arrayElemAt": bson.A{
					"$_auth.first_name",
					0,
				},
			},
			"last_name": bson.M{
				"$arrayElemAt": bson.A{
					"$_auth.last_name",
					0,
				},
			},
		},
	}

	pipeline := bson.A{
		filter,
		authLookup,
		authAddFields,
	}

	cur, err := database.UserCol.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}

	var data models.UserData
	if cur.Next(ctx) {
		err = cur.Decode(&data)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, mongo.ErrNoDocuments
	}

	return &data, nil
}

func HealthCheckHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Health Check: Service is Healthy!")
	}
}
