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

type EventReportORM struct {
	Database *database.Database
	Logger   *logrus.Logger
}

func (orm *EventReportORM) Insert(ctx context.Context, report *models.EventReportDao, opts *options.InsertOneOptions) error {
	_, err := orm.Database.EventReportCollection.InsertOne(ctx, report, opts)
	return err
}

func (orm *EventReportORM) Find(ctx context.Context, filter interface{}, opts *options.AggregateOptions) (*[]models.EventReport, error) {

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
			"as":           "_users",
		},
	}

	pipeline2 := bson.M{
		"$lookup": bson.M{
			"from":         "events",
			"localField":   "event_id",
			"foreignField": "_id",
			"as":           "_events",
		},
	}

	pipeline3 := bson.M{
		"$addFields": bson.M{
			"event": bson.M{
				"$arrayElemAt": bson.A{"$_events", 0},
			},
			"user": bson.M{
				"$arrayElemAt": bson.A{"$_users", 0},
			},
		},
	}

	pipeline4 := bson.M{
		"$addFields": bson.M{
			"field": bson.M{
				"$arrayElemAt": bson.A{"$fields", 0},
			},
		},
	}

	pipeline5 := bson.M{
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

	pipeline6 := bson.M{
		"$lookup": bson.M{
			"from":         "users",
			"localField":   "event.poster",
			"foreignField": "uuid",
			"as":           "_poster",
		},
	}

	pipeline7 := bson.M{
		"$addFields": bson.M{
			"event.poster": bson.M{
				"$arrayElemAt": bson.A{"$_poster", 0},
			},
		},
	}

	pipeline8 := bson.M{
		"$lookup": bson.M{
			"from":         "fields",
			"localField":   "event.field._id",
			"foreignField": "_id",
			"as":           "field_data",
		},
	}

	// add field data to document correctly
	pipeline9 := bson.M{
		"$addFields": bson.M{
			"event.field_data": bson.M{
				"$arrayElemAt": bson.A{
					"$field_data",
					0,
				},
			},
		},
	}

	// find clubs data
	pipeline10 := bson.M{
		"$lookup": bson.M{
			"from":         "clubs",
			"localField":   "event.organizers._id",
			"foreignField": "_id",
			"as":           "event.clubs",
		},
	}

	// find organizations data
	pipeline11 := bson.M{
		"$lookup": bson.M{
			"from":         "organizations",
			"localField":   "event.organizers._id",
			"foreignField": "_id",
			"as":           "event.organizations",
		},
	}

	pipeline12 := bson.M{
		"$project": bson.M{
			"_auth":    0,
			"_users":   0,
			"_poster":  0,
			"_events":  0,
			"event_id": 0,
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
		pipeline6,
		pipeline7,
		pipeline8,
		pipeline9,
		pipeline10,
		pipeline11,
		pipeline12,
	}

	cur, err := orm.Database.EventReportCollection.Aggregate(context.TODO(), pipeline, opts)
	if err != nil {
		return nil, err
	}

	var reports []models.EventReport
	for cur.Next(context.TODO()) {
		var report models.EventReport
		err = cur.Decode(&report)
		if err != nil {
			orm.Logger.Error(fmt.Sprintf("failed to decode report: %s", err.Error()))
		}
		reports = append(reports, report)
	}

	return &reports, nil
}

func (orm *EventReportORM) Update(ctx context.Context, filter interface{}, update interface{}) error {
	_, err := orm.Database.EventReportCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	return nil
}

func (orm *EventReportORM) Delete(ctx context.Context, filter interface{}) error {
	_, err := orm.Database.EventReportCollection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}
