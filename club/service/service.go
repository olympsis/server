package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"olympsis-server/aggregations"
	"olympsis-server/database"
	"olympsis-server/notifications"
	"olympsis-server/server"
	"olympsis-server/utils"
	"time"

	"github.com/gorilla/mux"
	"github.com/olympsis/models"
	"github.com/sirupsen/logrus"
	"github.com/stripe/stripe-go/v82"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Service struct {
	Database     *database.Database
	Logger       *logrus.Logger
	Router       *mux.Router
	StripeClient *stripe.Client
	Notification *notifications.Service
}

// Creates a new club service object
func NewClubService(i *server.ServerInterface) *Service {
	return &Service{
		Logger:       i.Logger,
		Router:       i.Router,
		Database:     i.Database,
		StripeClient: i.Stripe,
		Notification: i.Notification,
	}
}

// Gets clubs based on parameters given
//
// - Process club find parameters
// - Aggregate club objects
//
// Returns: array of Club objects
func (s *Service) GetClubs() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		params, err := parseQueryParams(r)
		if err != nil {
			http.Error(rw, `{"msg": "bad request"}`, http.StatusBadRequest)
			s.Logger.Error("Bad request. Error: ", err.Error())
			return
		}

		filter := bson.M{}

		// Location Query - Only add if we have enough info
		if params.Country != "" {
			filter["country"] = params.Country

			if params.State != "" {
				filter["state"] = params.State

				if params.City != "" {
					filter["city"] = params.City
				}
			}
		}

		// Sports Query
		if len(params.Sports) > 0 {
			filter["sports"] = bson.M{
				"$in": params.Sports,
			}
		}

		// Get all of the clubs data
		clubs, err := aggregations.AggregateClubs(
			filter,          // Regular filter for country/state/city/sports
			params.Location, // GeoJSON location if provided
			params.Radius,   // Radius in meters
			params.Limit,    // Use the limit from params
			params.Skip,     // Use the skip from params
			s.Database,
		)

		if err != nil {
			s.Logger.Error("Failed to find clubs: ", err.Error())
			http.Error(rw, `{ "msg": "failed to find clubs" }`, http.StatusInternalServerError)
			return
		}

		// No content
		if len(*clubs) == 0 {
			s.Logger.Error("No clubs found matching criteria")
			http.Error(rw, `{ "msg": "no clubs found" }`, http.StatusNoContent)
			return
		}

		resp := models.ClubsResponse{
			TotalClubs: len(*clubs),
			Clubs:      *clubs,
		}
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(resp)
	}
}

// Get a club object
//
// - Validates club id
// - Aggregates club data from database
// - Validate object post fetching
//
// Returns: Club object
func (c *Service) GetClub() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// Validate club ID
		id := mux.Vars(r)["id"]
		oid, err := utils.ValidateObjectID(id)
		if err != nil {
			utils.HandleInvalidIDError(rw)
			c.Logger.Error("Invalid Club ID - Error: ", err.Error())
			return
		}

		// Find club in database
		club, err := aggregations.AggregateClub(oid, c.Database)
		if err != nil {
			utils.HandleFindError(rw, err)
			c.Logger.Error(fmt.Sprintf("Failed to find club. ID: %s - Error: %s", id, err.Error()))
			return
		}

		// If club object is malformed
		if !utils.ValidateClubObject(club) {
			http.Error(rw, `{ "msg": "something went wrong" }`, http.StatusInternalServerError)
			c.Logger.Error(fmt.Sprintf("Club Object is malformed. ID: %s", id))
			return
		}

		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(club)
	}
}

// Creates a new club
//
// - Validates request body
// - Adds server-side updates
// - Add club to database
// - Create notification topics
//
// Returns: The created club ID
func (c *Service) CreateClub() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		ctx, cancel := context.WithTimeout(r.Context(), time.Second*15)
		defer cancel()

		uuid := r.Header.Get("UUID")

		// Decode request
		var req models.ClubDao
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			c.Logger.Error("Failed to decode request ", err.Error())
			http.Error(rw, "failed to decode request", http.StatusBadRequest)
			return
		}

		// Additional data
		timeStamp := bson.NewDateTimeFromTime(time.Now())
		verification := false
		req.IsVerified = &verification
		req.CreatedAt = &timeStamp

		// Create club in database
		id, err := c.InsertClub(context.TODO(), &req)
		if err != nil {
			c.Logger.Error("Failed to create club: ", err.Error())
			http.Error(rw, "Failed to create club", http.StatusInternalServerError)
			return
		}

		// Insert owner into members collection
		member := models.MemberDao{
			ID:       bson.NewObjectID(),
			UserID:   uuid,
			Role:     string(models.OwnerMember),
			ClubID:   id,
			JoinedAt: timeStamp,
		}
		_, err = c.InsertMember(ctx, &member)
		if err != nil {
			c.Logger.Error("Failed to create member owner. Error: ", err.Error())
			http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			return
		}

		// Create notification topics
		topicName := id.Hex()
		adminName := id.Hex() + "_admin"
		if err = c.Notification.CreateTopic(topicName, []string{uuid}); err != nil {
			c.Logger.Errorf("Failed to create club topic. Club ID: %s - Error: %s ", id, err.Error())
		}
		if err = c.Notification.CreateTopic(adminName, []string{uuid}); err != nil {
			c.Logger.Errorf("Failed to create club admin topic. Club ID: %s - Error: %s ", id, err.Error())
		}

		rw.WriteHeader(http.StatusCreated)
		rw.Write(fmt.Appendf(nil, `{"id": "%s"}`, id.Hex()))
	}
}

// Updates club data
//
// - Validates club ID
// - Validate role
// - Decodes request
// - Create changes map
// - Update club in database
//
// Returns: OK if successful
func (c *Service) ModifyClub() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		ctx, cancel := context.WithTimeout(r.Context(), time.Second*15)
		defer cancel()

		// Validate club ID
		id := mux.Vars(r)["id"]
		oid, err := utils.ValidateObjectID(id)
		if err != nil {
			utils.HandleInvalidIDError(rw)
			c.Logger.Error("Invalid Club ID - Error: ", err.Error())
			return
		}

		// Validate user role - admin/owners can modify a club
		uuid := r.Header.Get("UUID")
		member, err := c.FindMember(ctx, bson.M{"user_id": uuid, "club_id": oid})
		if err != nil {
			c.Logger.Error("Failed to find member. Error: ", err.Error())
			http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			return
		}
		if member.Role == string(models.MemberMember) {
			http.Error(rw, `{"msg": "invalid permission"}`, http.StatusUnauthorized)
			return
		}

		// Decode request
		var req models.ClubDao
		err = json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			c.Logger.Error("Failed to decode body: ", err.Error())
			http.Error(rw, `{ "msg": "Failed to decode body" }`, http.StatusBadRequest)
			return
		}

		// Conversions and new variables
		filter := bson.M{"_id": oid}
		change := bson.M{}
		update := bson.M{"$set": change}

		// Changes map
		if req.Name != nil {
			change["name"] = req.Name
		}
		if req.ParentID != nil {
			change["parent_id"] = req.ParentID
		}
		if req.Description != nil {
			change["description"] = req.Description
		}
		if req.Tags != nil && len(*req.Tags) > 0 {
			change["tags"] = req.Tags
		}
		if req.Sports != nil {
			change["sports"] = req.Sports
		}
		if req.City != nil {
			change["city"] = req.City
		}
		if req.State != nil {
			change["state"] = req.State
		}
		if req.Country != nil {
			change["country"] = req.Country
		}
		if req.Location != nil {
			change["location"] = req.Location
		}
		if req.Logo != nil {
			change["logo"] = req.Logo
		}
		if req.Banner != nil {
			change["banner"] = req.Banner
		}
		if req.Visibility != nil {
			change["visibility"] = req.Visibility
		}
		if req.BlackList != nil && len(*req.BlackList) > 0 {
			change["blacklist"] = req.BlackList
		}
		if req.Rules != nil && len(*req.Rules) > 0 {
			change["rules"] = req.Rules
		}

		// Update club data in database
		err = c.UpdateClub(context.Background(), filter, update)
		if err != nil {
			c.Logger.Error(fmt.Sprintf("Failed to update club. ID: %s - Error: %s", id, err.Error()))
			http.Error(rw, `{ "msg": "something went wrong" }`, http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{"msg": "OK"}`))
	}
}

// Deletes a club
//
// - Validates club ID
// - Validate user role
// - Remove members from club
// - Delete club from database
// - Deletes club topics
//
// Returns: OK if successful
func (c *Service) DeleteClub() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		ctx, cancel := context.WithTimeout(r.Context(), time.Second*15)
		defer cancel()

		// Validate club ID
		id := mux.Vars(r)["id"]
		oid, err := utils.ValidateObjectID(id)
		if err != nil {
			http.Error(rw, `{ "msg": "invalid club id" }`, http.StatusBadRequest)
			return
		}

		// Validate user role - only owners can delete a club
		uuid := r.Header.Get("UUID")
		member, err := c.FindMember(ctx, bson.M{"user_id": uuid, "club_id": oid})
		if err != nil {
			c.Logger.Error("Failed to find member. Error: ", err.Error())
			http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			return
		}
		if member.Role == string(models.AdminMember) || member.Role == string(models.MemberMember) {
			http.Error(rw, `{"msg": "invalid permission"}`, http.StatusUnauthorized)
			return
		}

		// Delete club members
		err = c.DeleteMembers(ctx, bson.M{"club_id": oid})
		if err != nil {
			c.Logger.Errorf("Failed to delete club members. ID: %s, Error: %s", id, err.Error())
			http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			return
		}

		// Delete club
		err = c.RemoveClub(ctx, bson.M{"_id": oid})
		if err != nil {
			c.Logger.Error(fmt.Sprintf("Failed to delete club. ID: %s - Error: %s", id, err.Error()))
			http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			return
		}

		// Delete club topics
		clubAdminTopic := id + "_admin"
		if err = c.Notification.RemoveTopic(id); err != nil {
			c.Logger.Error(fmt.Sprintf("Failed to delete topic. ID: %s - Error: %s", id, err.Error()))
		}
		if err = c.Notification.RemoveTopic(clubAdminTopic); err != nil {
			c.Logger.Error(fmt.Sprintf("Failed to delete admin topic. ID: %s - Error: %s", id, err.Error()))
		}

		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{ "msg": "OK" }`))
	}
}

// Change member rank
//
// - Validates club ID
// - Validates member ID
// - Validates user role
// - Update member
// - Notify member
//
// Returns: OK if successful
func (c *Service) ChangeMemberRank() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		ctx, cancel := context.WithTimeout(r.Context(), time.Second*15)
		defer cancel()

		// Validate club ID
		id := mux.Vars(r)["id"]
		oid, err := utils.ValidateObjectID(id)
		if err != nil {
			c.Logger.Error("Invalid club ID. Error: ", err.Error())
			http.Error(rw, `{"msg": "invalid club id"}`, http.StatusBadRequest)
			return
		}

		// Validate member ID
		mid := mux.Vars(r)["memberID"]
		moid, err := utils.ValidateObjectID(mid)
		if err != nil {
			c.Logger.Error("Invalid member ID. Error: ", err.Error())
			http.Error(rw, `{"msg": "invalid member id"}`, http.StatusBadRequest)
			return
		}

		// Validate user role - admin/owners can modify a club
		uuid := r.Header.Get("UUID")
		admin, err := c.FindMember(ctx, bson.M{"user_id": uuid, "club_id": oid})
		if err != nil {
			c.Logger.Error("Failed to find member. Error: ", err.Error())
			http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			return
		}
		if admin.Role == string(models.MemberMember) {
			http.Error(rw, `{"msg": "invalid permission"}`, http.StatusUnauthorized)
			return
		}

		// Decode request
		var req models.ChangeRoleRequest
		err = json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			c.Logger.Error("Failed to decode request. Error: ", err.Error())
			http.Error(rw, `{"msg": "invalid request body"}`, http.StatusBadRequest)
			return
		}

		// Find member
		member, err := c.FindMember(ctx, bson.M{"_id": moid})
		if err != nil {
			utils.HandleFindError(rw, err)
			c.Logger.Error("Failed to find member. Error: ", err.Error())
			return
		}

		// Update member
		err = c.UpdateMember(ctx, bson.M{"_id": moid}, bson.M{"role": req.Role})
		if err != nil {
			c.Logger.Error("Failed to update member. Error: ", err.Error())
			http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			return
		}

		// If user was member then add them to the admin topic
		if member.Role == "member" {
			if err = c.Notification.AddUsersToTopic(id+"_admin", []string{member.UserID}); err != nil {
				c.Logger.Errorf("Failed to add user to admin topic. Club ID: %s - Error: %s", id, err.Error())
			}
		}

		if err = c.Notification.ChangedRole(oid, member.UserID, models.MemberRole(member.Role), models.MemberRole(req.Role)); err != nil {
			c.Logger.Errorf("Failed to notify user of role change. Club ID: %s - Error: %s", id, err.Error())
		}

		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{ "msg": "OK" }`))
	}
}

// Kick member
//
// - Validate club ID
// - Validate member ID
// - Validate user role
// - Remove member from members collection
// - Notify the user and remove them from the topics
//
// Returns: OK if successful
func (c *Service) KickMember() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		ctx, cancel := context.WithTimeout(r.Context(), time.Second*15)
		defer cancel()

		// Validate club ID
		id := mux.Vars(r)["id"]
		oid, err := utils.ValidateObjectID(id)
		if err != nil {
			http.Error(rw, `{"msg": "invalid club id"}`, http.StatusBadRequest)
			return
		}

		// Validate member ID
		mid := mux.Vars(r)["memberID"]
		moid, err := utils.ValidateObjectID(mid)
		if err != nil {
			http.Error(rw, `{"msg": "invalid member id"}`, http.StatusBadRequest)
			return
		}

		// Validate user role - admin/owners can modify a club
		uuid := r.Header.Get("UUID")
		admin, err := c.FindMember(ctx, bson.M{"user_id": uuid, "club_id": oid})
		if err != nil {
			c.Logger.Error("Failed to find member. Error: ", err.Error())
			http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			return
		}
		if admin.Role == string(models.MemberMember) {
			http.Error(rw, `{"msg": "invalid permission"}`, http.StatusUnauthorized)
			return
		}

		// Find member
		member, err := c.FindMember(ctx, bson.M{"_id": moid})
		if err != nil {
			utils.HandleFindError(rw, err)
			c.Logger.Error(fmt.Sprintf("Failed to find member. ID: %s - Error: %s", mid, err.Error()))
			return
		}

		// Remove user from members collection
		err = c.DeleteMember(ctx, bson.M{"_id": mid})
		if err != nil {
			c.Logger.Error("Failed to find remove member. Error: ", err.Error())
			http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			return
		}

		if err = c.Notification.Kicked(&oid, member); err != nil {
			c.Logger.Errorf("Failed to notify user. Club ID: %s - Error: %s", id, err.Error())
		}

		// Remove user from topics
		clubTopic := id
		adminTopic := id + "_admin"
		if err = c.Notification.RemoveUsersFromTopic(clubTopic, []string{member.UserID}); err != nil {
			c.Logger.Errorf("Failed to remove user from topic. Club ID: %s - Error: %s", id, err.Error())
		}
		if err = c.Notification.RemoveUsersFromTopic(adminTopic, []string{member.UserID}); err != nil {
			c.Logger.Errorf("Failed to remove user from admin topic. Club ID: %s - Error: %s", id, err.Error())
		}

		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{"msg": "OK"}`))
	}
}

// Leave club
//
// - Validate club ID
// - Remove member from members collection
// - Remove member from topics
//
// Returns: OK if successful
func (c *Service) LeaveClub() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		ctx, cancel := context.WithTimeout(r.Context(), time.Second*15)
		defer cancel()

		uuid := r.Header.Get("UUID")

		// Validate club ID
		id := mux.Vars(r)["id"]
		oid, err := utils.ValidateObjectID(id)
		if err != nil {
			http.Error(rw, `{"msg": "invalid club id"}`, http.StatusBadRequest)
			return
		}

		// Remove member
		err = c.DeleteMember(ctx, bson.M{"user_id": uuid, "club_id": oid})
		if err != nil {
			c.Logger.Error("Failed to delete member. Error: ", err.Error())
			http.Error(rw, `{"mgs": "something went wrong"}`, http.StatusInternalServerError)
			return
		}

		// Remove member from topics
		topicName := id
		adminName := id + "_admin"
		if err = c.Notification.RemoveUsersFromTopic(topicName, []string{uuid}); err != nil {
			c.Logger.Errorf("Failed to remove user from topic. Club ID: %s - Error: %s", id, err.Error())
		}
		if err = c.Notification.RemoveUsersFromTopic(adminName, []string{uuid}); err != nil {
			c.Logger.Errorf("Failed to remove user from admin topic. Club ID: %s - Error: %s", id, err.Error())
		}

		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{ "msg": "OK" }`))
	}
}

/*
CLUB POST ENDPOINTS
*/

// Pin a club post
//
// - Validates club ID
// - Validates post ID
// - Updates the club to pin the post
//
// Returns: Ok if successful
func (s *Service) PinClubPost() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// Validate club ID
		id, err := utils.ValidateObjectID(mux.Vars(r)["id"])
		if err != nil {
			http.Error(rw, `{"msg": "invalid club id"}`, http.StatusBadRequest)
			return
		}

		// Validate post ID
		postID, err := utils.ValidateObjectID(mux.Vars(r)["postID"])
		if err != nil {
			http.Error(rw, `{"msg": "invalid post id"}`, http.StatusBadRequest)
			return
		}

		// Pin the post
		ok := s.PinPost(&id, &postID)
		if !ok {
			http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{"msg": "OK"}`))
	}
}

// Unpin a club post
//
// - Validates club id
// - Updates the club to remove the pinned post
//
// Returns: Ok if successful
func (s *Service) UnpinClubPost() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		// Validate club ID
		id, err := utils.ValidateObjectID(mux.Vars(r)["id"])
		if err != nil {
			http.Error(rw, `{"msg": "invalid club id"}`, http.StatusBadRequest)
			return
		}

		// Remove the pinned post
		ok := s.UnpinPost(&id)
		if !ok {
			http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			return

		}

		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{"msg": "OK"}`))
	}
}

/*
FINANCE ENDPOINTS
*/

// CreateFinancialAccount creates a Stripe Connect account for the club
//
// - Validates club ID and user permissions
// - Checks if financial account already exists
// - Creates Stripe Connect Express account
// - Stores account details in database
//
// Returns: Stripe account creation response with onboarding URL
func (s *Service) CreateFinancialAccount() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second*15)
		defer cancel()

		// Validate club ID
		id := mux.Vars(r)["id"]
		oid, err := utils.ValidateObjectID(id)
		if err != nil {
			utils.HandleInvalidIDError(rw)
			s.Logger.Error("Invalid Club ID - Error: ", err.Error())
			return
		}

		// Validate user permissions - only owners/admins can create financial accounts
		uuid := r.Header.Get("UUID")
		member, err := s.FindMember(ctx, bson.M{"user_id": uuid, "club_id": oid})
		if err != nil {
			utils.HandleFindError(rw, err)
			s.Logger.Error("Failed to find member. Error: ", err.Error())
			return
		}
		if member.Role == string(models.OwnerMember) {
			http.Error(rw, `{"msg": "insufficient permissions"}`, http.StatusUnauthorized)
			return
		}

		// Check if account already exists
		existingAccount, _ := s.FindFinancialAccount(ctx, bson.M{"club_id": oid})
		if existingAccount != nil {
			http.Error(rw, `{"msg": "financial account already exists"}`, http.StatusConflict)
			return
		}

		// Find club information for stripe customer account
		club, err := s.FindClub(ctx, bson.M{"_id": oid})
		if err != nil {
			http.Error(rw, `{"msg": "failed to get club"}`, http.StatusInternalServerError)
			s.Logger.Errorf("Failed to get club ID: %s - Error: %s", id, err.Error())
			return
		}

		// Create Stripe club customer
		params := &stripe.CustomerCreateParams{
			Name:             club.Name,
			Description:      stripe.String("Club wallet account"),
			Email:            stripe.String("gostripe@stripe.com"),
			PreferredLocales: stripe.StringSlice([]string{"en", "es"}),
			Metadata: map[string]string{
				"club_id": id,
			},
		}
		customer, err := s.StripeClient.V1Customers.Create(ctx, params)
		if err != nil {
			http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			s.Logger.Errorf("Failed to create club stripe customer. ClubID: %s - Error: %s", id, err.Error())
			return
		}

		// Add Club financial info to the database
		status := "active"
		tempBalance := float64(0.00)
		currencyString := "usd"
		timestamp := bson.DateTime(time.Now().Unix())
		account := models.ClubFinancialAccount{
			ClubID:          &oid,
			StripeAccountID: &customer.ID,
			AccountStatus:   &status,
			Currency:        &currencyString,
			CreatedAt:       &timestamp,
		}
		aid, err := s.InsertFinancialAccount(ctx, &account)
		if err != nil {
			http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			s.Logger.Errorf("Failed to add financial account. Club ID: %s - Error: %s", id, err.Error())
			return
		}

		account.ID = &aid
		account.AvailableBalance = &tempBalance

		rw.WriteHeader(http.StatusCreated)
		json.NewEncoder(rw).Encode(account)
	}
}

// GetFinancialAccount retrieves the club's financial account details
//
// - Validates club ID and user permissions
// - Fetches financial account from database
// - Returns account status and basic details
//
// Returns: Club financial account information
func (s *Service) GetFinancialAccount() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second*15)
		defer cancel()

		// Validate club ID
		id := mux.Vars(r)["id"]
		oid, err := utils.ValidateObjectID(id)
		if err != nil {
			utils.HandleInvalidIDError(rw)
			s.Logger.Error("Invalid Club ID - Error: ", err.Error())
			return
		}

		// Validate user permissions
		uuid := r.Header.Get("UUID")
		member, err := s.FindMember(ctx, bson.M{"user_id": uuid, "club_id": oid})
		if err != nil {
			utils.HandleFindError(rw, err)
			s.Logger.Error("Failed to find member. Error: ", err.Error())
			return
		}
		if member.Role == string(models.OwnerMember) {
			http.Error(rw, `{"msg": "insufficient permissions"}`, http.StatusUnauthorized)
			return
		}

		// Find financial account
		account, err := s.FindFinancialAccount(ctx, bson.M{"club_id": oid})
		if err != nil {
			utils.HandleFindError(rw, err)
			s.Logger.Error("Failed to find financial account. Error: ", err.Error())
			return
		}

		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(account)
	}
}

// GetFinancialOverview provides a summary of the club's financial status
//
// - Validates club ID and user permissions
// - Retrieves current balance from Stripe
// - Gets recent transactions
// - Calculates summary statistics
//
// Returns: Financial overview with balance and recent activity
func (s *Service) GetFinancialOverview() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second*15)
		defer cancel()

		// Validate club ID
		id := mux.Vars(r)["id"]
		oid, err := utils.ValidateObjectID(id)
		if err != nil {
			utils.HandleInvalidIDError(rw)
			s.Logger.Error("Invalid Club ID - Error: ", err.Error())
			return
		}

		// Validate user permissions
		uuid := r.Header.Get("UUID")
		member, err := s.FindMember(ctx, bson.M{"user_id": uuid, "club_id": oid})
		if err != nil {
			utils.HandleFindError(rw, err)
			s.Logger.Error("Failed to find member. Error: ", err.Error())
			return
		}
		if member.Role == string(models.OwnerMember) {
			http.Error(rw, `{"msg": "insufficient permissions"}`, http.StatusUnauthorized)
			return
		}

		// Find financial account
		account, err := s.FindFinancialAccount(ctx, bson.M{"club_id": oid})
		if err != nil {
			utils.HandleFindError(rw, err)
			s.Logger.Error("Failed to find financial account. Error: ", err.Error())
			return
		}

		// Get stripe customer
		customer, err := s.StripeClient.V1Customers.Retrieve(ctx, *account.StripeAccountID, &stripe.CustomerRetrieveParams{})
		if err != nil {
			http.Error(rw, `{"msg": "something went wrong"}`, http.StatusInternalServerError)
			s.Logger.Errorf("Failed to find stripe customer data. Error: %s", err.Error())
			return
		}

		// Get transactions
		var transactions []models.ClubTransaction
		stripeCursor := s.StripeClient.V1BalanceTransactions.List(ctx, &stripe.BalanceTransactionListParams{})
		for t := range stripeCursor {
			transaction := models.ClubTransaction{
				StripeChargeID: &t.ID,
				Amount:         t.Amount,
				Type:           string(t.Type),
				Status:         string(t.Status),
				Currency:       string(t.Currency),
				CreatedAt:      bson.DateTime(t.Created),
			}

			// Handle event id
			if t.Source != nil {
				eventID := t.Source.Payout.Metadata["event_id"]
				if eventID != "" {
					eid, _ := bson.ObjectIDFromHex(eventID)
					transaction.EventID = &eid
				}
			}

			transactions = append(transactions, transaction)
		}

		balance := float64(customer.Balance)
		account.AvailableBalance = &balance
		account.RecentTransactions = &transactions

		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(account)
	}
}

// GetTransactionHistory retrieves the club's transaction history
//
// - Validates club ID and user permissions
// - Processes query parameters (limit, offset, type filter)
// - Fetches transactions from database
// - Returns paginated transaction list
//
// Returns: List of club transactions with pagination info
func (s *Service) GetTransactionHistory() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second*15)
		defer cancel()

		// Validate club ID
		id := mux.Vars(r)["id"]
		oid, err := utils.ValidateObjectID(id)
		if err != nil {
			utils.HandleInvalidIDError(rw)
			s.Logger.Error("Invalid Club ID - Error: ", err.Error())
			return
		}

		// Validate user permissions
		uuid := r.Header.Get("UUID")
		member, err := s.FindMember(ctx, bson.M{"user_id": uuid, "club_id": oid})
		if err != nil {
			utils.HandleFindError(rw, err)
			s.Logger.Error("Failed to find member. Error: ", err.Error())
			return
		}
		if member.Role == string(models.OwnerMember) {
			http.Error(rw, `{"msg": "insufficient permissions"}`, http.StatusUnauthorized)
			return
		}

		// TODO: Parse query parameters (limit, offset, type, date range)
		// TODO: Fetch transactions from database with filters

		// Placeholder response
		response := map[string]interface{}{
			"club_id":      id,
			"transactions": []interface{}{},
			"total_count":  0,
			"limit":        20,
			"offset":       0,
		}

		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(response)
	}
}

// InitiatePayout processes a withdrawal request from club balance
//
// - Validates club ID and user permissions
// - Validates payout request (amount, destination)
// - Checks available balance
// - Creates Stripe payout
// - Records transaction in database
//
// Returns: Payout confirmation details
func (s *Service) InitiatePayout() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second*15)
		defer cancel()

		// Validate club ID
		id := mux.Vars(r)["id"]
		oid, err := utils.ValidateObjectID(id)
		if err != nil {
			utils.HandleInvalidIDError(rw)
			s.Logger.Error("Invalid Club ID - Error: ", err.Error())
			return
		}

		// Validate user permissions - only owners can initiate payouts
		uuid := r.Header.Get("UUID")
		member, err := s.FindMember(ctx, bson.M{"user_id": uuid, "club_id": oid})
		if err != nil {
			utils.HandleFindError(rw, err)
			s.Logger.Error("Failed to find member. Error: ", err.Error())
			return
		}
		if member.Role != string(models.OwnerMember) {
			http.Error(rw, `{"msg": "only club owners can initiate payouts"}`, http.StatusUnauthorized)
			return
		}

		// Decode payout request
		var req models.PayoutRequest
		err = json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			s.Logger.Error("Failed to decode payout request: ", err.Error())
			http.Error(rw, `{"msg": "invalid request body"}`, http.StatusBadRequest)
			return
		}

		// Find financial account
		account, err := s.FindFinancialAccount(ctx, bson.M{"club_id": oid})
		if err != nil {
			utils.HandleFindError(rw, err)
			s.Logger.Error("Failed to find financial account. Error: ", err.Error())
			return
		}

		if account.AccountStatus != nil && *account.AccountStatus == "active" {
			http.Error(rw, `{"msg": "account not active for payouts"}`, http.StatusBadRequest)
			return
		}

		// TODO: Check available balance with Stripe
		// TODO: Create Stripe payout
		// TODO: Record transaction in database

		// Placeholder response
		response := map[string]interface{}{
			"msg":               "Payout initiated successfully",
			"payout_id":         "po_placeholder",
			"amount":            req.Amount,
			"currency":          req.Currency,
			"estimated_arrival": "2-3 business days",
		}

		rw.WriteHeader(http.StatusCreated)
		json.NewEncoder(rw).Encode(response)
	}
}

// GetPayoutHistory retrieves the club's payout history
//
// - Validates club ID and user permissions
// - Fetches payout records from database
// - Returns paginated payout list with status information
//
// Returns: List of club payouts with details
func (s *Service) GetPayoutHistory() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second*15)
		defer cancel()

		// Validate club ID
		id := mux.Vars(r)["id"]
		oid, err := utils.ValidateObjectID(id)
		if err != nil {
			utils.HandleInvalidIDError(rw)
			s.Logger.Error("Invalid Club ID - Error: ", err.Error())
			return
		}

		// Validate user permissions
		uuid := r.Header.Get("UUID")
		member, err := s.FindMember(ctx, bson.M{"user_id": uuid, "club_id": oid})
		if err != nil {
			utils.HandleFindError(rw, err)
			s.Logger.Error("Failed to find member. Error: ", err.Error())
			return
		}
		if member.Role == string(models.OwnerMember) {
			http.Error(rw, `{"msg": "insufficient permissions"}`, http.StatusUnauthorized)
			return
		}

		// TODO: Fetch payout records from database
		// TODO: Parse query parameters for pagination

		// Placeholder response
		response := map[string]interface{}{
			"club_id":     id,
			"payouts":     []interface{}{},
			"total_count": 0,
			"limit":       20,
			"offset":      0,
		}

		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(response)
	}
}

// GetCustomerSheetConfig retrieves Stripe Customer Sheet configuration for iOS client
//
// - Validates club ID and user permissions
// - Fetches club's financial account to get Stripe customer/account ID
// - Creates ephemeral key for the customer
// - Creates setup intent for payment method attachment
// - Returns configuration needed for Stripe Customer Sheet on iOS
//
// Returns: Customer ID, ephemeral key secret, and setup intent client secret
func (s *Service) GetCustomerSheetConfig() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second*15)
		defer cancel()

		// Validate club ID
		id := mux.Vars(r)["id"]
		oid, err := utils.ValidateObjectID(id)
		if err != nil {
			utils.HandleInvalidIDError(rw)
			s.Logger.Error("Invalid Club ID - Error: ", err.Error())
			return
		}

		// Validate user permissions - only club members can access this
		uuid := r.Header.Get("UUID")
		member, err := s.FindMember(ctx, bson.M{"user_id": uuid, "club_id": oid})
		if err != nil {
			utils.HandleFindError(rw, err)
			s.Logger.Error("Failed to find member. Error: ", err.Error())
			return
		}
		if member.Role != string(models.OwnerMember) {
			http.Error(rw, `{"msg": "insufficient permissions"}`, http.StatusUnauthorized)
			return
		}

		// Find financial account to get Stripe account/customer ID
		account, err := s.FindFinancialAccount(ctx, bson.M{"club_id": oid})
		if err != nil {
			utils.HandleFindError(rw, err)
			s.Logger.Error("Failed to find financial account. Error: ", err.Error())
			return
		}

		if account.AccountStatus != nil && *account.AccountStatus == "active" {
			http.Error(rw, `{"msg": "club financial account not active"}`, http.StatusBadRequest)
			return
		}

		// TODO: Create Stripe customer if not exists
		// TODO: Create ephemeral key for customer
		// TODO: Create setup intent for payment method attachment

		// Placeholder response - replace with actual Stripe API calls
		response := models.StripeCustomerSheetResponse{
			CustomerID:              "cus_placeholder_" + *account.StripeAccountID,
			EphemeralKeySecret:      "ek_test_placeholder_ephemeral_key_secret",
			SetupIntentClientSecret: "seti_placeholder_client_secret",
		}

		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(response)
	}
}
