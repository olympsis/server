package database

import (
	"context"
	"errors"
	"fmt"
	"olympsis-server/utils"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type Database struct {
	Logger      *logrus.Logger
	Client      *mongo.Client
	NotifClient *mongo.Client

	AnnouncementCol *mongo.Collection

	AuthCol     *mongo.Collection
	UserCol     *mongo.Collection
	ClubCol     *mongo.Collection
	OrgCol      *mongo.Collection
	EventCol    *mongo.Collection
	VenueCol    *mongo.Collection
	PostCol     *mongo.Collection
	ClubInvCol  *mongo.Collection
	CommentsCol *mongo.Collection

	ClubApplicationCol *mongo.Collection
	OrgApplicationCol  *mongo.Collection

	EventInvitationCol *mongo.Collection
	ClubInvitationCol  *mongo.Collection
	OrgInvitationCol   *mongo.Collection

	BugReportCol    *mongo.Collection
	PostReportCol   *mongo.Collection
	MemberReportCol *mongo.Collection
	VenueReportCol  *mongo.Collection
	EventReportCol  *mongo.Collection

	CountriesCol     *mongo.Collection
	AdminAreasCol    *mongo.Collection
	SubAdminAreasCol *mongo.Collection
}

func NewDatabase(l *logrus.Logger) *Database {
	return &Database{Logger: l}
}

func (d *Database) EstablishConnection(config *utils.ServerConfig) {

	d.Logger.Info("Connecting to Database...")

	/*
		Connect to Mongo Database
	*/
	dbConfig := utils.GetDatabaseConfig()
	collectionConfig := utils.GetCollectionsConfig()

	switch config.Mode {
	case "PRODUCTION":
		opts := options.Client().ApplyURI(`mongodb+srv://` + dbConfig.User + `:` + dbConfig.Password + `@` + dbConfig.Address + `/?retryWrites=true&w=majority`)
		client, err := mongo.Connect(context.Background(), opts)
		if err != nil {
			d.Logger.Fatal("Failed to connect to Database: " + err.Error())
		}

		err = client.Ping(context.Background(), readpref.Primary())
		if err != nil {
			d.Logger.Fatal("Failed to connect to Database: " + err.Error())
		}

		d.Client = client
	default:
		opts := options.Client().ApplyURI(`mongodb://` + dbConfig.User + `:` + dbConfig.Password + `@` + dbConfig.Address + `/?retryWrites=true&w=majority`)
		client, err := mongo.Connect(context.Background(), opts)
		if err != nil {
			d.Logger.Fatal("Failed to connect to Database: " + err.Error())
		}

		err = client.Ping(context.Background(), readpref.Primary())
		if err != nil {
			d.Logger.Fatal("Failed to connect to Database: " + err.Error())
		}

		d.Client = client
	}

	d.SetUpCollections(&dbConfig, &collectionConfig)
	d.Logger.Info("Database connection successful")
}

func (d *Database) SetUpCollections(config *utils.DatabaseConfig, collectionConfig *utils.CollectionsConfig) error {
	database := d.Client.Database(config.Name)

	// ANNOUNCEMENT COLLECTION
	err := d.SetUpAnnouncementCollection(database, collectionConfig)
	if err != nil {
		return err
	}

	d.AuthCol = database.Collection(collectionConfig.AuthCollection)
	d.UserCol = database.Collection(collectionConfig.UserCollection)
	d.ClubCol = database.Collection(collectionConfig.ClubCollection)
	d.OrgCol = database.Collection(collectionConfig.OrgCollection)

	err = d.SetUpEventCollection(database, collectionConfig)
	if err != nil {
		return err
	}

	err = d.SetUpVenueCollection(database, collectionConfig)
	if err != nil {
		return err
	}

	d.PostCol = database.Collection(collectionConfig.PostCollection)

	d.ClubInvCol = database.Collection(collectionConfig.ClubInvitationCollection)
	d.CommentsCol = database.Collection(collectionConfig.CommentCollection)

	d.ClubApplicationCol = database.Collection(collectionConfig.ClubApplicationCollection)
	d.OrgApplicationCol = database.Collection(collectionConfig.OrgApplicationCollection)

	d.EventInvitationCol = database.Collection(collectionConfig.EventInvitationCollection)
	d.ClubInvitationCol = database.Collection(collectionConfig.ClubInvitationCollection)
	d.OrgInvitationCol = database.Collection(collectionConfig.OrgInvitationCollection)

	d.BugReportCol = database.Collection(collectionConfig.BugReportCollection)
	d.PostReportCol = database.Collection(collectionConfig.PostReportCollection)
	d.VenueReportCol = database.Collection(collectionConfig.VenueReportCollection)
	d.EventReportCol = database.Collection(collectionConfig.EventReportCollection)
	d.MemberReportCol = database.Collection(collectionConfig.MemberReportCollection)

	localeDB := d.Client.Database(config.LocaleName)
	d.CountriesCol = localeDB.Collection(collectionConfig.CountriesCollection)
	d.AdminAreasCol = localeDB.Collection(collectionConfig.AdminAreasCollection)
	d.SubAdminAreasCol = localeDB.Collection(collectionConfig.SubAdminAreasCollection)

	return nil
}

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

func (d *Database) SetUpEventCollection(db *mongo.Database, config *utils.CollectionsConfig) error {
	d.EventCol = db.Collection(config.EventCollection)

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
