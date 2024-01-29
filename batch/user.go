package batch

import (
	"context"
	"olympsis-server/database"
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
