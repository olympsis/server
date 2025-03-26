package aggregations

import (
	"context"
	"olympsis-server/database"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// AggregateClub gets a single club by ID with all related data
func AggregateClub(id primitive.ObjectID, database *database.Database) (*models.Club, error) {
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
	cur, err := database.ClubCol.Aggregate(ctx, completePipeline)
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
	limit int,
	skip int,
	database *database.Database,
) (*[]models.Club, error) {
	ctx := context.TODO()

	// Get the core pipeline stages
	corePipeline := BuildClubCorePipeline()

	// Add filter and pagination
	filterPipeline := bson.M{
		"$match": filter,
	}

	skipPipeline := bson.M{
		"$skip": skip,
	}

	limitPipeline := bson.M{
		"$limit": limit,
	}

	// Complete pipeline with filtering and pagination
	completePipeline := append(bson.A{filterPipeline}, corePipeline...)
	completePipeline = append(completePipeline, skipPipeline, limitPipeline)

	// Execute the aggregation
	cur, err := database.ClubCol.Aggregate(ctx, completePipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	// Process the results
	response := make([]models.Club, 0, limit)
	for cur.Next(ctx) {
		var club models.Club
		err := cur.Decode(&club)
		if err != nil {
			database.Logger.Error("Failed to decode club. Error: ", err.Error())
			continue
		}
		response = append(response, club)
	}

	return &response, nil
}

func AggregateClubApplication(id *primitive.ObjectID, database *database.Database) (*models.ClubApplication, error) {

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

	cur, err := database.ClubApplicationCol.Aggregate(ctx, pipeline)
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

func AggregateClubApplications(clubId *primitive.ObjectID, status string, database *database.Database) (*[]models.ClubApplication, error) {

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

	cur, err := database.ClubApplicationCol.Aggregate(ctx, pipeline)
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

	// Map users to members
	mapMembersUsersPipeline := bson.M{
		"$addFields": bson.M{
			"members": bson.M{
				"$map": bson.M{
					"input": "$_club_members",
					"as":    "member",
					"in": bson.M{
						"id":        "$$member._id",
						"role":      "$$member.role",
						"joined_at": "$$member.joined_at",
						"user": bson.M{
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
					},
				},
			},
		},
	}

	// If parent organization exists, lookup its members
	managersPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "users",
			"localField":   "parent.members.uuid",
			"foreignField": "uuid",
			"as":           "managers",
		},
	}

	// Add member user data to parent organization's members
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

	// Project to clean up temporary fields and limit exposed data
	projectPipeline := bson.M{
		"$project": bson.M{
			"_club_members":                     0,
			"_member_users":                     0,
			"users":                             0,
			"managers":                          0,
			"organizations":                     0,
			"parent.members.uuid":               0,
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
		mapMembersUsersPipeline,
		managersPipeline,
		addManagersPipeline,
		projectPipeline,
	}
}
