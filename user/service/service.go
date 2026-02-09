package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"olympsis-server/aggregations"
	"olympsis-server/server"
	"sync"
	"time"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

/*
Creates New User Service

  - Creates new instance of user service object

Args:

	i - server interface with references to common resources

Returns:

	*Service - pointer referencing to new instance of service object
*/
func NewUserService(i *server.ServerInterface) *Service {
	return &Service{
		Log:          i.Logger,
		Router:       i.Router,
		Database:     i.Database,
		Notification: i.Notification,
	}
}

/*
Check User Name (GET)

  - Grab uuid from params

  - Grabs user data from database

Returns:

	Http handler
		- Writes user data back to client
*/
func (u *Service) CheckUsername() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// grab username from query
		keys, ok := r.URL.Query()["username"]
		if !ok || len(keys[0]) < 1 {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(`{ "msg": "no userName found in request" }`))
			return
		}
		userName := keys[0]

		_, c := context.WithTimeout(context.Background(), 30*time.Second)
		defer c()

		// find user data in database
		var user models.User
		filter := bson.D{bson.E{Key: "username", Value: userName}}
		err := u.Database.UserCollection.FindOne(context.TODO(), filter).Decode(&user)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				rw.WriteHeader(http.StatusOK)
				rw.Write([]byte(`{ "is_available": true }`))
				return
			}
		}

		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{ "is_available": false }`))
	}
}

/*
Create User Data (PUT)

  - Creates new user for olympsis (on sign up)

  - Grab request body

  - Create User data in user database

Returns:

	Http handler
		- Writes token back to client
*/
func (s *Service) CreateUserData() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		uuid := r.Header.Get("userID")

		// decode request
		var req models.User
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			s.Log.Error(fmt.Sprintf("Failed to decode request: %s\n", err.Error()))
			http.Error(w, `{ "msg": "failed to decode request" }`, http.StatusBadRequest)
			return
		}

		user := models.User{
			ID:           bson.NewObjectID(),
			UUID:         uuid,
			UserName:     req.UserName,
			Sports:       req.Sports,
			Visibility:   "public",
			HasOnboarded: req.HasOnboarded,
		}

		// insert auth user in database
		err = s.InsertUser(context.Background(), &user)
		if err != nil {
			s.Log.Error(fmt.Sprintf("Failed to insert user into database: %s\n", err.Error()))
			http.Error(w, `{ "msg": "failed to insert user"}`, http.StatusInternalServerError)
			return
		}

		usr, err := aggregations.AggregateUser(&uuid, s.Database)
		if err != nil {
			s.Log.Error(fmt.Sprintf("Failed to find user data: %s\n", err.Error()))
			http.Error(w, `{ "msg": "failed to find user data"}`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(usr)
	}
}

/*
Update User Data (POST)

  - Updates user data

  - Grab parameters and update

Returns:

	Http handler
		- Writes token back to client
*/
func (s *Service) UpdateUserData() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		uuid := r.Header.Get("userID")

		// decode request
		var req models.UserDao
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			s.Log.Error(fmt.Sprintf("Failed to decode request: %s\n", err.Error()))
			http.Error(w, `{ "msg": "failed to decode request" }`, http.StatusBadRequest)
			return
		}

		// Special handling for notification devices
		if req.NotificationDevices != nil && len(*req.NotificationDevices) > 0 {
			// First get the current user to access existing devices
			filter := bson.M{"uuid": uuid}
			var currentUser models.User
			err = s.Database.UserCollection.FindOne(context.Background(), filter).Decode(&currentUser)
			if err != nil {
				s.Log.Error(fmt.Sprintf("Failed to find user: %s\n", err.Error()))
				http.Error(w, `{ "msg": "failed to find user" }`, http.StatusInternalServerError)
				return
			}

			// Create map of incoming devices by ID for quick lookup
			incomingDevices := make(map[string]models.NotificationDevice)
			for _, device := range *req.NotificationDevices {
				timestamp := bson.NewDateTimeFromTime(time.Now())
				device.UpdatedAt = &timestamp
				incomingDevices[device.DeviceID] = device
			}

			// Update existing devices or add new ones
			updatedDevices := []models.NotificationDevice{}
			if currentUser.NotificationDevices != nil {
				for _, existingDevice := range *currentUser.NotificationDevices {
					if updatedDevice, exists := incomingDevices[existingDevice.DeviceID]; exists {
						// Update existing device
						updatedDevices = append(updatedDevices, updatedDevice)
						// Remove from map to track which ones are processed
						delete(incomingDevices, existingDevice.DeviceID)
					} else {
						// Keep unchanged device
						updatedDevices = append(updatedDevices, existingDevice)
					}
				}
			}

			// Add any remaining new devices
			for _, device := range incomingDevices {
				updatedDevices = append(updatedDevices, device)
			}

			// Update the request with the merged devices
			req.NotificationDevices = &updatedDevices
		}

		filter := bson.M{"uuid": uuid}
		changes := bson.M{}
		if req.UserName != nil {
			changes["username"] = req.UserName
		}
		if req.ImageURL != nil {
			changes["image_url"] = req.ImageURL
		}
		if req.Gender != nil {
			changes["gender"] = req.Gender
		}
		if req.Bio != nil {
			changes["bio"] = req.Bio
		}
		if req.Sports != nil && len(*req.Sports) > 0 {
			changes["sports"] = req.Sports
		}
		if req.Visibility != nil {
			changes["visibility"] = req.Visibility
		}
		if req.AcceptedEULA != nil {
			changes["accepted_eula"] = req.AcceptedEULA
		}
		if req.HasOnboarded != nil {
			changes["has_onboarded"] = req.HasOnboarded
		}
		if req.Hometown != nil {
			changes["hometown"] = req.Hometown
		}
		if req.LastLocation != nil {
			changes["last_location"] = req.LastLocation
		}
		if req.BlockedUsers != nil {
			changes["blocked_users"] = req.BlockedUsers
		}
		if req.NotificationDevices != nil {
			changes["notification_devices"] = req.NotificationDevices
		}
		if req.NotificationPreference != nil {
			changes["notification_preference"] = req.NotificationPreference
		}

		timestamp := bson.NewDateTimeFromTime(time.Now())
		changes["updated_at"] = timestamp

		update := bson.M{"$set": changes}

		err = s.UpdateUser(context.Background(), filter, update, &req)
		if err != nil {
			s.Log.Error(fmt.Sprintf("Failed to update user data: %s\n", err.Error()))
			http.Error(w, `{ "msg": "failed to update user data" }`, http.StatusInternalServerError)
			return
		}

		user, err := aggregations.AggregateUser(&uuid, s.Database)
		if err != nil {
			s.Log.Error(fmt.Sprintf("Failed to find user data: %s\n", err.Error()))
			http.Error(w, `{ "msg": "failed to find user data" }`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(user)
	}
}

/*
Get User Data (GET)

  - Grab uuid from params

  - Grabs user data from database

Returns:

	Http handler
		- Writes user data back to client
*/
func (s *Service) GetUserData() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		uuid := r.Header.Get("userID")

		user, err := aggregations.AggregateUser(&uuid, s.Database)
		if err != nil || user.Username == "" {
			s.Log.Error(fmt.Sprintf("Failed to find user data: %s\n", err.Error()))
			http.Error(w, `{ "msg": "failed to find user data" }`, http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(user)
	}
}

/*
Delete User Data (DELETE)

  - Delete User data

Returns:

	Http handler
		- Writes user data back to client
*/
func (u *Service) DeleteUserData() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		uuid := r.Header.Get("userID")

		// delete user data from database
		filter := bson.M{"uuid": uuid}
		err := u.DeleteUser(context.Background(), filter)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				rw.WriteHeader(http.StatusNotFound)
				rw.Write([]byte(`{ "msg": "user not found" }`))
				return
			}
		}
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
	}
}

func (u *Service) GetOrganizationInvitations() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		uuid := r.Header.Get("userID")

		filter := bson.M{
			"recipient": uuid,
			"status":    "pending",
		}

		var invitations []models.Invitation
		cursor, err := u.Database.OrgInvitationCollection.Find(context.TODO(), filter)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			u.Log.Error("Failed to fetch invitations: " + err.Error())
			return
		}
		for cursor.Next(context.TODO()) {
			var invite models.Invitation
			err := cursor.Decode(&invite)
			if err != nil {
				u.Log.Error("Failed to decode invitation: " + err.Error())
			}
			var org models.Organization
			err = u.Database.OrgCollection.FindOne(context.TODO(), bson.M{"_id": invite.SubjectID}).Decode(&org)
			if err != nil {
				u.Log.Error("Failed to fetch org data: " + err.Error())
			}
			invite.Data = &models.InvitationData{
				Organization: &org,
			}
			invitations = append(invitations, invite)
		}

		if len(invitations) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		response := models.InvitationsResponse{
			TotalInvitations: len(invitations),
			Invitations:      invitations,
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

func (u *Service) SearchUsersByUserName() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// grab username from query
		keys, ok := r.URL.Query()["username"]
		if !ok || len(keys[0]) < 1 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{ "msg": "no userName found in request" }`))
			return
		}
		userName := keys[0]

		// fetch users that might be related data
		var users []models.UserData
		regex := bson.Regex{Pattern: userName, Options: "i"}
		filter := bson.M{"username": regex}
		cur, err := u.Database.UserCollection.Find(context.TODO(), filter)
		if err != nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		for cur.Next(context.TODO()) {
			var meta models.User
			var data models.UserData
			err := cur.Decode(&meta)
			if err != nil {
				u.Log.Error("Failed to decode user data: " + err.Error())
			}

			data.Bio = meta.Bio
			data.UUID = meta.UUID
			data.Username = meta.UserName
			if meta.ImageURL != nil {
				data.ImageURL = *meta.ImageURL
			}
			data.Visibility = meta.Visibility
			data.NotificationDevices = meta.NotificationDevices
			data.NotificationPreference = meta.NotificationPreference

			if data.Visibility == "public" {
				data.Clubs = meta.Clubs
				data.Sports = meta.Sports
				data.Organizations = meta.Organizations
			}
			users = append(users, data)
		}

		// fetch first and last name
		for i := range users {
			var auth models.AuthUser
			err := u.Database.AuthCollection.FindOne(context.TODO(), bson.M{"uuid": users[i].UUID}).Decode(&auth)
			if err != nil {
				u.Log.Error("Failed to decode user auth data: " + err.Error())
			} else {
				users[i].FirstName = auth.FirstName
				users[i].LastName = auth.LastName
			}
		}

		response := models.UsersDataResponse{
			TotalUsers: len(users),
			Users:      users,
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

func (u *Service) SearchUserByUUID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// grab username from query
		keys, ok := r.URL.Query()["uuid"]
		if !ok || len(keys[0]) < 1 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{ "msg": "no uuid found in request" }`))
			return
		}
		uuid := keys[0]

		// context/filter
		ctx := context.Background()
		filter := bson.M{"uuid": uuid}
		opts := options.FindOne()

		// find and decode auth user data
		var auth models.AuthUser
		err := u.Database.AuthCollection.FindOne(ctx, filter).Decode(&auth)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
		}

		// find and decode user metadata
		var user models.User
		err = u.Database.UserCollection.FindOne(ctx, filter, opts).Decode(&user)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
		}

		imageURL := ""
		if user.ImageURL != nil {
			imageURL = *user.ImageURL
		}

		// create user data object
		userData := models.UserData{
			UUID:                   user.UUID,
			Bio:                    user.Bio,
			Username:               user.UserName,
			FirstName:              auth.FirstName,
			LastName:               auth.LastName,
			ImageURL:               imageURL,
			Visibility:             user.Visibility,
			NotificationDevices:    user.NotificationDevices,
			NotificationPreference: user.NotificationPreference,
		}

		// if user visibility is public display this data if not then don't
		if user.Visibility == "public" {
			userData.Clubs = user.Clubs
			userData.Sports = user.Sports
			userData.Organizations = user.Organizations
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(userData)
	}
}

func (s *Service) CheckIn() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second*10)
		defer cancel()

		uuid := r.Header.Get("userID")
		response := models.CheckIn{}

		var wgError error
		var wg sync.WaitGroup

		// Find user thread
		wg.Add(1)
		go func() {
			defer wg.Done()
			user, err := aggregations.AggregateUser(&uuid, s.Database)
			if err != nil {
				s.Log.Error("Failed to find user. Error: ", err.Error())
				wgError = err
			}

			if user != nil {
				response.User = *user
			}
		}()

		// Find clubs thread
		wg.Add(1)
		go func() {
			defer wg.Done()
			clubs, err := aggregations.FindUserClubs(ctx, uuid, s.Database)
			if err != nil {
				s.Log.Error("Failed to find clubs. Error: ", err.Error())
				wgError = err
			}

			if clubs != nil {
				response.Clubs = clubs
			}
		}()

		// Find organizations thread
		wg.Add(1)
		go func() {
			defer wg.Done()
			orgs, err := aggregations.FindUserOrganizations(ctx, uuid, s.Database)
			if err != nil {
				s.Log.Error("Failed to find organizations. Error: ", err.Error())
				wgError = err
			}

			if orgs != nil {
				response.Organizations = orgs
			}
		}()

		wg.Wait()

		if wgError != nil {
			http.Error(w, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}
