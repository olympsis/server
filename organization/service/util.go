package service

import (
	"context"
	"olympsis-server/database"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func FindOrganization(id *primitive.ObjectID, database *database.Database) (*models.Organization, error) {

	ctx := context.Background()

	// filter out all docs by our ID
	idPipeline := bson.M{
		"$match": bson.M{
			"_id": id,
		},
	}

	membersPiepline := bson.M{
		"$lookup": bson.M{
			"from":         "users",
			"localField":   "members.uuid",
			"foreignField": "uuid",
			"as":           "users",
		},
	}

	addMembersPiepline := bson.M{
		"$addFields": bson.M{
			"members": bson.M{
				"$map": bson.M{
					"input": "$members",
					"as":    "member",
					"in": bson.M{
						"$mergeObjects": bson.A{
							"$$member",
							bson.M{
								"user": bson.M{
									"$arrayElemAt": bson.A{
										bson.M{
											"$filter": bson.M{
												"input": "$users",
												"as":    "u",
												"cond": bson.M{
													"$eq": bson.A{
														"$$u.uuid",
														"$$member.uuid",
													},
												},
											},
										},
										0,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	childrenPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "clubs",
			"localField":   "_id",
			"foreignField": "parent_id",
			"as":           "children",
		},
	}

	projectPiepline := bson.M{
		"$project": bson.M{
			"users":                      0,
			"members.uuid":               0,
			"members.user._id":           0,
			"members.user.token":         0,
			"members.user.clubs":         0,
			"members.user.sports":        0,
			"members.user.visibility":    0,
			"members.user.device_token":  0,
			"members.user.organizations": 0,
		},
	}

	pipeline := bson.A{
		idPipeline,
		membersPiepline,
		addMembersPiepline,
		childrenPipeline,
		projectPiepline,
	}

	cur, err := database.OrgCol.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}

	var org models.Organization
	if cur.Next(ctx) {
		err = cur.Decode(&org)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, mongo.ErrNoDocuments
	}

	return &org, nil
}

func FindOrganizations(filter interface{}, database *database.Database) (*[]models.Organization, error) {

	ctx := context.Background()

	membersPiepline := bson.M{
		"$lookup": bson.M{
			"from":         "users",
			"localField":   "members.uuid",
			"foreignField": "uuid",
			"as":           "users",
		},
	}

	addMembersPiepline := bson.M{
		"$addFields": bson.M{
			"members": bson.M{
				"$map": bson.M{
					"input": "$members",
					"as":    "member",
					"in": bson.M{
						"$mergeObjects": bson.A{
							"$$member",
							bson.M{
								"user": bson.M{
									"$arrayElemAt": bson.A{
										bson.M{
											"$filter": bson.M{
												"input": "$users",
												"as":    "u",
												"cond": bson.M{
													"$eq": bson.A{
														"$$u.uuid",
														"$$member.uuid",
													},
												},
											},
										},
										0,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	childrenPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "clubs",
			"localField":   "_id",
			"foreignField": "parent_id",
			"as":           "children",
		},
	}

	projectPiepline := bson.M{
		"$project": bson.M{
			"users":                      0,
			"members.uuid":               0,
			"members.user._id":           0,
			"members.user.token":         0,
			"members.user.clubs":         0,
			"members.user.sports":        0,
			"members.user.visibility":    0,
			"members.user.device_token":  0,
			"members.user.organizations": 0,
		},
	}

	pipeline := bson.A{
		filter,
		membersPiepline,
		addMembersPiepline,
		childrenPipeline,
		projectPiepline,
	}

	cur, err := database.OrgCol.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}

	var response []models.Organization
	for cur.Next(context.TODO()) {
		var org models.Organization
		err := cur.Decode(&org)
		if err != nil {
			database.Logger.Error("failed to decode event", err)
		}
		response = append(response, org)
	}

	return &response, nil
}
