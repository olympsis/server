package orm

import (
	"context"
	"fmt"
	"olympsis-server/database"

	"github.com/olympsis/models"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type FieldReportORM struct {
	Database *database.Database
	Logger   *logrus.Logger
}

func (orm *FieldReportORM) Insert(ctx context.Context, report *models.VenueReportDao, opts *options.InsertOneOptions) error {
	_, err := orm.Database.VenueReportCol.InsertOne(ctx, report, opts)
	return err
}

func (orm *FieldReportORM) Find(ctx context.Context, filter interface{}, opts *options.AggregateOptions) (*[]models.VenueReport, error) {

	pipeline0 := bson.M{
		"$lookup": bson.M{
			"from":         "auth",
			"localField":   "user",
			"foreignField": "uuid",
			"as":           "_auth",
		},
	}

	pipeline1 := bson.M{
		"$lookup": bson.M{
			"from":         "users",
			"localField":   "user",
			"foreignField": "uuid",
			"as":           "users",
		},
	}

	pipeline2 := bson.M{
		"$lookup": bson.M{
			"from":         "fields",
			"localField":   "venue_id",
			"foreignField": "_id",
			"as":           "fields",
		},
	}

	pipeline3 := bson.M{
		"$addFields": bson.M{
			"user": bson.M{
				"$arrayElemAt": bson.A{"$users", 0},
			},
			"field": bson.M{
				"$arrayElemAt": bson.A{"$fields", 0},
			},
		},
	}

	pipeline4 := bson.M{
		"$addFields": bson.M{
			"user.first_name": bson.M{
				"$arrayElemAt": bson.A{"$_auth.first_name", 0},
			},
			"user.last_name": bson.M{
				"$arrayElemAt": bson.A{"$_auth.last_name", 0},
			},
			"user.image_url": bson.M{
				"$arrayElemAt": bson.A{"$_auth.image_url", 0},
			},
		},
	}

	pipeline5 := bson.M{
		"$project": bson.M{
			"users":      0,
			"fields":     0,
			"venue_id":   0,
			"user.token": 0,
		},
	}

	pipeline := bson.A{
		filter,
		pipeline0,
		pipeline1,
		pipeline2,
		pipeline3,
		pipeline4,
		pipeline5,
	}

	cur, err := orm.Database.VenueReportCol.Aggregate(context.TODO(), pipeline, opts)
	if err != nil {
		return nil, err
	}

	var reports []models.VenueReport
	for cur.Next(context.TODO()) {
		var report models.VenueReport
		err = cur.Decode(&report)
		if err != nil {
			orm.Logger.Error(fmt.Sprintf("failed to decode report: %s", err.Error()))
		}
		reports = append(reports, report)
	}

	return &reports, nil
}

func (orm *FieldReportORM) Update(ctx context.Context, filter interface{}, update interface{}) error {
	_, err := orm.Database.VenueReportCol.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	return nil
}

func (orm *FieldReportORM) Delete(ctx context.Context, filter interface{}) error {
	_, err := orm.Database.VenueReportCol.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}
