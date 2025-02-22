package database

import (
	"context"
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

func (d *Database) SetUpCollections(config *utils.DatabaseConfig, collectionConfig *utils.CollectionsConfig) {
	database := d.Client.Database(config.Name)

	d.AuthCol = database.Collection(collectionConfig.AuthCollection)
	d.UserCol = database.Collection(collectionConfig.UserCollection)
	d.ClubCol = database.Collection(collectionConfig.ClubCollection)
	d.OrgCol = database.Collection(collectionConfig.OrgCollection)

	d.EventCol = database.Collection(collectionConfig.EventCollection)
	d.VenueCol = database.Collection(collectionConfig.VenueCollection)

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
}
