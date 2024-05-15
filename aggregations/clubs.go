package aggregations

import (
	"context"
	"olympsis-server/database"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func AggregateClub(id *primitive.ObjectID, database *database.Database) (*models.Club, error) {

	ctx := context.Background()

	// filter out all docs by our ID
	idPipeline := bson.M{
		"$match": bson.M{
			"_id": id,
		},
	}

	parentPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "organizations",
			"localField":   "parent_id",
			"foreignField": "_id",
			"as":           "organizations",
		},
	}

	addParentPipeline := bson.M{
		"$addFields": bson.M{
			"parent": bson.M{
				"$arrayElemAt": bson.A{
					"$organizations",
					0,
				},
			},
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

	managersPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "users",
			"localField":   "parent.members.uuid",
			"foreignField": "uuid",
			"as":           "managers",
		},
	}

	addManagersPipeline := bson.M{
		"$addFields": bson.M{
			"parent.members": bson.M{
				"$map": bson.M{
					"input": "$parent.members",
					"as":    "manager",
					"in": bson.M{
						"$mergeObjects": bson.A{
							"$$manager",
							bson.M{
								"user": bson.M{
									"$arrayElemAt": bson.A{
										bson.M{
											"$filter": bson.M{
												"input": "$managers",
												"as":    "m",
												"cond": bson.M{
													"$eq": bson.A{
														"$$m.uuid",
														"$$manager.uuid",
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

	projectPipeline := bson.M{
		"$project": bson.M{
			"users":                             0,
			"managers":                          0,
			"organizations":                     0,
			"members.uuid":                      0,
			"members.user._id":                  0,
			"members.user.clubs":                0,
			"members.user.sports":               0,
			"members.user.visibility":           0,
			"members.user.device_token":         0,
			"members.user.organizations":        0,
			"parent.members.uuid":               0,
			"parent.members.user._id":           0,
			"parent.members.user.clubs":         0,
			"parent.members.user.sports":        0,
			"parent.members.user.visibility":    0,
			"parent.members.user.device_token":  0,
			"parent.members.user.organizations": 0,
		},
	}

	pipeline := bson.A{
		idPipeline,
		parentPipeline,
		addParentPipeline,
		membersPipeline,
		addMembersPipeline,
		managersPipeline, addManagersPipeline,
		projectPipeline,
	}

	cur, err := database.ClubCol.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}

	var club models.Club
	if cur.Next(ctx) {
		err = cur.Decode(&club)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, mongo.ErrNoDocuments
	}

	return &club, nil
}

func AggregateClubs(filter interface{}, database *database.Database) (*[]models.Club, error) {

	ctx := context.Background()

	parentPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "organizations",
			"localField":   "parent_id",
			"foreignField": "_id",
			"as":           "organizations",
		},
	}

	addParentPipeline := bson.M{
		"$addFields": bson.M{
			"parent": bson.M{
				"$arrayElemAt": bson.A{
					"$organizations",
					0,
				},
			},
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

	managersPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "users",
			"localField":   "parent.members.uuid",
			"foreignField": "uuid",
			"as":           "managers",
		},
	}

	addManagersPipeline := bson.M{
		"$addFields": bson.M{
			"parent.members": bson.M{
				"$map": bson.M{
					"input": "$parent.members",
					"as":    "manager",
					"in": bson.M{
						"$mergeObjects": bson.A{
							"$$manager",
							bson.M{
								"user": bson.M{
									"$arrayElemAt": bson.A{
										bson.M{
											"$filter": bson.M{
												"input": "$managers",
												"as":    "m",
												"cond": bson.M{
													"$eq": bson.A{
														"$$m.uuid",
														"$$manager.uuid",
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

	projectPipeline := bson.M{
		"$project": bson.M{
			"users":                             0,
			"managers":                          0,
			"organizations":                     0,
			"members.uuid":                      0,
			"members.user._id":                  0,
			"members.user.clubs":                0,
			"members.user.sports":               0,
			"members.user.visibility":           0,
			"members.user.device_token":         0,
			"members.user.organizations":        0,
			"parent.members.uuid":               0,
			"parent.members.user._id":           0,
			"parent.members.user.clubs":         0,
			"parent.members.user.sports":        0,
			"parent.members.user.visibility":    0,
			"parent.members.user.device_token":  0,
			"parent.members.user.organizations": 0,
		},
	}

	pipeline := bson.A{
		filter,
		parentPipeline,
		addParentPipeline,
		membersPipeline,
		addMembersPipeline,
		managersPipeline, addManagersPipeline,
		projectPipeline,
	}

	cur, err := database.ClubCol.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}

	var response []models.Club
	for cur.Next(context.TODO()) {
		var club models.Club
		err := cur.Decode(&club)
		if err != nil {
			database.Logger.Error("failed to decode event", err)
		}
		response = append(response, club)
	}

	return &response, nil
}
