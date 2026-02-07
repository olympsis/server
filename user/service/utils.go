package service

import (
	"context"
	"olympsis-server/database"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func CheckInUser(uuid string, database *database.Database) (*models.CheckIn, error) {

	ctx := context.Background()

	filter := bson.M{
		"$match": bson.M{
			"uuid": uuid,
		},
	}

	authLookup := bson.M{
		"$lookup": bson.M{
			"from":         "auth",
			"localField":   "uuid",
			"foreignField": "uuid",
			"as":           "_auth",
		},
	}

	clubsLookup := bson.M{
		"$lookup": bson.M{
			"from":         "clubs",
			"localField":   "clubs",
			"foreignField": "_id",
			"as":           "_clubs",
		},
	}

	orgsLookup := bson.M{
		"$lookup": bson.M{
			"from":         "organizations",
			"localField":   "organizations",
			"foreignField": "_id",
			"as":           "_orgs",
		},
	}

	clubMembersLookup := bson.M{
		"$lookup": bson.M{
			"from":         "user",
			"localField":   "_clubs.members.uuid",
			"foreignField": "uuid",
			"as":           "_clubs_members",
		},
	}

	orgMembersLookup := bson.M{
		"$lookup": bson.M{
			"from":         "user",
			"localField":   "_orgs.members.uuid",
			"foreignField": "uuid",
			"as":           "_orgs_members",
		},
	}

	authAddFields := bson.M{
		"$addFields": bson.M{
			"first_name": bson.M{
				"$arrayElemAt": bson.A{
					"$_auth.first_name",
					0,
				},
			},
			"last_name": bson.M{
				"$arrayElemAt": bson.A{
					"$_auth.last_name",
					0,
				},
			},
		},
	}

	addClubMembers := bson.M{
		"$addFields": bson.M{
			"_clubs": bson.M{
				"$map": bson.M{
					"input": "$_clubs",
					"as":    "club",
					"in": bson.M{
						"$mergeObjects": bson.A{
							"$$club",
							bson.M{
								"members": bson.M{
									"$map": bson.M{
										"input": "$$club.members",
										"as":    "member",
										"in": bson.M{
											"$mergeObjects": bson.A{
												"$$member",
												bson.M{
													"user": bson.M{
														"$arrayElemAt": bson.A{
															bson.M{
																"$filter": bson.M{
																	"input": "$_clubs_members",
																	"as":    "data",
																	"cond": bson.M{
																		"$eq": bson.A{"$$data.uuid", "$$member.uuid"},
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
						},
					},
				},
			},
		},
	}

	addOrgMembers := bson.M{
		"$addFields": bson.M{
			"_orgs": bson.M{
				"$map": bson.M{
					"input": "$_orgs",
					"as":    "org",
					"in": bson.M{
						"$mergeObjects": bson.A{
							"$$org",
							bson.M{
								"members": bson.M{
									"$map": bson.M{
										"input": "$$org.members",
										"as":    "member",
										"in": bson.M{
											"$mergeObjects": bson.A{
												"$$member",
												bson.M{
													"user": bson.M{
														"$arrayElemAt": bson.A{
															bson.M{
																"$filter": bson.M{
																	"input": "$_orgs_members",
																	"as":    "data",
																	"cond": bson.M{
																		"$eq": bson.A{"$$data.uuid", "$$member.uuid"},
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
						},
					},
				},
			},
		},
	}

	reformDocument := bson.M{
		"$project": bson.M{
			"user_data": bson.M{
				"uuid":       "$uuid",
				"username":   "$username",
				"visibility": "$visibility",
				"sports":     "$sports",
				"image_url":  "$image_url",
				"first_name": "$first_name",
				"last_name":  "$last_name",
				"bio":        "$bio",
			},
			"clubs":         "$_clubs",
			"organizations": "$_orgs",
		},
	}

	project := bson.M{
		"$project": bson.M{
			"_id":                                      0,
			"_auth":                                    0,
			"_clubs":                                   0,
			"_orgs":                                    0,
			"token":                                    0,
			"device_token":                             0,
			"_clubs_members":                           0,
			"_orgs_members":                            0,
			"clubs.members.uuid":                       0,
			"clubs.members.user._id":                   0,
			"clubs.members.user.clubs":                 0,
			"clubs.members.user.sports":                0,
			"clubs.members.user.visibility":            0,
			"clubs.members.user.token":                 0,
			"clubs.members.user.device_token":          0,
			"clubs.members.user.organizations":         0,
			"organizations.members.uuid":               0,
			"organizations.members.user._id":           0,
			"organizations.members.user.clubs":         0,
			"organizations.members.user.sports":        0,
			"organizations.members.user.visibility":    0,
			"organizations.members.user.token":         0,
			"organizations.members.user.device_token":  0,
			"organizations.members.user.organizations": 0,
		},
	}

	pipeline := bson.A{
		filter,
		authLookup,
		clubsLookup,
		orgsLookup,
		clubMembersLookup,
		orgMembersLookup,
		authAddFields,
		addClubMembers,
		addOrgMembers,
		reformDocument,
		project,
	}

	cur, err := database.UserCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}

	var data models.CheckIn
	if cur.Next(ctx) {
		err = cur.Decode(&data)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, mongo.ErrNoDocuments
	}

	return &data, nil
}

func FindUser(uuid string, database *database.Database) (*models.UserData, error) {

	ctx := context.Background()

	filter := bson.M{
		"$match": bson.M{
			"uuid": uuid,
		},
	}

	authLookup := bson.M{
		"$lookup": bson.M{
			"from":         "auth",
			"localField":   "uuid",
			"foreignField": "uuid",
			"as":           "_auth",
		},
	}

	authAddFields := bson.M{
		"$addFields": bson.M{
			"first_name": bson.M{
				"$arrayElemAt": bson.A{
					"$_auth.first_name",
					0,
				},
			},
			"last_name": bson.M{
				"$arrayElemAt": bson.A{
					"$_auth.last_name",
					0,
				},
			},
		},
	}

	pipeline := bson.A{
		filter,
		authLookup,
		authAddFields,
	}

	cur, err := database.UserCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}

	var data models.UserData
	if cur.Next(ctx) {
		err = cur.Decode(&data)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, mongo.ErrNoDocuments
	}

	return &data, nil
}
