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

type PostReportORM struct {
	Database *database.Database
	Logger   *logrus.Logger
}

func (orm *PostReportORM) Insert(ctx context.Context, report *models.PostReportDao, opts *options.InsertOneOptionsBuilder) error {
	_, err := orm.Database.PostReportCollection.InsertOne(ctx, report, opts)
	return err
}

func (orm *PostReportORM) Find(ctx context.Context, filter interface{}, opts *options.AggregateOptionsBuilder) (*[]models.PostReport, error) {

	pipeline1 := bson.M{
		"$lookup": bson.M{
			"from":         "posts",
			"localField":   "post_id",
			"foreignField": "_id",
			"as":           "_post",
		},
	}

	pipeline2 := bson.M{
		"$lookup": bson.M{
			"from":         "users",
			"localField":   "_post.poster",
			"foreignField": "user_id",
			"as":           "_poster",
		},
	}

	pipeline3 := bson.M{
		"$addFields": bson.M{
			"_post.poster": bson.M{
				"$arrayElemAt": bson.A{"$_poster", 0},
			},
		},
	}

	pipeline4 := bson.M{
		"$addFields": bson.M{
			"post": bson.M{
				"$arrayElemAt": bson.A{"$_post", 0},
			},
		},
	}

	pipeline5 := bson.M{
		"$project": bson.M{
			"_post":           0,
			"_poster":         0,
			"post_id":         0,
			"group_id":        0,
			"post.poster._id": 0,
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

	cur, err := orm.Database.PostReportCollection.Aggregate(context.TODO(), pipeline, opts)
	if err != nil {
		return nil, err
	}

	var reports []models.PostReport
	for cur.Next(context.TODO()) {
		var report models.PostReport
		err = cur.Decode(&report)
		if err != nil {
			orm.Logger.Error(fmt.Sprintf("failed to decode report: %s", err.Error()))
		}
		reports = append(reports, report)
	}

	return &reports, nil
}

func (orm *PostReportORM) Update(ctx context.Context, filter interface{}, update interface{}) error {
	_, err := orm.Database.PostReportCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	return nil
}

func (orm *PostReportORM) Delete(ctx context.Context, filter interface{}) error {
	_, err := orm.Database.PostReportCollection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}
