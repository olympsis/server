package database

import (
	"context"
	"fmt"
	"olympsis-server/utils"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type Database struct {
	Logger      *logrus.Logger
	Client      *mongo.Client
	NotifClient *mongo.Client

	AnnouncementCol *mongo.Collection

	AuthCol          *mongo.Collection
	UserCol          *mongo.Collection
	ClubCol          *mongo.Collection
	OrgCol           *mongo.Collection
	VenuesCollection *mongo.Collection
	PostCol          *mongo.Collection
	CommentsCol      *mongo.Collection

	EventsCollection                    *mongo.Collection
	EventLogsCollection                 *mongo.Collection
	EventViewsCollection                *mongo.Collection
	EventTeamsCollection                *mongo.Collection
	EventCommentsCollection             *mongo.Collection
	EventInvitationsCollection          *mongo.Collection
	EventParticipantsCollection         *mongo.Collection
	EventTeamsWaitlistCollection        *mongo.Collection
	EventParticipantsWaitlistCollection *mongo.Collection

	ClubApplicationCol *mongo.Collection
	OrgApplicationCol  *mongo.Collection

	ClubInvitationCol *mongo.Collection
	OrgInvitationCol  *mongo.Collection

	BugReportCol    *mongo.Collection
	PostReportCol   *mongo.Collection
	MemberReportCol *mongo.Collection
	VenueReportCol  *mongo.Collection
	EventReportCol  *mongo.Collection

	CountriesCol     *mongo.Collection
	AdminAreasCol    *mongo.Collection
	SubAdminAreasCol *mongo.Collection

	TagsCollection   *mongo.Collection
	SportsCollection *mongo.Collection
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

	err := d.SetUpCollections(&dbConfig, &collectionConfig)
	if err != nil {
		panic(fmt.Sprintf("Failed to set up database collections. Error: %s", err.Error()))
	}
	d.Logger.Info("Database connection successful")
}

// Sets up all of the database collections
func (d *Database) SetUpCollections(config *utils.DatabaseConfig, collectionConfig *utils.CollectionsConfig) error {
	database := d.Client.Database(config.Name)

	// Initialize User Collections
	err := d.SetUpUserCollections(database, collectionConfig)
	if err != nil {
		return err
	}

	// Initialize Announcements Collection
	err = d.SetUpAnnouncementCollection(database, collectionConfig)
	if err != nil {
		return err
	}

	// Event Collections Setup
	err = d.SetUpEventCollections(database, collectionConfig)
	if err != nil {
		return err
	}

	// Venue Collections Setup
	err = d.SetUpVenueCollections(database, collectionConfig)
	if err != nil {
		return err
	}

	// Initialize Club Collections
	err = d.SetUpClubCollections(database, collectionConfig)
	if err != nil {
		return err
	}

	// Initialize Organization Collections
	err = d.SetUpOrganizationCollections(database, collectionConfig)
	if err != nil {
		return err
	}

	// Initialize Post Collections
	err = d.SetUpPostCollections(database, collectionConfig)
	if err != nil {
		return err
	}

	// Initialize Reports Collection
	err = d.SetUpReportCollections(database, collectionConfig)
	if err != nil {
		return err
	}

	// Initialize Locales Collection
	err = d.SetUpLocaleCollections(database, collectionConfig, config)
	if err != nil {
		return err
	}

	// Initialize Application Config Collections
	err = d.SetUpAppConfigCollections(database, collectionConfig)
	if err != nil {
		return err
	}

	return nil
}
