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
	Logger             *logrus.Logger
	Client             *mongo.Client
	AuthCol            *mongo.Collection
	UserCol            *mongo.Collection
	ClubCol            *mongo.Collection
	EventCol           *mongo.Collection
	FieldCol           *mongo.Collection
	PostCol            *mongo.Collection
	ClubInvCol         *mongo.Collection
	FriendReqCol       *mongo.Collection
	ClubApplicationCol *mongo.Collection
}

func NewDatabase(l *logrus.Logger) *Database {
	return &Database{Logger: l}
}

func (d *Database) EstablishConnection() {

	d.Logger.Info("Connecting to Database...")

	credential := options.Credential{
		AuthSource: os.Getenv("admin"),
		Username:   os.Getenv("DB_USR"),
		Password:   os.Getenv("DB_PASS"),
	}
	dbLoc := os.Getenv("DB_ADDR")
	opts := options.Client().ApplyURI("mongodb://" + dbLoc + ":27017")
	opts.Auth = &credential

	client, err := mongo.Connect(context.Background(), opts)

	// logs connection result and sets client
	if err != nil {
		panic("Failed to connect to Database: " + err.Error())
	} else {
		// ping database
		if err = client.Ping(context.TODO(), readpref.Primary()); err != nil {
			panic(err)
		}
		d.Client = client
		d.AuthCol = d.Client.Database(os.Getenv("DB_NAME")).Collection(os.Getenv("AUTH_COL"))
		d.UserCol = d.Client.Database(os.Getenv("DB_NAME")).Collection(os.Getenv("USER_COL"))
		d.ClubCol = d.Client.Database(os.Getenv("DB_NAME")).Collection(os.Getenv("CLUB_COL"))
		d.EventCol = d.Client.Database(os.Getenv("DB_NAME")).Collection(os.Getenv("EVENT_COL"))
		d.FieldCol = d.Client.Database(os.Getenv("DB_NAME")).Collection(os.Getenv("FIELD_COL"))
		d.PostCol = d.Client.Database(os.Getenv("DB_NAME")).Collection(os.Getenv("POST_COL"))
		d.ClubInvCol = d.Client.Database(os.Getenv("DB_NAME")).Collection(os.Getenv("CINVITE_COL"))
		d.FriendReqCol = d.Client.Database(os.Getenv("DB_NAME")).Collection(os.Getenv("FREQUEST_COL"))
		d.ClubApplicationCol = d.Client.Database(os.Getenv("DB_NAME")).Collection(os.Getenv("CAPPICATIONS_COL"))

		d.Logger.Info("Database connection successful.")
	}
}

/*
  - Pings the database to make sure we have a connection

Returns:

	bool - wether or not we have a response from database
*/
func (d *Database) PingDatabase() bool {
	if err := d.Client.Ping(context.TODO(), readpref.Primary()); err != nil {
		d.Logger.Error("failed to ping database")
		return false
	}
	return true
}
