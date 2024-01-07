package batch

import (
	"context"
	"olympsis-server/database"
	"olympsis-server/utils"
	"sync"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/bson"
)

func ProcessBatchedUserData(uuids []string, d *database.Database) ([]models.UserData, error) {

	ctx := context.Background()
	var users []models.UserData

	for i := range uuids {

		var usr models.User
		var wg sync.WaitGroup
		var auth models.AuthUser
		filter := bson.M{"uuid": uuids[i]}

		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			err := d.AuthCol.FindOne(ctx, filter).Decode(&auth)
			if err != nil {
				d.Logger.Error("failed to fetch auth user data: ", err.Error())
			}
		}(i)

		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			err := d.UserCol.FindOne(ctx, filter).Decode(&usr)
			if err != nil {
				d.Logger.Error("failed to fetch user data: ", err.Error())
			}
		}(i)

		wg.Wait()

		user := models.UserData{
			UUID:          auth.UUID,
			Username:      usr.UserName,
			FirstName:     auth.FirstName,
			LastName:      auth.LastName,
			ImageURL:      usr.ImageURL,
			Visibility:    usr.Visibility,
			Bio:           usr.Bio,
			Clubs:         usr.Clubs,
			Organizations: usr.Organizations,
			Sports:        usr.Sports,
		}
		users = append(users, user)
	}

	return users, nil
}

func ProcessBatchedClubData(filter interface{}, d *database.Database) ([]models.Club, error) {

	// fetch clubs data
	ctx := context.Background()
	var clubs []models.Club
	cursor, err := d.ClubCol.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	for cursor.Next(ctx) {
		var club models.Club
		err := cursor.Decode(&club)
		if err != nil {
			d.Logger.Error("failed to decode club: ", err.Error())
		}
		clubs = append(clubs, club)
	}

	// wait group for goroutines
	// dictionary for org/user data
	var wg sync.WaitGroup

	members := utils.NewSafeUsers()
	organizations := utils.NewSafeOrganization()

	// get clubs organization data if they have any
	for i := range clubs {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			// if the club has a parent
			parentId := clubs[index].ParentID
			if parentId != nil {
				// lookup org in the dictionary
				o := organizations.FindOrganization(*parentId)
				if o == nil { // if org is not in found fetch it
					var org models.Organization
					err := d.OrgCol.FindOne(context.Background(), bson.M{"_id": parentId}).Decode(&org)
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
				u := members.FindUser(uuid)
				if u == nil { // if not found search for it

					var auth models.AuthUser
					var usr models.User
					filter := bson.M{"uuid": uuid}

					d.AuthCol.FindOne(ctx, filter).Decode(&auth)
					d.UserCol.FindOne(ctx, filter).Decode(&usr)
					user := models.UserData{
						UUID:          auth.UUID,
						Username:      usr.UserName,
						FirstName:     auth.FirstName,
						LastName:      auth.LastName,
						ImageURL:      usr.ImageURL,
						Visibility:    usr.Visibility,
						Bio:           usr.Bio,
						Clubs:         usr.Clubs,
						Organizations: usr.Organizations,
						Sports:        usr.Sports,
					}

					if err == nil {
						clubs[index].Members[j].Data = &user
						members.AddUser(&user)
					} else {
						d.Logger.Error("failed to get user data: ", err.Error())
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
