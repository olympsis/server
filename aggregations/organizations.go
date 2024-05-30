package aggregations

import (
	"context"
	"olympsis-server/database"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func AggregateOrganization(id *primitive.ObjectID, database *database.Database) (*models.Organization, error) {

	ctx := context.Background()

	// filter out all docs by our ID
	idPipeline := bson.M{
		"$match": bson.M{
			"_id": id,
		},
	}

	membersPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "users",
			"localField":   "members.uuid",
			"foreignField": "uuid",
			"as":           "users",
		},
	}

	addMembersPipeline := bson.M{
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

	projectPipeline := bson.M{
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
		membersPipeline,
		addMembersPipeline,
		childrenPipeline,
		projectPipeline,
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

func AggregateOrganizations(filter interface{}, database *database.Database) (*[]models.Organization, error) {

	ctx := context.Background()

	membersPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "users",
			"localField":   "members.uuid",
			"foreignField": "uuid",
			"as":           "users",
		},
	}

	addMembersPipeline := bson.M{
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

	projectPipeline := bson.M{
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
		membersPipeline,
		addMembersPipeline,
		childrenPipeline,
		projectPipeline,
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

func AggregateOrganizationApplication(id *primitive.ObjectID, database *database.Database) (*models.OrganizationApplication, error) {

	ctx := context.Background()

	matchPipeline := bson.M{
		"$match": bson.M{
			"_id": id,
		},
	}

	lookupPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "clubs",
			"localField":   "club_id",
			"foreignField": "_id",
			"as":           "result",
		},
	}

	movePipeline := bson.M{
		"$addFields": bson.M{
			"club": bson.M{
				"$arrayElemAt": bson.A{
					"$result",
					0,
				},
			},
		},
	}

	cleanupPipeline := bson.M{
		"$project": bson.M{
			"result":          0,
			"club_id":         0,
			"organization_id": 0,
		},
	}

	pipeline := bson.A{
		matchPipeline,
		lookupPipeline,
		movePipeline,
		cleanupPipeline,
	}

	cur, err := database.OrgApplicationCol.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}

	var application models.OrganizationApplication
	if cur.Next(ctx) {
		err = cur.Decode(&application)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, mongo.ErrNoDocuments
	}

	return &application, nil
}

func AggregateOrganizationApplications(organizationID *primitive.ObjectID, status string, database *database.Database) (*[]models.OrganizationApplication, error) {

	ctx := context.Background()

	matchPipeline := bson.M{
		"$match": bson.M{
			"organization_id": organizationID,
			"status":          status,
		},
	}

	lookupPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "clubs",
			"localField":   "club_id",
			"foreignField": "_id",
			"as":           "result",
		},
	}

	movePipeline := bson.M{
		"$addFields": bson.M{
			"club": bson.M{
				"$arrayElemAt": bson.A{
					"$result",
					0,
				},
			},
		},
	}

	cleanupPipeline := bson.M{
		"$project": bson.M{
			"result":          0,
			"club_id":         0,
			"organization_id": 0,
		},
	}

	pipeline := bson.A{
		matchPipeline,
		lookupPipeline,
		movePipeline,
		cleanupPipeline,
	}

	cur, err := database.OrgApplicationCol.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}

	var response []models.OrganizationApplication
	for cur.Next(context.TODO()) {
		var application models.OrganizationApplication
		err := cur.Decode(&application)
		if err != nil {
			database.Logger.Error("failed to decode event", err)
		}
		response = append(response, application)
	}

	return &response, nil
}
