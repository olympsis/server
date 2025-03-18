package database

import (
	"context"
	"errors"
	"fmt"
	"olympsis-server/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Sets up the announcement collection
func (d *Database) SetUpAnnouncementCollection(db *mongo.Database, config *utils.CollectionsConfig) error {
	d.AnnouncementCol = db.Collection(config.AnnouncementCollection)
	announceModel := mongo.IndexModel{
		Keys: bson.M{"location": "2dsphere"},
	}
	_, err := d.AnnouncementCol.Indexes().CreateOne(context.Background(), announceModel)
	if err != nil {
		return errors.New(fmt.Sprintf("Could not create geospatial index for announcements: %v", err))
	}

	regionModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "country", Value: 1},
			{Key: "state", Value: 1},
			{Key: "city", Value: 1},
		},
	}
	_, err = d.AnnouncementCol.Indexes().CreateOne(context.Background(), regionModel)
	if err != nil {
		return errors.New(fmt.Sprintf("Could not create region index for announcements: %v", err))
	}

	timeModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "start_time", Value: 1},
			{Key: "end_time", Value: 1},
		},
	}
	_, err = d.AnnouncementCol.Indexes().CreateOne(context.Background(), timeModel)
	if err != nil {
		return errors.New(fmt.Sprintf("Could not create time index for announcements: %v", err))
	}
	return nil
}

// Sets up the collections associated with user data
func (d *Database) SetUpUserCollections(db *mongo.Database, config *utils.CollectionsConfig) error {
	d.AuthCol = db.Collection(config.AuthCollection)
	d.UserCol = db.Collection(config.UserCollection)
	return nil
}

// Sets up all of the events collections
func (d *Database) SetUpEventCollections(db *mongo.Database, config *utils.CollectionsConfig) error {
	d.EventCol = db.Collection(config.EventCollection)
	d.EventInvitationCol = db.Collection(config.EventInvitationCollection)

	// Create indexes in a more structured way with error handling
	indexes := []mongo.IndexModel{
		// 1. Basic Indexes
		{
			Keys:    bson.D{{Key: "start_time", Value: 1}},
			Options: options.Index().SetName("start_time_index"),
		},
		{
			Keys:    bson.D{{Key: "sports", Value: 1}},
			Options: options.Index().SetName("sports_index"),
		},
		{
			Keys:    bson.D{{Key: "poster", Value: 1}},
			Options: options.Index().SetName("poster_index"),
		},

		// 2. Compound Indexes
		{
			Keys: bson.D{
				{Key: "sports", Value: 1},
				{Key: "start_time", Value: 1},
			},
			Options: options.Index().SetName("sports_start_time_index"),
		},
		{
			Keys: bson.D{
				{Key: "parent_event_id", Value: 1},
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

		// 3. Geospatial Index
		{
			Keys:    bson.D{{Key: "venues.location", Value: "2dsphere"}},
			Options: options.Index().SetName("venues_location_index"),
		},

		// 4. Specialized Indexes
		{
			Keys:    bson.D{{Key: "participants.uuid", Value: 1}},
			Options: options.Index().SetName("participants_uuid_index"),
		},
	}

	// Create all indexes
	_, err := d.EventCol.Indexes().CreateMany(context.Background(), indexes)
	if err != nil {
		return fmt.Errorf("could not create indexes for events: %v", err)
	}

	return nil
}

// Sets up all of the venue collections
func (d *Database) SetUpVenueCollection(db *mongo.Database, config *utils.CollectionsConfig) error {
	d.VenueCol = db.Collection(config.VenueCollection)
	geoModel := mongo.IndexModel{
		Keys: bson.M{"location": "2dsphere"},
	}
	_, err := d.VenueCol.Indexes().CreateOne(context.Background(), geoModel)
	if err != nil {
		return errors.New(fmt.Sprintf("Could not create geospatial index for venues: %v", err))
	}

	regionModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "country", Value: 1},
			{Key: "state", Value: 1},
			{Key: "city", Value: 1},
		},
	}
	_, err = d.VenueCol.Indexes().CreateOne(context.Background(), regionModel)
	if err != nil {
		return errors.New(fmt.Sprintf("Could not create region index for venues: %v", err))
	}
	return nil
}

// Sets up all of the club collections
func (d *Database) SetUpClubCollections(db *mongo.Database, config *utils.CollectionsConfig) error {
	d.ClubCol = db.Collection(config.ClubCollection)
	d.ClubInvitationCol = db.Collection(config.ClubInvitationCollection)
	d.ClubApplicationCol = db.Collection(config.ClubApplicationCollection)
	return nil
}

// Sets up all of the organization collections
func (d *Database) SetUpOrganizationCollections(db *mongo.Database, config *utils.CollectionsConfig) error {
	d.OrgCol = db.Collection(config.OrgCollection)
	d.OrgInvitationCol = db.Collection(config.OrgInvitationCollection)
	d.OrgApplicationCol = db.Collection(config.OrgApplicationCollection)
	return nil
}

// Sets up all of the post collections
func (d *Database) SetUpPostCollections(db *mongo.Database, config *utils.CollectionsConfig) error {
	d.PostCol = db.Collection(config.PostCollection)
	d.CommentsCol = db.Collection(config.CommentCollection)
	return nil
}

// Sets up all of the Report collections
func (d *Database) SetUpReportCollections(db *mongo.Database, config *utils.CollectionsConfig) error {
	d.BugReportCol = db.Collection(config.BugReportCollection)
	d.PostReportCol = db.Collection(config.PostReportCollection)
	d.VenueReportCol = db.Collection(config.VenueReportCollection)
	d.EventReportCol = db.Collection(config.EventReportCollection)
	d.MemberReportCol = db.Collection(config.MemberReportCollection)
	return nil
}

// Sets up all of the locale collections
func (d *Database) SetUpLocaleCollections(db *mongo.Database, config *utils.CollectionsConfig, dbConfig *utils.DatabaseConfig) error {
	localeDB := d.Client.Database(dbConfig.LocaleName)
	d.CountriesCol = localeDB.Collection(config.CountriesCollection)
	d.AdminAreasCol = localeDB.Collection(config.AdminAreasCollection)
	d.SubAdminAreasCol = localeDB.Collection(config.SubAdminAreasCollection)
	return nil
}

// Sets up the application configuration collections
func (d *Database) SetUpAppConfigCollections(db *mongo.Database, config *utils.CollectionsConfig) error {
	d.TagsCollection = db.Collection(config.TagsCollections)
	d.SportsCollection = db.Collection(config.SportsCollection)
	return nil
}
