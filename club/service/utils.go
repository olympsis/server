package service

import (
	"context"
	"sync"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SafeOrganizations struct {
	mu            sync.Mutex
	organizations map[primitive.ObjectID]*models.Organization
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

type SafeMembers struct {
	mu      sync.Mutex
	members map[string]*models.UserData
}

func (m *SafeMembers) AddMember(usr *models.UserData) {
	m.mu.Lock()
	m.members[usr.UUID] = usr
	m.mu.Unlock()
}

func (m *SafeMembers) FindMember(uuid string) *models.UserData {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.members[uuid]
}

/*
Provided a filter, return a club and their metadata.

Returns:

	models.Club // an club struct if found if not is nil
	error - // if there is one otherwise is nil

The two return values are mutually exclusive.
If there is an error the error value will be populated and the model struct will be empty and vice versa.
*/
func (s *Service) GetClubAndMetadata(filter interface{}) (models.Club, error) {

	// fetch club data
	var club models.Club
	err := s.FindClub(context.TODO(), filter, &club)
	if err != nil {
		return models.Club{}, err
	}

	var wg sync.WaitGroup
	members := SafeMembers{
		mu:      sync.Mutex{},
		members: make(map[string]*models.UserData),
	}

	// get parent data if it exists
	if club.ParentID != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var org models.Organization
			err := s.Database.OrgCol.FindOne(context.TODO(), bson.M{"_id": club.ParentID}).Decode(&org)
			if err != nil {
				s.Logger.Error("failed to find organization: ", err.Error())
			}
		}()
	}

	// fetch club members data
	for i := range club.Members {
		wg.Add(1)
		go func(index int) {
			uuid := club.Members[index].UUID
			defer wg.Done()
			// lookup member in dictionary
			u := members.FindMember(uuid)
			if u == nil { // if not found search for it
				usr, err := s.SearchService.SearchUserByUUID(uuid)
				if err == nil {
					club.Members[index].Data = &usr
					members.AddMember(&usr)
				} else {
					s.Logger.Error("failed to get user data: ", err.Error())
				}
			} else { // if found just assign it
				club.Members[index].Data = u
			}
		}(i)
	}

	wg.Wait()

	return club, nil
}

/*
Provided a filter, return all of the clubs and their metadata.

Returns:

	*[]models.Club // an array of the clubs found if not is nil
	error - // if there is one otherwise is nil

The two return values are mutually exclusive.
If there is an error the error value will be populated and the array of clubs will be set to nil and vice versa.
*/
func (s *Service) GetClubsAndMetadata(filter interface{}) ([]models.Club, error) {

	// find clubs data
	var clubs []models.Club
	err := s.FindClubs(context.TODO(), filter, &clubs)
	if err != nil {
		return nil, err
	}

	// wait group for goroutines
	// dictionary for org/user data
	var wg sync.WaitGroup

	members := SafeMembers{
		mu:      sync.Mutex{},
		members: make(map[string]*models.UserData),
	}
	organizations := SafeOrganizations{
		mu:            sync.Mutex{},
		organizations: make(map[primitive.ObjectID]*models.Organization),
	}

	// get clubs organization data if they have any
	for i := range clubs {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			// if the club has a parent
			if clubs[index].ParentID != nil {
				// lookup org in the dictionary
				o := organizations.FindOrganization(*clubs[index].ParentID)
				if o == nil { // if org is not in found fetch it
					var org models.Organization
					err := s.Database.OrgCol.FindOne(context.Background(), bson.M{"_id": clubs[index].ParentID}).Decode(&org)
					if err == nil {
						clubs[index].Data = &models.ClubData{
							Parent: &org,
						}
						organizations.AddOrganization(&org)
					}
				} else { // if found just assign it
					clubs[index].Data = &models.ClubData{
						Parent: o,
					}
				}

			}
		}(i)

		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			// get club members data
			for j := range clubs[index].Members {
				uuid := clubs[index].Members[j].UUID
				// lookup member in dictionary
				u := members.FindMember(uuid)
				if u == nil { // if not found search for it
					usr, err := s.SearchService.SearchUserByUUID(clubs[index].Members[j].UUID)
					if err == nil {
						clubs[index].Members[j].Data = &usr
						members.AddMember(&usr)
					} else {
						s.Logger.Error("Failed to get user data: ", err.Error())
					}
				} else { // if found just assign it
					clubs[index].Members[j].Data = u
				}
			}
		}(i)
	}
	wg.Wait()

	return clubs, nil
}
