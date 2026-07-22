package database

import (
	"context"
	"fmt"
	"olympsis-server/utils"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Sets up the auth collection with indexes for user_id lookups
func (d *Database) SetUpAuthCollections(db *mongo.Database, config *utils.CollectionsConfig) error {
	// Create auth collection if it doesn't exist
	if !d.collectionExists(db, config.AuthCollection) {
		if err := d.createCollection(db, config.AuthCollection); err != nil {
			return err
		}
	}

	d.AuthCollection = db.Collection(config.AuthCollection)

	// user_id is the primary lookup key for every auth operation
	authIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}},
			Options: options.Index().SetName("user_id_index").SetUnique(true),
		},
	}

	if err := createIndexes(d.AuthCollection, authIndexes, "auth"); err != nil {
		return err
	}

	return nil
}

// Sets up the user collection with indexes for user_id and username lookups
func (d *Database) SetUpUserCollections(db *mongo.Database, config *utils.CollectionsConfig) error {
	// Create user collection if it doesn't exist
	if !d.collectionExists(db, config.UserCollection) {
		if err := d.createCollection(db, config.UserCollection); err != nil {
			return err
		}
	}

	d.UserCollection = db.Collection(config.UserCollection)

	userIndexes := []mongo.IndexModel{
		{
			// Primary lookup for get/update/delete user data
			Keys:    bson.D{{Key: "user_id", Value: 1}},
			Options: options.Index().SetName("user_id_index").SetUnique(true),
		},
		{
			// Used for CheckUsername availability and SearchUsersByUserName
			Keys:    bson.D{{Key: "username", Value: 1}},
			Options: options.Index().SetName("username_index").SetUnique(true),
		},
		{
			// Enables text search for user discovery
			Keys:    bson.D{{Key: "username", Value: "text"}},
			Options: options.Index().SetName("username_text_index"),
		},
	}

	if err := createIndexes(d.UserCollection, userIndexes, "users"); err != nil {
		return err
	}

	return nil
}

// Sets up all of the events collections
func (d *Database) SetUpEventCollections(db *mongo.Database, config *utils.CollectionsConfig) error {

	// Collection configurations (name, whether it's a time series, and if so, what's the time field)
	collections := []struct {
		name         string
		isTimeSeries bool
		timeField    string
	}{
		{config.EventsCollection, false, ""},
		{config.EventLogsCollection, true, "timestamp"},
		{config.EventViewsCollection, true, "view_time"},
		{config.EventTeamsCollection, false, ""},
		{config.EventTeamApplicationsCollection, false, ""},
		{config.EventCommentsCollection, false, ""},
		{config.EventInvitationsCollection, false, ""},
		{config.EventParticipantsCollection, false, ""},
	}

	// Create collections if they don't exist
	for _, col := range collections {
		if !d.collectionExists(db, col.name) {
			if col.isTimeSeries {
				if err := d.createTimeSeriesCollection(db, col.name, col.timeField); err != nil {
					return err
				}
			} else {
				if err := d.createCollection(db, col.name); err != nil {
					return err
				}
			}
		}
	}

	// Assign collections to the Database struct
	d.EventsCollection = db.Collection(config.EventsCollection)
	d.EventLogsCollection = db.Collection(config.EventLogsCollection)
	d.EventViewsCollection = db.Collection(config.EventViewsCollection)
	d.EventTeamsCollection = db.Collection(config.EventTeamsCollection)
	d.EventTeamApplicationsCollection = db.Collection(config.EventTeamApplicationsCollection)
	d.EventCommentsCollection = db.Collection(config.EventCommentsCollection)
	d.EventInvitationsCollection = db.Collection(config.EventInvitationsCollection)
	d.EventParticipantsCollection = db.Collection(config.EventParticipantsCollection)

	// Create indexes for EventsCollection
	eventIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "type", Value: 1}},
			Options: options.Index().SetName("type_index"),
		},
		{
			Keys:    bson.D{{Key: "start_time", Value: 1}},
			Options: options.Index().SetName("start_time_index"),
		},
		{
			Keys:    bson.D{{Key: "sports", Value: 1}},
			Options: options.Index().SetName("sports_index"),
		},
		{
			Keys:    bson.D{{Key: "poster_id", Value: 1}},
			Options: options.Index().SetName("poster_index"),
		},
		{
			Keys: bson.D{
				{Key: "sports", Value: 1},
				{Key: "start_time", Value: 1},
			},
			Options: options.Index().SetName("sports_start_time_index"),
		},
		{
			Keys: bson.D{
				{Key: "recurrence_config.parent_event_id", Value: 1},
				{Key: "start_time", Value: 1},
			},
			Options: options.Index().SetName("parent_event_start_time_index"),
		},
		{
			Keys: bson.D{
				{Key: "visibility", Value: 1},
				{Key: "start_time", Value: 1},
			},
			Options: options.Index().SetName("visibility_start_time_index"),
		},
		{
			Keys:    bson.D{{Key: "venues.location", Value: "2dsphere"}},
			Options: options.Index().SetName("venues_location_index"),
		},
		{
			Keys:    bson.D{{Key: "title", Value: "text"}, {Key: "body", Value: "text"}},
			Options: options.Index().SetName("content_text_index"),
		},
	}

	// Create indexes for EventLogsCollection
	eventLogsIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "event_id", Value: 1}},
			Options: options.Index().SetName("event_id_index"),
		},
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}},
			Options: options.Index().SetName("user_id_index"),
		},
		{
			Keys:    bson.D{{Key: "timestamp", Value: -1}},
			Options: options.Index().SetName("timestamp_index"),
		},
		{
			Keys: bson.D{
				{Key: "event_id", Value: 1},
				{Key: "timestamp", Value: -1},
			},
			Options: options.Index().SetName("event_id_timestamp_index"),
		},
		{
			Keys:    bson.D{{Key: "action", Value: 1}},
			Options: options.Index().SetName("action_index"),
		},
	}

	// Create indexes for EventViewsCollection
	eventViewsIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "event_id", Value: 1}},
			Options: options.Index().SetName("event_id_index"),
		},
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}},
			Options: options.Index().SetName("user_id_index"),
		},
		{
			Keys:    bson.D{{Key: "view_time", Value: -1}},
			Options: options.Index().SetName("view_time_index"),
		},
		{
			Keys: bson.D{
				{Key: "event_id", Value: 1},
				{Key: "view_time", Value: -1},
			},
			Options: options.Index().SetName("event_id_view_time_index"),
		},
		{
			Keys:    bson.D{{Key: "device_info.platform", Value: 1}},
			Options: options.Index().SetName("platform_index"),
		},
		{
			Keys:    bson.D{{Key: "source", Value: 1}},
			Options: options.Index().SetName("source_index"),
		},
	}

	// Create indexes for EventTeamsCollection
	eventTeamsIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "event_id", Value: 1}},
			Options: options.Index().SetName("event_id_index"),
		},
		{
			Keys:    bson.D{{Key: "name", Value: "text"}},
			Options: options.Index().SetName("name_text_index"),
		},
		{
			Keys:    bson.D{{Key: "members.user_id", Value: 1}},
			Options: options.Index().SetName("members_user_id_index"),
		},
		{
			Keys:    bson.D{{Key: "created_at", Value: -1}},
			Options: options.Index().SetName("created_at_index"),
		},
		{
			// A user may be on at most one team per event (and not twice on the
			// same team). event_id is scalar and members.user_id is the only array
			// field, so this is a legal multikey compound unique index. The
			// application-level checks return a friendly 409 before this fires; this
			// is the DB backstop. Requires no pre-existing duplicate memberships.
			Keys: bson.D{
				{Key: "event_id", Value: 1},
				{Key: "members.user_id", Value: 1},
			},
			Options: options.Index().SetName("event_member_compound_index").SetUnique(true),
		},
	}

	// Create indexes for EventTeamApplicationsCollection
	eventTeamApplicationsIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "event_id", Value: 1}},
			Options: options.Index().SetName("event_id_index"),
		},
		{
			Keys:    bson.D{{Key: "team_id", Value: 1}},
			Options: options.Index().SetName("team_id_index"),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetName("status_index"),
		},
		{
			Keys: bson.D{
				{Key: "team_id", Value: 1},
				{Key: "status", Value: 1},
			},
			Options: options.Index().SetName("team_id_status_index"),
		},
		{ // A user may only have one application per team
			Keys: bson.D{
				{Key: "team_id", Value: 1},
				{Key: "applicant", Value: 1},
			},
			Options: options.Index().SetName("team_applicant_compound_index").SetUnique(true),
		},
	}

	// Create indexes for EventCommentsCollection
	eventCommentsIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "event_id", Value: 1}},
			Options: options.Index().SetName("event_id_index"),
		},
		{
			Keys:    bson.D{{Key: "user", Value: 1}},
			Options: options.Index().SetName("user_index"),
		},
		{
			Keys:    bson.D{{Key: "created_at", Value: -1}},
			Options: options.Index().SetName("created_at_index"),
		},
		{
			Keys: bson.D{
				{Key: "event_id", Value: 1},
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().SetName("event_id_created_at_index"),
		},
		{
			Keys:    bson.D{{Key: "text", Value: "text"}},
			Options: options.Index().SetName("text_index"),
		},
	}

	// Create indexes for EventInvitationsCollection
	eventInvitationsIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "event_id", Value: 1}},
			Options: options.Index().SetName("event_id_index"),
		},
		{
			Keys:    bson.D{{Key: "invitee_id", Value: 1}},
			Options: options.Index().SetName("invitee_id_index"),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetName("status_index"),
		},
		{
			Keys: bson.D{
				{Key: "event_id", Value: 1},
				{Key: "status", Value: 1},
			},
			Options: options.Index().SetName("event_id_status_index"),
		},
		{
			Keys: bson.D{
				{Key: "invitee_id", Value: 1},
				{Key: "status", Value: 1},
			},
			Options: options.Index().SetName("invitee_id_status_index"),
		},
	}

	// Create indexes for EventParticipantsCollection
	eventParticipantsIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "event_id", Value: 1}},
			Options: options.Index().SetName("event_id_index"),
		},
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}},
			Options: options.Index().SetName("user_index"),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetName("status_index"),
		},
		{
			Keys: bson.D{
				{Key: "event_id", Value: 1},
				{Key: "status", Value: 1},
			},
			Options: options.Index().SetName("event_id_status_index"),
		},
		{
			Keys:    bson.D{{Key: "created_at", Value: -1}},
			Options: options.Index().SetName("created_at_index"),
		},
		{ // A participant can only RSVP to an event once
			Keys: bson.D{
				{Key: "user_id", Value: 1},
				{Key: "event_id", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
	}

	// Create all indexes for each collection
	collectionsIndexes := map[string]struct {
		collection *mongo.Collection
		indexes    []mongo.IndexModel
	}{
		"events":                {d.EventsCollection, eventIndexes},
		"eventLogs":             {d.EventLogsCollection, eventLogsIndexes},
		"eventViews":            {d.EventViewsCollection, eventViewsIndexes},
		"eventTeams":            {d.EventTeamsCollection, eventTeamsIndexes},
		"eventTeamApplications": {d.EventTeamApplicationsCollection, eventTeamApplicationsIndexes},
		"eventComments":         {d.EventCommentsCollection, eventCommentsIndexes},
		"eventInvitations":      {d.EventInvitationsCollection, eventInvitationsIndexes},
		"eventParticipants":     {d.EventParticipantsCollection, eventParticipantsIndexes},
	}

	for name, info := range collectionsIndexes {
		if err := createIndexes(info.collection, info.indexes, name); err != nil {
			return err
		}
	}

	return nil
}

// Sets up the announcement collection
func (d *Database) SetUpAnnouncementCollection(db *mongo.Database, config *utils.CollectionsConfig) error {
	d.AnnouncementCollection = db.Collection(config.AnnouncementCollection)
	announceModel := mongo.IndexModel{
		Keys: bson.M{"location": "2dsphere"},
	}
	_, err := d.AnnouncementCollection.Indexes().CreateOne(context.Background(), announceModel)
	if err != nil {
		return fmt.Errorf("could not create geospatial index for announcements: %v", err)
	}

	regionModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "country", Value: 1},
			{Key: "state", Value: 1},
			{Key: "city", Value: 1},
		},
	}
	_, err = d.AnnouncementCollection.Indexes().CreateOne(context.Background(), regionModel)
	if err != nil {
		return fmt.Errorf("could not create region index for announcements: %v", err)
	}

	timeModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "start_time", Value: 1},
			{Key: "end_time", Value: 1},
		},
	}
	_, err = d.AnnouncementCollection.Indexes().CreateOne(context.Background(), timeModel)
	if err != nil {
		return fmt.Errorf("could not create time index for announcements: %v", err)
	}
	return nil
}

// Sets up all of the venue collections
func (d *Database) SetUpVenueCollections(db *mongo.Database, config *utils.CollectionsConfig) error {
	// Check if collection exists
	collectionExists := func(name string) bool {
		collections, err := db.ListCollectionNames(context.Background(), bson.M{"name": name})
		if err != nil {
			// If there's an error, assume the collection doesn't exist
			return false
		}
		return len(collections) > 0
	}

	// --- Venues Collection ---
	if !collectionExists(config.VenuesCollection) {
		err := db.CreateCollection(context.Background(), config.VenuesCollection)
		if err != nil {
			return fmt.Errorf("could not create venues collection: %v", err)
		}

		d.VenuesCollection = db.Collection(config.VenuesCollection)

		venueIndexes := []mongo.IndexModel{
			{
				Keys:    bson.M{"location": "2dsphere"},
				Options: options.Index().SetName("location_2dsphere_index"),
			},
			{
				Keys: bson.D{
					{Key: "country", Value: 1},
					{Key: "state", Value: 1},
					{Key: "city", Value: 1},
				},
				Options: options.Index().SetName("region_index"),
			},
			{
				Keys:    bson.D{{Key: "sports", Value: 1}},
				Options: options.Index().SetName("sports_index"),
			},
			{
				Keys:    bson.D{{Key: "name", Value: "text"}},
				Options: options.Index().SetName("name_text_index"),
			},
		}

		if err := createIndexes(d.VenuesCollection, venueIndexes, "venues"); err != nil {
			return fmt.Errorf("failed to create venue indexes: %v", err)
		}
	} else {
		d.VenuesCollection = db.Collection(config.VenuesCollection)
	}

	// --- Venue Units Collection ---
	if !collectionExists(config.VenueUnitsCollection) {
		err := db.CreateCollection(context.Background(), config.VenueUnitsCollection)
		if err != nil {
			return fmt.Errorf("could not create venue units collection: %v", err)
		}

		d.VenueUnitsCollection = db.Collection(config.VenueUnitsCollection)

		venueUnitIndexes := []mongo.IndexModel{
			{
				Keys:    bson.D{{Key: "venue_id", Value: 1}},
				Options: options.Index().SetName("venue_id_index"),
			},
			{
				Keys:    bson.M{"location": "2dsphere"},
				Options: options.Index().SetName("unit_location_2dsphere_index"),
			},
			{
				Keys:    bson.D{{Key: "sports", Value: 1}},
				Options: options.Index().SetName("unit_sports_index"),
			},
			{
				Keys:    bson.D{{Key: "unit_type", Value: 1}},
				Options: options.Index().SetName("unit_type_index"),
			},
		}

		if err := createIndexes(d.VenueUnitsCollection, venueUnitIndexes, "venue_units"); err != nil {
			return fmt.Errorf("failed to create venue unit indexes: %v", err)
		}
	} else {
		d.VenueUnitsCollection = db.Collection(config.VenueUnitsCollection)
	}

	// --- Transit Lines Collection ---
	if !collectionExists(config.TransitLinesCollection) {
		err := db.CreateCollection(context.Background(), config.TransitLinesCollection)
		if err != nil {
			return fmt.Errorf("could not create transit lines collection: %v", err)
		}

		d.TransitLinesCollection = db.Collection(config.TransitLinesCollection)

		transitLineIndexes := []mongo.IndexModel{
			{
				// Compound index for querying by system + name (the primary query pattern)
				Keys: bson.D{
					{Key: "system", Value: 1},
					{Key: "name", Value: 1},
				},
				Options: options.Index().SetName("system_name_index"),
			},
			{
				Keys:    bson.D{{Key: "locality", Value: 1}},
				Options: options.Index().SetName("locality_index"),
			},
			{
				Keys:    bson.D{{Key: "type", Value: 1}},
				Options: options.Index().SetName("transit_type_index"),
			},
		}

		if err := createIndexes(d.TransitLinesCollection, transitLineIndexes, "transit_lines"); err != nil {
			return fmt.Errorf("failed to create transit line indexes: %v", err)
		}
	} else {
		d.TransitLinesCollection = db.Collection(config.TransitLinesCollection)
	}

	return nil
}

// Sets up all of the club collections
func (d *Database) SetUpClubCollections(db *mongo.Database, config *utils.CollectionsConfig) error {
	// Check if collection exists
	collectionExists := func(name string) bool {
		collections, err := db.ListCollectionNames(context.Background(), bson.M{"name": name})
		if err != nil {
			// If there's an error, assume the collection doesn't exist
			return false
		}
		return len(collections) > 0
	}

	// Set up Club Collection
	if !collectionExists(config.ClubCollection) {
		err := db.CreateCollection(context.Background(), config.ClubCollection)
		if err != nil {
			return fmt.Errorf("could not create club collection: %v", err)
		}

		d.ClubCollection = db.Collection(config.ClubCollection)

		// Define all club indexes
		clubIndexes := []mongo.IndexModel{
			{
				Keys:    bson.D{{Key: "name", Value: "text"}},
				Options: options.Index().SetName("name_text_index"),
			},
			{
				Keys: bson.D{
					{Key: "country", Value: 1},
					{Key: "state", Value: 1},
					{Key: "city", Value: 1},
				},
				Options: options.Index().SetName("region_index"),
			},
			{
				Keys:    bson.D{{Key: "sports", Value: 1}},
				Options: options.Index().SetName("sports_index"),
			},
			{
				Keys:    bson.D{{Key: "tags", Value: 1}},
				Options: options.Index().SetName("tags_index"),
			},
			{
				Keys:    bson.D{{Key: "visibility", Value: 1}},
				Options: options.Index().SetName("visibility_index"),
			},
			{
				Keys:    bson.D{{Key: "is_verified", Value: 1}},
				Options: options.Index().SetName("verified_index"),
			},
			{
				Keys:    bson.D{{Key: "parent_id", Value: 1}},
				Options: options.Index().SetName("parent_id_index"),
			},
		}

		// Create indexes using the safe method
		if err := createIndexes(d.ClubCollection, clubIndexes, "clubs"); err != nil {
			return fmt.Errorf("failed to create club indexes: %v", err)
		}
	} else {
		d.ClubCollection = db.Collection(config.ClubCollection)
	}

	// Set up Club Invitation Collection
	if !collectionExists(config.ClubInvitationCollection) {
		err := db.CreateCollection(context.Background(), config.ClubInvitationCollection)
		if err != nil {
			return fmt.Errorf("could not create club invitation collection: %v", err)
		}

		d.ClubInvitationCollection = db.Collection(config.ClubInvitationCollection)

		// Define club invitation indexes
		invitationIndexes := []mongo.IndexModel{
			{
				Keys:    bson.D{{Key: "club_id", Value: 1}},
				Options: options.Index().SetName("club_id_index"),
			},
			{
				Keys:    bson.D{{Key: "email", Value: 1}},
				Options: options.Index().SetName("email_index"),
			},
			{
				Keys:    bson.D{{Key: "status", Value: 1}},
				Options: options.Index().SetName("status_index"),
			},
		}

		// Create indexes using the safe method
		if err := createIndexes(d.ClubInvitationCollection, invitationIndexes, "club_invitations"); err != nil {
			return fmt.Errorf("failed to create club invitation indexes: %v", err)
		}
	} else {
		d.ClubInvitationCollection = db.Collection(config.ClubInvitationCollection)
	}

	// Set up Club Members Collection
	if !collectionExists(config.ClubMembersCollection) {
		err := db.CreateCollection(context.Background(), config.ClubMembersCollection)
		if err != nil {
			return fmt.Errorf("could not create club members collection: %v", err)
		}

		d.ClubMembersCollection = db.Collection(config.ClubMembersCollection)

		// Define club members indexes
		membersIndexes := []mongo.IndexModel{
			{
				Keys:    bson.D{{Key: "club_id", Value: 1}},
				Options: options.Index().SetName("club_id_index"),
			},
			{
				Keys:    bson.D{{Key: "user_id", Value: 1}},
				Options: options.Index().SetName("user_id_index"),
			},
			{
				Keys: bson.D{
					{Key: "club_id", Value: 1},
					{Key: "user_id", Value: 1},
				},
				Options: options.Index().SetName("club_user_compound_index").SetUnique(true),
			},
			{
				Keys:    bson.D{{Key: "role", Value: 1}},
				Options: options.Index().SetName("role_index"),
			},
		}

		// Create indexes using the safe method
		if err := createIndexes(d.ClubMembersCollection, membersIndexes, "club_members"); err != nil {
			return fmt.Errorf("failed to create club members indexes: %v", err)
		}
	} else {
		d.ClubMembersCollection = db.Collection(config.ClubMembersCollection)
	}

	// Set up Club Application Collection
	if !collectionExists(config.ClubApplicationCollection) {
		err := db.CreateCollection(context.Background(), config.ClubApplicationCollection)
		if err != nil {
			return fmt.Errorf("could not create club application collection: %v", err)
		}

		d.ClubApplicationCollection = db.Collection(config.ClubApplicationCollection)

		// Define club application indexes
		applicationIndexes := []mongo.IndexModel{
			{
				Keys:    bson.D{{Key: "club_id", Value: 1}},
				Options: options.Index().SetName("club_id_index"),
			},
			{
				Keys:    bson.D{{Key: "user_id", Value: 1}},
				Options: options.Index().SetName("user_id_index"),
			},
			{
				Keys: bson.D{
					{Key: "club_id", Value: 1},
					{Key: "user_id", Value: 1},
				},
				Options: options.Index().SetName("club_user_compound_index").SetUnique(true),
			},
			{
				Keys:    bson.D{{Key: "status", Value: 1}},
				Options: options.Index().SetName("status_index"),
			},
		}

		// Create indexes using the safe method
		if err := createIndexes(d.ClubApplicationCollection, applicationIndexes, "club_applications"); err != nil {
			return fmt.Errorf("failed to create club application indexes: %v", err)
		}
	} else {
		d.ClubApplicationCollection = db.Collection(config.ClubApplicationCollection)
	}

	// Set up Club Financial Accounts Collection
	if !collectionExists(config.ClubFinancialAccountsCollection) {
		err := db.CreateCollection(context.Background(), config.ClubFinancialAccountsCollection)
		if err != nil {
			return fmt.Errorf("could not create club financial accounts collection: %v", err)
		}
		d.ClubFinancialAccountsCollection = db.Collection(config.ClubFinancialAccountsCollection)
		// Define club financial accounts indexes
		financialAccountIndexes := []mongo.IndexModel{
			{
				Keys:    bson.D{{Key: "club_id", Value: 1}},
				Options: options.Index().SetName("club_id_index").SetUnique(true),
			},
			{
				Keys:    bson.D{{Key: "stripe_account_id", Value: 1}},
				Options: options.Index().SetName("stripe_account_id_index").SetUnique(true),
			},
			{
				Keys:    bson.D{{Key: "account_status", Value: 1}},
				Options: options.Index().SetName("account_status_index"),
			},
		}
		// Create indexes using the safe method
		if err := createIndexes(d.ClubFinancialAccountsCollection, financialAccountIndexes, "club_financial_accounts"); err != nil {
			return fmt.Errorf("failed to create club financial accounts indexes: %v", err)
		}
	} else {
		d.ClubFinancialAccountsCollection = db.Collection(config.ClubFinancialAccountsCollection)
	}

	// Set up Club Transactions Collection
	if !collectionExists(config.ClubTransactionsCollection) {
		err := db.CreateCollection(context.Background(), config.ClubTransactionsCollection)
		if err != nil {
			return fmt.Errorf("could not create club transactions collection: %v", err)
		}
		d.ClubTransactionsCollection = db.Collection(config.ClubTransactionsCollection)
		// Define club transactions indexes
		transactionIndexes := []mongo.IndexModel{
			{
				Keys:    bson.D{{Key: "club_id", Value: 1}},
				Options: options.Index().SetName("club_id_index"),
			},
			{
				Keys:    bson.D{{Key: "event_id", Value: 1}},
				Options: options.Index().SetName("event_id_index"),
			},
			{
				Keys:    bson.D{{Key: "type", Value: 1}},
				Options: options.Index().SetName("type_index"),
			},
			{
				Keys:    bson.D{{Key: "status", Value: 1}},
				Options: options.Index().SetName("status_index"),
			},
			{
				Keys:    bson.D{{Key: "created_at", Value: -1}},
				Options: options.Index().SetName("created_at_desc_index"),
			},
			{
				Keys:    bson.D{{Key: "stripe_charge_id", Value: 1}},
				Options: options.Index().SetName("stripe_charge_id_index").SetSparse(true),
			},
			{
				Keys:    bson.D{{Key: "stripe_payout_id", Value: 1}},
				Options: options.Index().SetName("stripe_payout_id_index").SetSparse(true),
			},
		}
		// Create indexes using the safe method
		if err := createIndexes(d.ClubTransactionsCollection, transactionIndexes, "club_transactions"); err != nil {
			return fmt.Errorf("failed to create club transactions indexes: %v", err)
		}
	} else {
		d.ClubTransactionsCollection = db.Collection(config.ClubTransactionsCollection)
	}

	return nil
}

// Sets up all of the organization collections
func (d *Database) SetUpOrganizationCollections(db *mongo.Database, config *utils.CollectionsConfig) error {
	// Check if collection exists
	collectionExists := func(name string) bool {
		collections, err := db.ListCollectionNames(context.Background(), bson.M{"name": name})
		if err != nil {
			// If there's an error, assume the collection doesn't exist
			return false
		}
		return len(collections) > 0
	}

	// Set up Organization Collection
	if !collectionExists(config.OrgCollection) {
		err := db.CreateCollection(context.Background(), config.OrgCollection)
		if err != nil {
			return fmt.Errorf("could not create organization collection: %v", err)
		}

		d.OrgCollection = db.Collection(config.OrgCollection)

		// Define all organization indexes
		orgIndexes := []mongo.IndexModel{
			{
				Keys:    bson.D{{Key: "name", Value: "text"}},
				Options: options.Index().SetName("name_text_index"),
			},
			{
				Keys: bson.D{
					{Key: "country", Value: 1},
					{Key: "state", Value: 1},
					{Key: "city", Value: 1},
				},
				Options: options.Index().SetName("region_index"),
			},
			{
				Keys:    bson.D{{Key: "sports", Value: 1}},
				Options: options.Index().SetName("sports_index"),
			},
			{
				Keys:    bson.D{{Key: "tags", Value: 1}},
				Options: options.Index().SetName("tags_index"),
			},
			{
				Keys:    bson.D{{Key: "is_verified", Value: 1}},
				Options: options.Index().SetName("verified_index"),
			},
		}

		// Create indexes using the safe method
		if err := createIndexes(d.OrgCollection, orgIndexes, "organizations"); err != nil {
			return fmt.Errorf("failed to create organization indexes: %v", err)
		}
	} else {
		d.OrgCollection = db.Collection(config.OrgCollection)
	}

	// Set up Organization Invitation Collection
	if !collectionExists(config.OrgInvitationCollection) {
		err := db.CreateCollection(context.Background(), config.OrgInvitationCollection)
		if err != nil {
			return fmt.Errorf("could not create organization invitation collection: %v", err)
		}

		d.OrgInvitationCollection = db.Collection(config.OrgInvitationCollection)

		// Define organization invitation indexes
		invitationIndexes := []mongo.IndexModel{
			{
				Keys:    bson.D{{Key: "org_id", Value: 1}},
				Options: options.Index().SetName("org_id_index"),
			},
			{
				Keys:    bson.D{{Key: "email", Value: 1}},
				Options: options.Index().SetName("email_index"),
			},
			{
				Keys:    bson.D{{Key: "status", Value: 1}},
				Options: options.Index().SetName("status_index"),
			},
		}

		// Create indexes using the safe method
		if err := createIndexes(d.OrgInvitationCollection, invitationIndexes, "org_invitations"); err != nil {
			return fmt.Errorf("failed to create organization invitation indexes: %v", err)
		}
	} else {
		d.OrgInvitationCollection = db.Collection(config.OrgInvitationCollection)
	}

	// Set up Organization Application Collection
	if !collectionExists(config.OrgApplicationCollection) {
		err := db.CreateCollection(context.Background(), config.OrgApplicationCollection)
		if err != nil {
			return fmt.Errorf("could not create organization application collection: %v", err)
		}

		d.OrgApplicationCollection = db.Collection(config.OrgApplicationCollection)

		// Define organization application indexes
		applicationIndexes := []mongo.IndexModel{
			{
				Keys:    bson.D{{Key: "org_id", Value: 1}},
				Options: options.Index().SetName("org_id_index"),
			},
			{
				Keys:    bson.D{{Key: "user_id", Value: 1}},
				Options: options.Index().SetName("user_id_index"),
			},
			{
				Keys: bson.D{
					{Key: "org_id", Value: 1},
					{Key: "user_id", Value: 1},
				},
				Options: options.Index().SetName("org_user_compound_index").SetUnique(true),
			},
			{
				Keys:    bson.D{{Key: "status", Value: 1}},
				Options: options.Index().SetName("status_index"),
			},
		}

		// Create indexes using the safe method
		if err := createIndexes(d.OrgApplicationCollection, applicationIndexes, "org_applications"); err != nil {
			return fmt.Errorf("failed to create organization application indexes: %v", err)
		}
	} else {
		d.OrgApplicationCollection = db.Collection(config.OrgApplicationCollection)
	}

	// Set up Organization Members Collection
	if !collectionExists(config.OrganizationMembersCollection) {
		err := db.CreateCollection(context.Background(), config.OrganizationMembersCollection)
		if err != nil {
			return fmt.Errorf("could not create organization members collection: %v", err)
		}

		d.OrganizationMembersCollection = db.Collection(config.OrganizationMembersCollection)

		// Define organization members indexes
		membersIndexes := []mongo.IndexModel{
			{
				Keys:    bson.D{{Key: "org_id", Value: 1}},
				Options: options.Index().SetName("org_id_index"),
			},
			{
				Keys:    bson.D{{Key: "user_id", Value: 1}},
				Options: options.Index().SetName("user_id_index"),
			},
			{
				Keys: bson.D{
					{Key: "org_id", Value: 1},
					{Key: "user_id", Value: 1},
				},
				Options: options.Index().SetName("org_user_compound_index").SetUnique(true),
			},
			{
				Keys:    bson.D{{Key: "role", Value: 1}},
				Options: options.Index().SetName("role_index"),
			},
		}

		// Create indexes using the safe method
		if err := createIndexes(d.OrganizationMembersCollection, membersIndexes, "org_members"); err != nil {
			return fmt.Errorf("failed to create organization members indexes: %v", err)
		}
	} else {
		d.OrganizationMembersCollection = db.Collection(config.OrganizationMembersCollection)
	}

	return nil
}

// Sets up all of the post collections
func (d *Database) SetUpPostCollections(db *mongo.Database, config *utils.CollectionsConfig) error {
	collectionExists := func(name string) bool {
		collections, err := db.ListCollectionNames(context.Background(), bson.M{"name": name})
		if err != nil {
			// If there's an error, assume the collection doesn't exist
			return false
		}
		return len(collections) > 0
	}

	// Set up Posts Collection
	if !collectionExists(config.PostCollection) {
		err := db.CreateCollection(context.Background(), config.PostCollection)
		if err != nil {
			return fmt.Errorf("could not create posts collection: %v", err)
		}

		d.PostsCollection = db.Collection(config.PostCollection)

		// Define all post indexes
		postIndexes := []mongo.IndexModel{
			{
				Keys:    bson.D{{Key: "poster._id", Value: 1}},
				Options: options.Index().SetName("poster_id_index"),
			},
			{
				Keys:    bson.D{{Key: "type", Value: 1}},
				Options: options.Index().SetName("type_index"),
			},
			{
				Keys:    bson.D{{Key: "group_id", Value: 1}},
				Options: options.Index().SetName("group_id_index"),
			},
			{
				Keys:    bson.D{{Key: "event_id", Value: 1}},
				Options: options.Index().SetName("event_id_index"),
			},
			{
				Keys:    bson.D{{Key: "body", Value: "text"}},
				Options: options.Index().SetName("body_text_index"),
			},
			{
				Keys:    bson.D{{Key: "created_at", Value: -1}},
				Options: options.Index().SetName("created_at_index"),
			},
			{
				Keys:    bson.D{{Key: "is_sensitive", Value: 1}},
				Options: options.Index().SetName("is_sensitive_index"),
			},
		}

		// Create indexes using the safe method
		if err := createIndexes(d.PostsCollection, postIndexes, "posts"); err != nil {
			return fmt.Errorf("failed to create post indexes: %v", err)
		}
	} else {
		d.PostsCollection = db.Collection(config.PostCollection)
	}

	// Set up Post Reactions Collection
	if !collectionExists(config.PostReactionsCollection) {
		err := db.CreateCollection(context.Background(), config.PostReactionsCollection)
		if err != nil {
			return fmt.Errorf("could not create post reactions collection: %v", err)
		}

		d.PostReactionsCollection = db.Collection(config.PostReactionsCollection)

		// Define post reactions indexes
		reactionIndexes := []mongo.IndexModel{
			{
				Keys:    bson.D{{Key: "post_id", Value: 1}},
				Options: options.Index().SetName("post_id_index"),
			},
			{
				Keys:    bson.D{{Key: "user_id", Value: 1}},
				Options: options.Index().SetName("user_id_index"),
			},
			{
				Keys: bson.D{
					{Key: "post_id", Value: 1},
					{Key: "user_id", Value: 1},
				},
				Options: options.Index().SetName("post_user_compound_index").SetUnique(true),
			},
			{
				Keys:    bson.D{{Key: "created_at", Value: -1}},
				Options: options.Index().SetName("created_at_index"),
			},
		}

		// Create indexes using the safe method
		if err := createIndexes(d.PostReactionsCollection, reactionIndexes, "post_reactions"); err != nil {
			return fmt.Errorf("failed to create post reactions indexes: %v", err)
		}
	} else {
		d.PostReactionsCollection = db.Collection(config.PostReactionsCollection)
	}

	// Set up Post Comments Collection
	if !collectionExists(config.PostCommentsCollection) {
		err := db.CreateCollection(context.Background(), config.PostCommentsCollection)
		if err != nil {
			return fmt.Errorf("could not create post comments collection: %v", err)
		}

		d.PostCommentsCollection = db.Collection(config.PostCommentsCollection)

		// Define post comments indexes
		commentsIndexes := []mongo.IndexModel{
			{
				Keys:    bson.D{{Key: "post_id", Value: 1}},
				Options: options.Index().SetName("post_id_index"),
			},
			{
				Keys:    bson.D{{Key: "user_id", Value: 1}},
				Options: options.Index().SetName("user_id_index"),
			},
			{
				Keys:    bson.D{{Key: "created_at", Value: -1}},
				Options: options.Index().SetName("created_at_index"),
			},
			{
				Keys:    bson.D{{Key: "text", Value: "text"}},
				Options: options.Index().SetName("text_search_index"),
			},
		}

		// Create indexes using the safe method
		if err := createIndexes(d.PostCommentsCollection, commentsIndexes, "post_comments"); err != nil {
			return fmt.Errorf("failed to create post comments indexes: %v", err)
		}
	} else {
		d.PostCommentsCollection = db.Collection(config.PostCommentsCollection)
	}

	return nil
}

// Sets up all of the Report collections
func (d *Database) SetUpReportCollections(db *mongo.Database, config *utils.CollectionsConfig) error {
	d.BugReportCollection = db.Collection(config.BugReportCollection)
	d.PostReportCollection = db.Collection(config.PostReportCollection)
	d.VenueReportCollection = db.Collection(config.VenueReportCollection)
	d.EventReportCollection = db.Collection(config.EventReportCollection)
	d.MemberReportCollection = db.Collection(config.MemberReportCollection)
	return nil
}

// Sets up all of the locale collections
func (d *Database) SetUpLocaleCollections(db *mongo.Database, config *utils.CollectionsConfig, dbConfig *utils.DatabaseConfig) error {
	localeDB := d.Client.Database(dbConfig.LocaleName)
	d.CountriesCollection = localeDB.Collection(config.CountriesCollection)
	d.AdminAreasCollection = localeDB.Collection(config.AdminAreasCollection)
	d.SubAdminAreasCollection = localeDB.Collection(config.SubAdminAreasCollection)
	return nil
}

// Sets up the application configuration collections (tags and sports)
func (d *Database) SetUpAppConfigCollections(db *mongo.Database, config *utils.CollectionsConfig) error {

	// Set up Tags Collection
	if !d.collectionExists(db, config.TagsCollections) {
		if err := d.createCollection(db, config.TagsCollections); err != nil {
			return err
		}
	}

	d.TagsCollection = db.Collection(config.TagsCollections)

	// Index on name for lookups, and on is_enabled to filter active tags
	tagsIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "name", Value: 1}},
			Options: options.Index().SetName("name_index").SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "is_enabled", Value: 1}},
			Options: options.Index().SetName("is_enabled_index"),
		},
	}

	if err := createIndexes(d.TagsCollection, tagsIndexes, "tags"); err != nil {
		return err
	}

	// Set up Sports Collection
	if !d.collectionExists(db, config.SportsCollection) {
		if err := d.createCollection(db, config.SportsCollection); err != nil {
			return err
		}
	}

	d.SportsCollection = db.Collection(config.SportsCollection)

	// Index on name for lookups, and on is_enabled to filter active sports
	sportsIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "name", Value: 1}},
			Options: options.Index().SetName("name_index").SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "is_enabled", Value: 1}},
			Options: options.Index().SetName("is_enabled_index"),
		},
	}

	if err := createIndexes(d.SportsCollection, sportsIndexes, "sports"); err != nil {
		return err
	}

	return nil
}

// Sets up notification collections
func (d *Database) SetUpNotificationsCollections(db *mongo.Database, config *utils.CollectionsConfig) error {
	d.PushNotificationsCollection = db.Collection(config.NotificationsCollection)
	d.NotificationLogsCollection = db.Collection(config.NotificationLogsCollection)
	d.UserNotificationsCollection = db.Collection(config.UserNotificationsCollection)
	d.NotificationTopicsCollection = db.Collection(config.NotificationTopicsCollection)
	return nil
}
