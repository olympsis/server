package database

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type Database struct {
	Logger      *logrus.Logger
	Client      *mongo.Client
	NotifClient *mongo.Client

	AuthCol      *mongo.Collection
	UserCol      *mongo.Collection
	ClubCol      *mongo.Collection
	OrgCol       *mongo.Collection
	EventCol     *mongo.Collection
	FieldCol     *mongo.Collection
	PostCol      *mongo.Collection
	ClubInvCol   *mongo.Collection
	CommentsCol  *mongo.Collection
	FriendReqCol *mongo.Collection

	ClubApplicationCol *mongo.Collection
	OrgApplicationCol  *mongo.Collection

	EventInvitationCol *mongo.Collection
	ClubInvitationCol  *mongo.Collection
	OrgInvitationCol   *mongo.Collection

	BugReportCol    *mongo.Collection
	PostReportCol   *mongo.Collection
	MemberReportCol *mongo.Collection
	FieldReportCol  *mongo.Collection
	EventReportCol  *mongo.Collection

	CountriesCol     *mongo.Collection
	AdminAreasCol    *mongo.Collection
	SubAdminAreasCol *mongo.Collection
}

func NewDatabase(l *logrus.Logger) *Database {
	return &Database{Logger: l}
}

func (d *Database) EstablishConnection() {

	d.Logger.Info("Connecting to Database...")

	/*
		Connect to Mongo Database
	*/
	// mode := os.Getenv("MODE")
	dbUser := os.Getenv("DB_USR")
	dbPass := os.Getenv("DB_PASS")
	dbLoc := os.Getenv("DB_ADDR")

	// if mode == "PRODUCTION" {
	opts := options.Client().ApplyURI(`mongodb+srv://` + dbUser + `:` + dbPass + `@` + dbLoc + `/?retryWrites=true&w=majority`)
	client, err := mongo.Connect(context.Background(), opts)
	if err != nil {
		d.Logger.Fatal("Failed to connect to Database: " + err.Error())
	}

	err = client.Ping(context.Background(), readpref.Primary())
	if err != nil {
		d.Logger.Fatal("Failed to connect to Database: " + err.Error())
	}

	d.Client = client

	// } else {

	// 	opts := options.Client().ApplyURI(`mongodb://` + dbUser + `:` + dbPass + `@` + dbLoc + `/?retryWrites=true&w=majority`)
	// 	client, err := mongo.Connect(context.Background(), opts)
	// 	if err != nil {
	// 		d.Logger.Fatal("Failed to connect to Database: " + err.Error())
	// 	}

	// 	err = client.Ping(context.Background(), readpref.Primary())
	// 	if err != nil {
	// 		d.Logger.Fatal("Failed to connect to Database: " + err.Error())
	// 	}

	// 	d.Client = client
	// }

	d.LinkCollections()
	d.Logger.Info("Database connection successful.")
}

func (d *Database) LinkCollections() {
	database := d.Client.Database(os.Getenv("DB_NAME"))
	d.AuthCol = database.Collection(os.Getenv("AUTH_COL"))
	d.UserCol = database.Collection(os.Getenv("USER_COL"))
	d.ClubCol = database.Collection(os.Getenv("CLUB_COL"))
	d.OrgCol = database.Collection(os.Getenv("ORG_COL"))
	d.EventCol = database.Collection(os.Getenv("EVENT_COL"))
	d.FieldCol = database.Collection(os.Getenv("VENUE_COL"))
	d.PostCol = database.Collection(os.Getenv("POST_COL"))
	d.ClubInvCol = database.Collection(os.Getenv("CLUB_INVITE_COL"))
	d.CommentsCol = database.Collection(os.Getenv("COMMENTS_COL"))
	d.FriendReqCol = database.Collection(os.Getenv("FRIEND_REQUEST_COL"))

	d.ClubApplicationCol = database.Collection(os.Getenv("CLUB_APPLICATIONS_COL"))
	d.OrgApplicationCol = database.Collection(os.Getenv("ORG_APPLICATIONS_COL"))

	d.EventInvitationCol = database.Collection(os.Getenv("EVENT_INVITATIONS_COL"))
	d.ClubInvitationCol = database.Collection(os.Getenv("CLUB_INVITATIONS_COL"))
	d.OrgInvitationCol = database.Collection(os.Getenv("ORG_INVITATIONS_COL"))

	d.BugReportCol = database.Collection(os.Getenv("BUG_REPORT_COL"))
	d.PostReportCol = database.Collection(os.Getenv("POST_REPORT_COL"))
	d.FieldReportCol = database.Collection(os.Getenv("FIELD_REPORT_COL"))
	d.EventReportCol = database.Collection(os.Getenv("EVENT_REPORT_COL"))
	d.MemberReportCol = database.Collection(os.Getenv("MEMBER_REPORT_COL"))

	localeDB := d.Client.Database(os.Getenv("LOCALE_DB"))
	d.CountriesCol = localeDB.Collection(os.Getenv("COUNTRIES_COL"))
	d.AdminAreasCol = localeDB.Collection(os.Getenv("ADMIN_AREAS_COL"))
	d.SubAdminAreasCol = localeDB.Collection(os.Getenv("SUB_ADMIN_AREAS_COL"))
}
