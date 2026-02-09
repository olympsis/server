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

type MemberReportORM struct {
	Database *database.Database
	Logger   *logrus.Logger
}

func (orm *MemberReportORM) Insert(ctx context.Context, report *models.MemberReportDao, opts *options.InsertOneOptionsBuilder) error {
	_, err := orm.Database.MemberReportCollection.InsertOne(ctx, report, opts)
	return err
}

func (orm *MemberReportORM) Find(ctx context.Context, filter interface{}, opts *options.AggregateOptionsBuilder) (*[]models.MemberReport, error) {

	pipeline1 := bson.M{
		"$lookup": bson.M{
			"from":         "clubs",
			"localField":   "group_id",
			"foreignField": "_id",
			"as":           "club",
		},
	}

	pipeline2 := bson.M{
		"$addFields": bson.M{
			"club": bson.M{
				"$arrayElemAt": bson.A{"$club", 0},
			},
		},
	}

	pipeline3 := bson.M{
		"$addFields": bson.M{
			"user": bson.M{
				"$arrayElemAt": bson.A{
					bson.M{
						"$filter": bson.M{
							"input": "$club.members",
							"as":    "member",
							"cond":  bson.M{"$eq": bson.A{"$$member._id", "$member_id"}},
						},
					},
					0,
				},
			},
		},
	}

	pipeline4 := bson.M{
		"$lookup": bson.M{
			"from":         "users",
			"localField":   "user.uuid",
			"foreignField": "user_id",
			"as":           "user",
		},
	}

	pipeline5 := bson.M{
		"$addFields": bson.M{
			"member": bson.M{
				"$arrayElemAt": bson.A{"$user", 0},
			},
		},
	}

	pipeline6 := bson.M{
		"$project": bson.M{
			"club":         0,
			"member_id":    0,
			"group_id":     0,
			"token":        0,
			"user":         0,
			"member._id":   0,
			"member.token": 0,
		},
	}

	pipeline := bson.A{
		filter,
		pipeline1,
		pipeline2,
		pipeline3,
		pipeline4,
		pipeline5,
		pipeline6,
	}

	cur, err := orm.Database.MemberReportCollection.Aggregate(context.TODO(), pipeline, opts)
	if err != nil {
		return nil, err
	}

	var reports []models.MemberReport
	for cur.Next(context.TODO()) {
		var report models.MemberReport
		err = cur.Decode(&report)
		if err != nil {
			orm.Logger.Error(fmt.Sprintf("failed to decode report: %s", err.Error()))
		}
		reports = append(reports, report)
	}

	return &reports, nil
}

func (orm *MemberReportORM) Update(ctx context.Context, filter interface{}, update interface{}) error {
	_, err := orm.Database.MemberReportCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	return nil
}

func (orm *MemberReportORM) Delete(ctx context.Context, filter interface{}) error {
	_, err := orm.Database.MemberReportCollection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}
