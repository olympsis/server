package database

import (
	"context"
	"fmt"
	"olympsis-server/utils"
	"olympsis-server/utils/secrets"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

type Database struct {
	Logger *logrus.Logger
	Client *mongo.Client

	AnnouncementCollection *mongo.Collection

	AuthCollection   *mongo.Collection
	UserCollection   *mongo.Collection
	VenuesCollection *mongo.Collection

	PostsCollection         *mongo.Collection
	PostCommentsCollection  *mongo.Collection
	PostReactionsCollection *mongo.Collection

	ClubCollection            *mongo.Collection
	ClubInvitationCollection  *mongo.Collection
	ClubApplicationCollection *mongo.Collection
	ClubMembersCollection     *mongo.Collection

	OrgCollection                 *mongo.Collection
	OrgInvitationCollection       *mongo.Collection
	OrgApplicationCollection      *mongo.Collection
	OrganizationMembersCollection *mongo.Collection

	EventsCollection                    *mongo.Collection
	EventLogsCollection                 *mongo.Collection
	EventViewsCollection                *mongo.Collection
	EventTeamsCollection                *mongo.Collection
	EventCommentsCollection             *mongo.Collection
	EventInvitationsCollection          *mongo.Collection
	EventParticipantsCollection         *mongo.Collection
	EventTeamsWaitlistCollection        *mongo.Collection
	EventParticipantsWaitlistCollection *mongo.Collection

	ClubTransactionsCollection      *mongo.Collection
	ClubFinancialAccountsCollection *mongo.Collection

	BugReportCollection    *mongo.Collection
	PostReportCollection   *mongo.Collection
	MemberReportCollection *mongo.Collection
	VenueReportCollection  *mongo.Collection
	EventReportCollection  *mongo.Collection

	CountriesCollection     *mongo.Collection
	AdminAreasCollection    *mongo.Collection
	SubAdminAreasCollection *mongo.Collection

	TagsCollection   *mongo.Collection
	SportsCollection *mongo.Collection

	// NOTIFICATIONS
	NotificationsClient          *mongo.Client
	NotificationTopicsCollection *mongo.Collection
	NotificationLogsCollection   *mongo.Collection
	UserNotificationsCollection  *mongo.Collection
	PushNotificationsCollection  *mongo.Collection
}

func NewDatabase(l *logrus.Logger) *Database {
	return &Database{Logger: l}
}

func (d *Database) EstablishConnection(manager *secrets.Manager, config *utils.ServerConfig) {

	d.Logger.Info("Connecting to Database...")

	/*
		Connect to Mongo Database
	*/
	dbConfig := utils.GetDatabaseConfig(manager)
	collectionConfig := utils.GetCollectionsConfig()

	switch config.Mode {
	case "PRODUCTION":
		opts := options.Client().ApplyURI(`mongodb+srv://` + dbConfig.User + `:` + dbConfig.Password + `@` + dbConfig.Address + `/?retryWrites=true&w=majority`)
		client, err := mongo.Connect(opts)
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
		client, err := mongo.Connect(opts)
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

	// Initialize Auth Collections
	err := d.SetUpAuthCollections(database, collectionConfig)
	if err != nil {
		return err
	}

	// Initialize User Collections
	err = d.SetUpUserCollections(database, collectionConfig)
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

	// Initialize Notification collections
	noteDatabase := d.Client.Database(config.NotificationName)
	err = d.SetUpNotificationsCollections(noteDatabase, collectionConfig)
	if err != nil {
		return err
	}

	return nil
}
