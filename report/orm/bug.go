package orm

import (
	"context"
	"fmt"
	"olympsis-server/database"

	"github.com/olympsis/models"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type BugReportORM struct {
	Database *database.Database
	Logger   *logrus.Logger
}

func (orm *BugReportORM) Insert(ctx context.Context, report *models.BugReportDao, opts *options.InsertOneOptionsBuilder) error {
	_, err := orm.Database.BugReportCollection.InsertOne(ctx, report, opts)
	return err
}

func (orm *BugReportORM) Find(ctx context.Context, filter interface{}, opts *options.AggregateOptionsBuilder) (*[]models.BugReport, error) {

	pipeline1 := bson.M{
		"$lookup": bson.M{
			"from":         "auth",
			"localField":   "user",
			"foreignField": "uuid",
			"as":           "_auth",
		},
	}

	pipeline2 := bson.M{
		"$lookup": bson.M{
			"from":         "users",
			"localField":   "user",
			"foreignField": "uuid",
			"as":           "_users",
		},
	}

	pipeline3 := bson.M{
		"$addFields": bson.M{
			"user": bson.M{
				"$arrayElemAt": bson.A{"$_users", 0},
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
			"_auth":      0,
			"user._id":   0,
			"user.token": 0,
		},
	}

	pipeline := bson.A{
		filter,
		pipeline1,
		pipeline2,
		pipeline3,
		pipeline4,
		pipeline5,
	}

	cur, err := orm.Database.BugReportCollection.Aggregate(context.TODO(), pipeline, opts)
	if err != nil {
		return nil, err
	}

	var reports []models.BugReport
	for cur.Next(context.TODO()) {
		var report models.BugReport
		err = cur.Decode(&report)
		if err != nil {
			orm.Logger.Error(fmt.Sprintf("failed to decode report: %s", err.Error()))
		}
		reports = append(reports, report)
	}

	return &reports, nil
}

func (orm *BugReportORM) Update(ctx context.Context, filter interface{}, update interface{}) error {
	_, err := orm.Database.BugReportCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	return nil
}

func (orm *BugReportORM) Delete(ctx context.Context, filter interface{}) error {
	_, err := orm.Database.BugReportCollection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}
