package aggregations

import (
	"context"
	"olympsis-server/database"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// AggregateOrganization gets a single organization by ID with all related data
func AggregateOrganization(id primitive.ObjectID, database *database.Database) (*models.Organization, error) {
	ctx := context.Background()

	// Create ID filter pipeline stage
	idPipeline := bson.M{
		"$match": bson.M{
			"_id": id,
		},
	}

	// Get the core pipeline stages
	corePipeline := BuildOrganizationCorePipeline()

	// Insert the ID filter at the beginning of the pipeline
	completePipeline := append(bson.A{idPipeline}, corePipeline...)

	// Execute the aggregation
	cur, err := database.OrgCollection.Aggregate(ctx, completePipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

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

// AggregateOrganizations fetches multiple organizations based on filter criteria
func AggregateOrganizations(
	filter bson.M,
	limit int,
	skip int,
	database *database.Database,
) (*[]models.Organization, error) {
	ctx := context.TODO()

	// Get the core pipeline stages
	corePipeline := BuildOrganizationCorePipeline()

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
	cur, err := database.OrgCollection.Aggregate(ctx, completePipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	// Process the results
	response := make([]models.Organization, 0, limit)
	for cur.Next(ctx) {
		var org models.Organization
		err := cur.Decode(&org)
		if err != nil {
			database.Logger.Error("Failed to decode organization. Error: ", err.Error())
			continue
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

	cur, err := database.OrgApplicationCollection.Aggregate(ctx, pipeline)
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

	cur, err := database.OrgApplicationCollection.Aggregate(ctx, pipeline)
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

// FindUserOrganizations fetches all organizations that a user is a member of
func FindUserOrganizations(ctx context.Context, userID string, database *database.Database) (*[]models.Organization, error) {
	// Start from the organization members collection
	pipeline := bson.A{
		// Match documents for the specified user
		bson.M{
			"$match": bson.M{
				"user_id": userID,
			},
		},
		// Lookup the organization documents
		bson.M{
			"$lookup": bson.M{
				"from":         "organizations",
				"localField":   "organization_id",
				"foreignField": "_id",
				"as":           "organization",
			},
		},
		// Unwind the organization array (will be a single item)
		bson.M{
			"$unwind": "$organization",
		},
		// Replace the root with the organization object
		bson.M{
			"$replaceRoot": bson.M{
				"newRoot": "$organization",
			},
		},
	}

	// Append the core organization pipeline to enrich the organization data
	corePipeline := BuildOrganizationCorePipeline()
	completePipeline := append(pipeline, corePipeline...)

	// Execute the aggregation
	cur, err := database.OrganizationMembersCollection.Aggregate(ctx, completePipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	// Process the results
	var organizations []models.Organization
	for cur.Next(ctx) {
		var org models.Organization
		err := cur.Decode(&org)
		if err != nil {
			database.Logger.Error("Failed to decode organization. Error: ", err.Error())
			continue
		}
		organizations = append(organizations, org)
	}

	// Handle case where no organizations were found
	if organizations == nil {
		organizations = []models.Organization{}
	}

	return &organizations, nil
}

// BuildOrganizationCorePipeline returns the common aggregation pipeline stages for organizations
func BuildOrganizationCorePipeline() bson.A {
	// Lookup for organization members
	membersLookupPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "organizationMembers",
			"localField":   "_id",
			"foreignField": "organization_id",
			"as":           "_organization_members",
		},
	}

	// Lookup user data for members
	memberUsersLookupPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "users",
			"localField":   "_organization_members.user_id",
			"foreignField": "uuid",
			"as":           "_member_users",
		},
	}

	// Lookup auth data for members to get first/last name
	memberAuthLookupPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "auth",
			"localField":   "_organization_members.user_id",
			"foreignField": "uuid",
			"as":           "_member_auth",
		},
	}

	// Map users to members with auth data
	mapMembersUsersPipeline := bson.M{
		"$addFields": bson.M{
			"members": bson.M{
				"$map": bson.M{
					"input": "$_organization_members",
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

	// Lookup child clubs
	childrenPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "clubs",
			"localField":   "_id",
			"foreignField": "parent_id",
			"as":           "children",
		},
	}

	// Project to clean up temporary fields and limit exposed data
	projectPipeline := bson.M{
		"$project": bson.M{
			"_organization_members":      0,
			"_member_users":              0,
			"_member_auth":               0,
			"users":                      0,
			"members.user._id":           0,
			"members.user.token":         0,
			"members.user.clubs":         0,
			"members.user.sports":        0,
			"members.user.visibility":    0,
			"members.user.device_token":  0,
			"members.user.organizations": 0,
		},
	}

	return bson.A{
		membersLookupPipeline,
		memberUsersLookupPipeline,
		memberAuthLookupPipeline,
		mapMembersUsersPipeline,
		childrenPipeline,
		projectPipeline,
	}
}
