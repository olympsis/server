package aggregations

import (
	"context"
	"olympsis-server/database"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// AggregatePost gets a single post by ID with all related data
func AggregatePost(id bson.ObjectID, database *database.Database) (*models.Post, error) {
	ctx := context.Background()

	// Create ID filter pipeline stage
	idPipeline := bson.M{
		"$match": bson.M{
			"_id": id,
		},
	}

	// Get the core pipeline stages
	corePipeline := BuildPostCorePipeline()

	// Insert the ID filter at the beginning of the pipeline
	completePipeline := append(bson.A{idPipeline}, corePipeline...)

	// Execute the aggregation
	cur, err := database.PostsCollection.Aggregate(ctx, completePipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var post models.Post
	if cur.Next(ctx) {
		err = cur.Decode(&post)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, mongo.ErrNoDocuments
	}

	return &post, nil
}

// AggregatePosts fetches multiple posts based on filter criteria
func AggregatePosts(
	filter bson.M,
	limit int,
	skip int,
	database *database.Database,
) (*[]models.Post, error) {
	ctx := context.TODO()

	// Get the core pipeline stages
	corePipeline := BuildPostCorePipeline()

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

	// Sort by created_at descending (newest first)
	sortPipeline := bson.M{
		"$sort": bson.M{
			"created_at": -1,
		},
	}

	// Complete pipeline with filtering, sorting and pagination
	completePipeline := append(bson.A{filterPipeline, sortPipeline}, corePipeline...)
	completePipeline = append(completePipeline, skipPipeline, limitPipeline)

	// Execute the aggregation
	cur, err := database.PostsCollection.Aggregate(ctx, completePipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	// Process the results
	response := make([]models.Post, 0, limit)
	for cur.Next(ctx) {
		var post models.Post
		err := cur.Decode(&post)
		if err != nil {
			database.Logger.Error("Failed to decode post. Error: ", err.Error())
			continue
		}
		response = append(response, post)
	}

	return &response, nil
}

// BuildPostCorePipeline returns the common aggregation pipeline stages for posts
func BuildPostCorePipeline() bson.A {
	// Lookup for poster user data
	posterLookupPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "users",
			"localField":   "poster",
			"foreignField": "user_id",
			"as":           "_poster_user",
		},
	}

	// Lookup auth data for poster to get first/last name
	posterAuthLookupPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "auth",
			"localField":   "poster",
			"foreignField": "user_id",
			"as":           "_poster_auth",
		},
	}

	// Create poster user snippet
	createPosterSnippetPipeline := bson.M{
		"$addFields": bson.M{
			"poster": bson.M{
				"$cond": bson.A{
					bson.M{"$gt": bson.A{bson.M{"$size": "$_poster_user"}, 0}},
					bson.M{
						"$mergeObjects": bson.A{
							bson.M{
								"user_id":   bson.M{"$arrayElemAt": bson.A{"$_poster_user.uuid", 0}},
								"username":  bson.M{"$arrayElemAt": bson.A{"$_poster_user.username", 0}},
								"image_url": bson.M{"$arrayElemAt": bson.A{"$_poster_user.image_url", 0}},
							},
							bson.M{
								"first_name": bson.M{"$arrayElemAt": bson.A{"$_poster_auth.first_name", 0}},
								"last_name":  bson.M{"$arrayElemAt": bson.A{"$_poster_auth.last_name", 0}},
							},
						},
					},
					nil,
				},
			},
		},
	}

	// Lookup for event if there's an event_id
	eventLookupPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "events",
			"localField":   "event_id",
			"foreignField": "_id",
			"as":           "_event",
		},
	}

	// Add event if it exists
	addEventPipeline := bson.M{
		"$addFields": bson.M{
			"event": bson.M{
				"$cond": bson.A{
					bson.M{"$gt": bson.A{bson.M{"$size": "$_event"}, 0}},
					bson.M{"$arrayElemAt": bson.A{"$_event", 0}},
					nil,
				},
			},
		},
	}

	// Lookup for comments
	commentsLookupPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "postComments",
			"localField":   "_id",
			"foreignField": "post_id",
			"as":           "_comments",
		},
	}

	// Lookup user data for comments
	commentUsersLookupPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "users",
			"localField":   "_comments.user_id",
			"foreignField": "user_id",
			"as":           "_comment_users",
		},
	}

	// Lookup auth data for comment users
	commentAuthLookupPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "auth",
			"localField":   "_comments.user_id",
			"foreignField": "user_id",
			"as":           "_comment_auth",
		},
	}

	// Map users to comments
	mapCommentsUsersPipeline := bson.M{
		"$addFields": bson.M{
			"comments": bson.M{
				"$map": bson.M{
					"input": "$_comments",
					"as":    "comment",
					"in": bson.M{
						"_id":        "$$comment._id",
						"text":       "$$comment.text",
						"created_at": "$$comment.created_at",
						"user": bson.M{
							"$let": bson.M{
								"vars": bson.M{
									"userData": bson.M{
										"$arrayElemAt": bson.A{
											bson.M{
												"$filter": bson.M{
													"input": "$_comment_users",
													"as":    "cu",
													"cond": bson.M{
														"$eq": bson.A{
															"$$cu.uuid",
															"$$comment.user_id",
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
													"input": "$_comment_auth",
													"as":    "ca",
													"cond": bson.M{
														"$eq": bson.A{
															"$$ca.uuid",
															"$$comment.user_id",
														},
													},
												},
											},
											0,
										},
									},
								},
								"in": bson.M{
									"user_id":    "$$userData.uuid",
									"username":   "$$userData.username",
									"image_url":  "$$userData.image_url",
									"first_name": "$$authData.first_name",
									"last_name":  "$$authData.last_name",
								},
							},
						},
					},
				},
			},
		},
	}

	// Lookup for reactions
	reactionsLookupPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "postReactions",
			"localField":   "_id",
			"foreignField": "post_id",
			"as":           "_reactions",
		},
	}

	// Lookup user data for reactions
	reactionUsersLookupPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "users",
			"localField":   "_reactions.user_id",
			"foreignField": "user_id",
			"as":           "_reaction_users",
		},
	}

	// Lookup auth data for reaction users
	reactionAuthLookupPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "auth",
			"localField":   "_reactions.user_id",
			"foreignField": "user_id",
			"as":           "_reaction_auth",
		},
	}

	// Map users to reactions
	mapReactionsUsersPipeline := bson.M{
		"$addFields": bson.M{
			"reactions": bson.M{
				"$map": bson.M{
					"input": "$_reactions",
					"as":    "reaction",
					"in": bson.M{
						"_id":        "$$reaction._id",
						"created_at": "$$reaction.created_at",
						"user": bson.M{
							"$let": bson.M{
								"vars": bson.M{
									"userData": bson.M{
										"$arrayElemAt": bson.A{
											bson.M{
												"$filter": bson.M{
													"input": "$_reaction_users",
													"as":    "ru",
													"cond": bson.M{
														"$eq": bson.A{
															"$$ru.uuid",
															"$$reaction.user_id",
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
													"input": "$_reaction_auth",
													"as":    "ra",
													"cond": bson.M{
														"$eq": bson.A{
															"$$ra.uuid",
															"$$reaction.user_id",
														},
													},
												},
											},
											0,
										},
									},
								},
								"in": bson.M{
									"user_id":    "$$userData.uuid",
									"username":   "$$userData.username",
									"image_url":  "$$userData.image_url",
									"first_name": "$$authData.first_name",
									"last_name":  "$$authData.last_name",
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
			"_poster_user":    0,
			"_poster_auth":    0,
			"_event":          0,
			"_comments":       0,
			"_comment_users":  0,
			"_comment_auth":   0,
			"_reactions":      0,
			"_reaction_users": 0,
			"_reaction_auth":  0,
			"event_id":        0,
		},
	}

	return bson.A{
		posterLookupPipeline,
		posterAuthLookupPipeline,
		createPosterSnippetPipeline,
		eventLookupPipeline,
		addEventPipeline,
		commentsLookupPipeline,
		commentUsersLookupPipeline,
		commentAuthLookupPipeline,
		mapCommentsUsersPipeline,
		reactionsLookupPipeline,
		reactionUsersLookupPipeline,
		reactionAuthLookupPipeline,
		mapReactionsUsersPipeline,
		projectPipeline,
	}
}
