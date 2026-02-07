package utils

import (
	"sync"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type SafeClubs struct {
	mu    sync.Mutex
	clubs map[bson.ObjectID]*models.Club
}

func NewSafeClub() *SafeClubs {
	return &SafeClubs{
		mu:    sync.Mutex{},
		clubs: make(map[bson.ObjectID]*models.Club),
	}
}
func (c *SafeClubs) AddClub(club *models.Club) {
	c.mu.Lock()
	c.clubs[club.ID] = club
	c.mu.Unlock()
}
func (c *SafeClubs) FindClub(id bson.ObjectID) *models.Club {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.clubs[id]
}

type SafeOrganizations struct {
	mu            sync.Mutex
	organizations map[bson.ObjectID]*models.Organization
}

func NewSafeOrganization() *SafeOrganizations {
	return &SafeOrganizations{
		mu:            sync.Mutex{},
		organizations: make(map[bson.ObjectID]*models.Organization),
	}
}
func (o *SafeOrganizations) AddOrganization(org *models.Organization) {
	o.mu.Lock()
	o.organizations[org.ID] = org
	o.mu.Unlock()
}
func (o *SafeOrganizations) FindOrganization(id bson.ObjectID) *models.Organization {
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
	fields map[bson.ObjectID]*models.Venue
}

func NewSafeFields() *SafeFields {
	return &SafeFields{
		mu:     sync.Mutex{},
		fields: make(map[bson.ObjectID]*models.Venue),
	}
}
func (m *SafeFields) AddField(field *models.Venue) {
	m.mu.Lock()
	m.fields[field.ID] = field
	m.mu.Unlock()
}
func (m *SafeFields) FindField(id bson.ObjectID) *models.Venue {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.fields[id]
}
