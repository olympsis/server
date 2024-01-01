package database

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type Database struct {
	Logger      *logrus.Logger
	Pool        *pgxpool.Pool
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

	NotifCol *mongo.Collection

	ClubApplicationCol *mongo.Collection
	OrgApplicationCol  *mongo.Collection

	EventInvitationCol *mongo.Collection
	ClubInvitationCol  *mongo.Collection
	OrgInvitationCol   *mongo.Collection
}

func NewDatabase(l *logrus.Logger) *Database {
	return &Database{Logger: l}
}

func (d *Database) EstablishConnection() {

	d.Logger.Info("Connecting to Database...")

	/*
		Connect to Mongo Database
	*/
	dbUser := os.Getenv("DB_USR")
	dbPass := os.Getenv("DB_PASS")
	dbLoc := os.Getenv("DB_ADDR")
	opts := options.Client().ApplyURI(`mongodb+srv://` + dbUser + `:` + dbPass + `@` + dbLoc + `/?retryWrites=true&w=majority`)
	client, err := mongo.Connect(context.Background(), opts)
	if err != nil {
		d.Logger.Fatal("Failed to connect to Database: " + err.Error())
	}

	err = client.Ping(context.Background(), readpref.Primary())
	if err != nil {
		d.Logger.Fatal("Failed to connect to Database: " + err.Error())
	}

	noteLoc := os.Getenv("NOTIF_DB_ADDR")
	opts2 := options.Client().ApplyURI(`mongodb+srv://` + dbUser + `:` + dbPass + `@` + noteLoc + `/?retryWrites=true&w=majority`)
	client2, err := mongo.Connect(context.Background(), opts2)
	if err != nil {
		d.Logger.Fatal("Failed to connect to Database: " + err.Error())
	}

	err = client2.Ping(context.Background(), readpref.Primary())
	if err != nil {
		d.Logger.Fatal("Failed to connect to Database: " + err.Error())
	}

	d.Client = client
	d.AuthCol = d.Client.Database(os.Getenv("DB_NAME")).Collection(os.Getenv("AUTH_COL"))
	d.UserCol = d.Client.Database(os.Getenv("DB_NAME")).Collection(os.Getenv("USER_COL"))
	d.ClubCol = d.Client.Database(os.Getenv("DB_NAME")).Collection(os.Getenv("CLUB_COL"))
	d.OrgCol = d.Client.Database(os.Getenv("DB_NAME")).Collection(os.Getenv("ORG_COL"))
	d.EventCol = d.Client.Database(os.Getenv("DB_NAME")).Collection(os.Getenv("EVENT_COL"))
	d.FieldCol = d.Client.Database(os.Getenv("DB_NAME")).Collection(os.Getenv("FIELD_COL"))
	d.PostCol = d.Client.Database(os.Getenv("DB_NAME")).Collection(os.Getenv("POST_COL"))
	d.ClubInvCol = d.Client.Database(os.Getenv("DB_NAME")).Collection(os.Getenv("CINVITE_COL"))
	d.CommentsCol = d.Client.Database(os.Getenv("DB_NAME")).Collection(os.Getenv("COMMENTS_COL"))
	d.FriendReqCol = d.Client.Database(os.Getenv("DB_NAME")).Collection(os.Getenv("FREQUEST_COL"))

	d.ClubApplicationCol = d.Client.Database(os.Getenv("DB_NAME")).Collection(os.Getenv("CAPPICATIONS_COL"))
	d.OrgApplicationCol = d.Client.Database(os.Getenv("DB_NAME")).Collection(os.Getenv("OAPPICATIONS_COL"))

	d.EventInvitationCol = d.Client.Database(os.Getenv("DB_NAME")).Collection(os.Getenv("EVENT_INVITATIONS_COL"))
	d.ClubInvitationCol = d.Client.Database(os.Getenv("DB_NAME")).Collection(os.Getenv("CLUB_INVITATIONS_COL"))
	d.OrgInvitationCol = d.Client.Database(os.Getenv("DB_NAME")).Collection(os.Getenv("ORG_INVITATIONS_COL"))

	d.NotifClient = client2
	d.NotifCol = d.NotifClient.Database(os.Getenv("NOTIF_DB_NAME")).Collection(os.Getenv("NOTIF_COL"))

	d.Logger.Info("Database connection successful.")
}
