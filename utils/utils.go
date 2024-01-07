package utils

import (
	"errors"
	"net/http"
	"os"
	"sync"
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
	claims["exp"] = time.Now().Add(30 * 24 * time.Hour) // 30 days

	ts, err := token.SignedString(key)

	if err != nil {
		return "", err
	}

	return ts, nil
}

func ValidateAuthToken(s string) (string, string, float64, float64, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(s, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("KEY")), nil
	})

	if err != nil {
		return "", "", 0, 0, err
	} else {
		uuid, ok := claims["sub"].(string)
		if !ok {
			return "", "", 0, 0, errors.New("sub claim not found")
		}
		provider, ok := claims["pod"].(string)
		if !ok {
			return "", "", 0, 0, errors.New("pod claim not found")
		}
		createdAt, ok := claims["iat"].(float64)
		if !ok {
			return "", "", 0, 0, errors.New("iat claim not found")
		}
		expiresAt, ok := claims["exp"].(float64)
		if !ok {
			return "", "", 0, 0, errors.New("exp claim not found")
		}

		return uuid, provider, createdAt, expiresAt, nil
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
	fields map[primitive.ObjectID]*models.Field
}

func NewSafeFields() *SafeFields {
	return &SafeFields{
		mu:     sync.Mutex{},
		fields: make(map[primitive.ObjectID]*models.Field),
	}
}
func (m *SafeFields) AddField(field *models.Field) {
	m.mu.Lock()
	m.fields[field.ID] = field
	m.mu.Unlock()
}
func (m *SafeFields) FindField(id primitive.ObjectID) *models.Field {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.fields[id]
}
