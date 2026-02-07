package aggregations

import (
	"context"
	"olympsis-server/database"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// AggregateClub gets a single club by ID with all related data
func AggregateClub(id bson.ObjectID, database *database.Database) (*models.Club, error) {
	ctx := context.Background()

	// Create ID filter pipeline stage
	idPipeline := bson.M{
		"$match": bson.M{
			"_id": id,
		},
	}

	// Get the core pipeline stages
	corePipeline := BuildClubCorePipeline()

	// Insert the ID filter at the beginning of the pipeline
	completePipeline := append(bson.A{idPipeline}, corePipeline...)

	// Execute the aggregation
	cur, err := database.ClubCollection.Aggregate(ctx, completePipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

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

// AggregateClubs fetches multiple clubs based on filter criteria
func AggregateClubs(
	filter bson.M,
	location *models.GeoJSON,
	radius float64,
	limit int,
	skip int,
	database *database.Database,
) (*[]models.Club, error) {
	ctx := context.TODO()

	// Get the core pipeline stages
	corePipeline := BuildClubCorePipeline()

	// Use a valid default filter if none provided
	if filter == nil {
		filter = bson.M{}
	}

	// Add location filter if provided
	var pipeline bson.A
	if location != nil && radius > 0 {
		// Create a separate geospatial pipeline stage rather than combining with the filter
		geoFilter := bson.M{
			"$match": bson.M{
				"location": bson.M{
					"$geoWithin": bson.M{
						"$centerSphere": bson.A{
							location.Coordinates,
							radius / 3963.2, // miles to radians
						},
					},
				},
			},
		}

		// First apply regular filters, then geo filter
		filterPipeline := bson.M{
			"$match": filter,
		}

		pipeline = append(bson.A{filterPipeline, geoFilter}, corePipeline...)
	} else {
		// Just use the regular filter if no location
		filterPipeline := bson.M{
			"$match": filter,
		}
		pipeline = append(bson.A{filterPipeline}, corePipeline...)
	}

	// Add pagination stages
	pipeline = append(pipeline,
		bson.M{"$skip": skip},
		bson.M{"$limit": limit},
	)

	// Execute the aggregation
	cur, err := database.ClubCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	// Process the results
	response := make([]models.Club, 0, limit)
	err = cur.All(ctx, &response)
	if err != nil {
		database.Logger.Error("Failed to decode club model. Error: ", err.Error())
	}

	return &response, nil
}

func AggregateClubApplication(id *bson.ObjectID, database *database.Database) (*models.ClubApplication, error) {

	ctx := context.Background()

	idPipeline := bson.M{
		"$match": bson.M{
			"_id": id,
		},
	}

	metaLookup := bson.M{
		"$lookup": bson.M{
			"from":         "users",
			"localField":   "uuid",
			"foreignField": "uuid",
			"as":           "meta",
		},
	}

	authLookup := bson.M{
		"$lookup": bson.M{
			"from":         "auth",
			"localField":   "uuid",
			"foreignField": "uuid",
			"as":           "auth",
		},
	}

	authObject := bson.M{
		"$addFields": bson.M{
			"auth": bson.M{
				"$arrayElemAt": bson.A{
					"$auth",
					0,
				},
			},
		},
	}

	metaObject := bson.M{
		"$addFields": bson.M{
			"meta": bson.M{
				"$arrayElemAt": bson.A{
					"$meta",
					0,
				},
			},
		},
	}

	userObject := bson.M{
		"$addFields": bson.M{
			"applicant": bson.M{
				"$mergeObjects": bson.A{
					"$auth",
					"$meta",
				},
			},
		},
	}

	cleanUp := bson.M{
		"$project": bson.M{
			"meta":    0,
			"auth":    0,
			"uuid":    0,
			"club_id": 0,
			"user": bson.M{
				"_id":          0,
				"email":        0,
				"token":        0,
				"access_token": 0,
				"provider":     0,
			},
		},
	}

	pipeline := bson.A{
		idPipeline,
		metaLookup,
		authLookup,
		authObject,
		metaObject,
		userObject,
		cleanUp,
	}

	cur, err := database.ClubApplicationCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}

	var application models.ClubApplication
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

func AggregateClubApplications(clubId *bson.ObjectID, status string, database *database.Database) (*[]models.ClubApplication, error) {

	ctx := context.Background()

	filter := bson.M{
		"$match": bson.M{
			"club_id": clubId,
			"status":  status,
		},
	}

	metaLookup := bson.M{
		"$lookup": bson.M{
			"from":         "users",
			"localField":   "applicant",
			"foreignField": "uuid",
			"as":           "meta",
		},
	}

	authLookup := bson.M{
		"$lookup": bson.M{
			"from":         "auth",
			"localField":   "applicant",
			"foreignField": "uuid",
			"as":           "auth",
		},
	}

	authObject := bson.M{
		"$addFields": bson.M{
			"auth": bson.M{
				"$arrayElemAt": bson.A{
					"$auth",
					0,
				},
			},
		},
	}

	metaObject := bson.M{
		"$addFields": bson.M{
			"meta": bson.M{
				"$arrayElemAt": bson.A{
					"$meta",
					0,
				},
			},
		},
	}

	userObject := bson.M{
		"$addFields": bson.M{
			"applicant": bson.M{
				"$mergeObjects": bson.A{
					"$auth",
					"$meta",
				},
			},
		},
	}

	cleanUp := bson.M{
		"$project": bson.M{
			"meta":            0,
			"auth":            0,
			"uuid":            0,
			"club_id":         0,
			"applicant._id":   0,
			"applicant.email": 0,
			"applicant.token": 0,
		},
	}

	pipeline := bson.A{
		filter,
		metaLookup,
		authLookup,
		authObject,
		metaObject,
		userObject,
		cleanUp,
	}

	cur, err := database.ClubApplicationCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}

	var response []models.ClubApplication
	for cur.Next(context.TODO()) {
		var app models.ClubApplication
		err := cur.Decode(&app)
		if err != nil {
			database.Logger.Error("failed to decode event", err)
		}
		response = append(response, app)
	}

	return &response, nil
}

// FindUserClubs fetches all clubs that a user is a member of
func FindUserClubs(ctx context.Context, userID string, database *database.Database) (*[]models.Club, error) {
	// Start from the club members collection
	pipeline := bson.A{
		// Match documents for the specified user
		bson.M{
			"$match": bson.M{
				"user_id": userID,
			},
		},
		// Lookup the club documents
		bson.M{
			"$lookup": bson.M{
				"from":         "clubs",
				"localField":   "club_id",
				"foreignField": "_id",
				"as":           "club",
			},
		},
		// Unwind the club array (will be a single item)
		bson.M{
			"$unwind": "$club",
		},
		// Replace the root with the club object
		bson.M{
			"$replaceRoot": bson.M{
				"newRoot": "$club",
			},
		},
	}

	// Append the core club pipeline to enrich the club data
	corePipeline := BuildClubCorePipeline()
	completePipeline := append(pipeline, corePipeline...)

	// Execute the aggregation
	cur, err := database.ClubMembersCollection.Aggregate(ctx, completePipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	// Process the results
	var clubs []models.Club
	for cur.Next(ctx) {
		var club models.Club
		err := cur.Decode(&club)
		if err != nil {
			database.Logger.Error("Failed to decode club. Error: ", err.Error())
			continue
		}
		clubs = append(clubs, club)
	}

	// Handle case where no clubs were found
	if clubs == nil {
		clubs = []models.Club{}
	}

	return &clubs, nil
}

// BuildClubCorePipeline returns the common aggregation pipeline stages for clubs
func BuildClubCorePipeline() bson.A {
	// Parent organization lookup
	parentPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "organizations",
			"localField":   "parent_id",
			"foreignField": "_id",
			"as":           "organizations",
		},
	}

	// Add parent field
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

	// Lookup for club members
	membersLookupPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "clubMembers",
			"localField":   "_id",
			"foreignField": "club_id",
			"as":           "_club_members",
		},
	}

	// Lookup user data for members
	memberUsersLookupPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "users",
			"localField":   "_club_members.user_id",
			"foreignField": "uuid",
			"as":           "_member_users",
		},
	}

	// Lookup auth data for members to get first/last name
	memberAuthLookupPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "auth",
			"localField":   "_club_members.user_id",
			"foreignField": "uuid",
			"as":           "_member_auth",
		},
	}

	// Map users to members with auth data
	mapMembersUsersPipeline := bson.M{
		"$addFields": bson.M{
			"members": bson.M{
				"$map": bson.M{
					"input": "$_club_members",
					"as":    "member",
					"in": bson.M{
						// Use _id instead of id to match Member struct
						"_id":       "$$member._id",
						"role":      "$$member.role",
						"joined_at": "$$member.joined_at",
						"user": bson.M{
							"$let": bson.M{
								"vars": bson.M{
									"userData": bson.M{
										"$arrayElemAt": bson.A{
											bson.M{
												"$filter": bson.M{
													"input": "$_member_users",
													"as":    "mu",
													"cond": bson.M{
														"$eq": bson.A{
															"$$mu.uuid",
															"$$member.user_id",
														},
													},
												},
											},
											0,
										},
									},
									"authData": bson.M{
										"$arrayElemAt": bson.A{
											bson.M{
												"$filter": bson.M{
													"input": "$_member_auth",
													"as":    "ma",
													"cond": bson.M{
														"$eq": bson.A{
															"$$ma.uuid",
															"$$member.user_id",
														},
													},
												},
											},
											0,
										},
									},
								},
								"in": bson.M{
									"$mergeObjects": bson.A{
										"$$userData",
										bson.M{
											"first_name": "$$authData.first_name",
											"last_name":  "$$authData.last_name",
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

	// If parent organization exists, lookup its members
	parentMembersLookupPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "organizationMembers",
			"localField":   "parent_id",
			"foreignField": "organization_id",
			"as":           "_parent_members",
		},
	}

	// Lookup user data for parent organization members
	parentMemberUsersLookupPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "users",
			"localField":   "_parent_members.user_id",
			"foreignField": "uuid",
			"as":           "_parent_member_users",
		},
	}

	// Lookup auth data for parent organization members
	parentMemberAuthLookupPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "auth",
			"localField":   "_parent_members.user_id",
			"foreignField": "uuid",
			"as":           "_parent_member_auth",
		},
	}

	// Add parent.members field with user data
	addParentMembersPipeline := bson.M{
		"$addFields": bson.M{
			"parent.members": bson.M{
				"$map": bson.M{
					"input": "$_parent_members",
					"as":    "pm",
					"in": bson.M{
						"_id":       "$$pm._id",
						"role":      "$$pm.role",
						"joined_at": "$$pm.joined_at",
						"user": bson.M{
							"$let": bson.M{
								"vars": bson.M{
									"userData": bson.M{
										"$arrayElemAt": bson.A{
											bson.M{
												"$filter": bson.M{
													"input": "$_parent_member_users",
													"as":    "pmu",
													"cond": bson.M{
														"$eq": bson.A{
															"$$pmu.uuid",
															"$$pm.user_id",
														},
													},
												},
											},
											0,
										},
									},
									"authData": bson.M{
										"$arrayElemAt": bson.A{
											bson.M{
												"$filter": bson.M{
													"input": "$_parent_member_auth",
													"as":    "pma",
													"cond": bson.M{
														"$eq": bson.A{
															"$$pma.uuid",
															"$$pm.user_id",
														},
													},
												},
											},
											0,
										},
									},
								},
								"in": bson.M{
									"$mergeObjects": bson.A{
										"$$userData",
										bson.M{
											"first_name": "$$authData.first_name",
											"last_name":  "$$authData.last_name",
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

	// Project to clean up temporary fields and limit exposed data
	projectPipeline := bson.M{
		"$project": bson.M{
			"_club_members":                     0,
			"_member_users":                     0,
			"_member_auth":                      0,
			"_parent_members":                   0,
			"_parent_member_users":              0,
			"_parent_member_auth":               0,
			"organizations":                     0,
			"parent.members.user._id":           0,
			"parent.members.user.clubs":         0,
			"parent.members.user.sports":        0,
			"parent.members.user.visibility":    0,
			"parent.members.user.device_token":  0,
			"parent.members.user.organizations": 0,
		},
	}

	return bson.A{
		parentPipeline,
		addParentPipeline,
		membersLookupPipeline,
		memberUsersLookupPipeline,
		memberAuthLookupPipeline,
		mapMembersUsersPipeline,
		parentMembersLookupPipeline,
		parentMemberUsersLookupPipeline,
		parentMemberAuthLookupPipeline,
		addParentMembersPipeline,
		projectPipeline,
	}
}
