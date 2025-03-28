package aggregations

import (
	"context"
	"olympsis-server/database"
	"time"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// AggregateEvent gets a single event by ID with all related data
func AggregateEvent(id primitive.ObjectID, database *database.Database) (*models.Event, error) {
	ctx := context.Background()

	// Create ID filter pipeline stage
	idPipeline := bson.M{
		"$match": bson.M{
			"_id": id,
		},
	}

	// Get the core pipeline stages
	corePipeline := BuildEventCorePipeline()

	// Insert the ID filter at the beginning of the pipeline
	completePipeline := append(bson.A{idPipeline}, corePipeline...)

	// Execute the aggregation
	cur, err := database.EventsCollection.Aggregate(ctx, completePipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var event models.Event
	if cur.Next(ctx) {
		err = cur.Decode(&event)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, mongo.ErrNoDocuments
	}

	return &event, nil
}

// AggregateEvents fetches multiple events based on filter criteria
func AggregateEvents(
	userID *string,
	sports *[]string,
	location *models.GeoJSON,
	venues *[]primitive.ObjectID,
	clubs *[]primitive.ObjectID,
	orgs *[]primitive.ObjectID,
	radius int,
	limit int,
	skip int,
	database *database.Database,
) (*[]models.Event, error) {
	ctx := context.TODO()

	// Build the aggregation pipeline
	pipeline := BuildEventsAggregation(userID, sports, location, venues, clubs, orgs, radius, limit, skip)

	// Execute the aggregation
	cur, err := database.EventsCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	// Process the results
	response := make([]models.Event, 0, limit)
	for cur.Next(ctx) {
		var event models.Event
		err := cur.Decode(&event)
		if err != nil {
			database.Logger.Error("Failed to decode event. Error: ", err.Error())
			continue
		}
		response = append(response, event)
	}

	return &response, nil
}

// Builds a pipeline for handling the event object itself
func BuildEventCorePipeline() bson.A {
	// Lookup for poster user data
	posterLookupPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "users",
			"localField":   "poster",
			"foreignField": "uuid",
			"as":           "_poster_user",
		},
	}

	// Lookup auth data for poster to get first/last name
	posterAuthLookupPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "auth",
			"localField":   "poster",
			"foreignField": "uuid",
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
								"uuid":      bson.M{"$arrayElemAt": bson.A{"$_poster_user.uuid", 0}},
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

	// Lookup for participants
	participantsLookupPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "eventParticipants",
			"localField":   "_id",
			"foreignField": "event_id",
			"as":           "participants",
		},
	}

	// Lookup user data for participants
	participantUsersLookupPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "users",
			"localField":   "participants.user_id",
			"foreignField": "uuid",
			"as":           "_participant_users",
		},
	}

	// Lookup auth data for participants to get first/last name
	participantAuthLookupPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "auth",
			"localField":   "participants.user_id",
			"foreignField": "uuid",
			"as":           "_participant_auth",
		},
	}

	// Map users to participants
	mapParticipantsUsersPipeline := bson.M{
		"$addFields": bson.M{
			"participants": bson.M{
				"$map": bson.M{
					"input": "$participants",
					"as":    "participant",
					"in": bson.M{
						"$mergeObjects": bson.A{
							"$$participant",
							bson.M{
								"user": bson.M{
									"$mergeObjects": bson.A{
										bson.M{
											"$arrayElemAt": bson.A{
												bson.M{
													"$filter": bson.M{
														"input": "$_participant_users",
														"as":    "pu",
														"cond": bson.M{
															"$eq": bson.A{
																"$$pu.uuid",
																"$$participant.user_id",
															},
														},
													},
												},
												0,
											},
										},
										bson.M{
											"first_name": bson.M{
												"$arrayElemAt": bson.A{
													bson.M{
														"$filter": bson.M{
															"input": "$_participant_auth",
															"as":    "pa",
															"cond": bson.M{
																"$eq": bson.A{
																	"$$pa.uuid",
																	"$$participant.user_id",
																},
															},
														},
													},
													0,
													"first_name",
												},
											},
											"last_name": bson.M{
												"$arrayElemAt": bson.A{
													bson.M{
														"$filter": bson.M{
															"input": "$_participant_auth",
															"as":    "pa",
															"cond": bson.M{
																"$eq": bson.A{
																	"$$pa.uuid",
																	"$$participant.user_id",
																},
															},
														},
													},
													0,
													"last_name",
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

	// Create participants waitlist pipeline
	participantsWaitlistPipeline := bson.M{
		"$addFields": bson.M{
			"participants_waitlist": bson.M{
				"$filter": bson.M{
					"input": "$participants",
					"as":    "p",
					"cond": bson.M{
						"$eq": bson.A{"$$p.status", 2}, // Assuming 2 is waitlist status
					},
				},
			},
		},
	}

	// Filter regular participants to exclude waitlist
	filterRegularParticipantsPipeline := bson.M{
		"$addFields": bson.M{
			"participants": bson.M{
				"$filter": bson.M{
					"input": "$participants",
					"as":    "p",
					"cond": bson.M{
						"$ne": bson.A{"$$p.status", 2}, // Exclude waitlist status
					},
				},
			},
		},
	}

	// Lookup for comments
	commentsLookupPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "eventComments",
			"localField":   "_id",
			"foreignField": "event_id",
			"as":           "comments",
		},
	}

	// Lookup user data for comments
	commentUsersLookupPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "users",
			"localField":   "comments.user_id",
			"foreignField": "uuid",
			"as":           "_comment_users",
		},
	}

	// Lookup auth data for comment users to get first/last name
	commentAuthLookupPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "auth",
			"localField":   "comments.user_id",
			"foreignField": "uuid",
			"as":           "_comment_auth",
		},
	}

	// Map users to comments
	mapCommentsUsersPipeline := bson.M{
		"$addFields": bson.M{
			"comments": bson.M{
				"$map": bson.M{
					"input": "$comments",
					"as":    "comment",
					"in": bson.M{
						"$mergeObjects": bson.A{
							"$$comment",
							bson.M{
								"user": bson.M{
									"$mergeObjects": bson.A{
										bson.M{
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
										bson.M{
											"first_name": bson.M{
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
													"first_name",
												},
											},
											"last_name": bson.M{
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
													"last_name",
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

	// Lookup for teams
	teamsLookupPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "eventTeams",
			"localField":   "_id",
			"foreignField": "event_id",
			"as":           "teams",
		},
	}

	// Project the result structure
	projectPipeline := bson.M{
		"$project": bson.M{
			"_id":                   1,
			"poster":                1,
			"organizers":            1,
			"venues":                1,
			"media_url":             1,
			"media_type":            1,
			"title":                 1,
			"body":                  1,
			"sports":                1,
			"tags":                  1,
			"format_config":         1,
			"start_time":            1,
			"stop_time":             1,
			"participants":          1,
			"participants_waitlist": 1,
			"participants_count":    1,
			"participants_config":   1,
			"teams":                 1,
			"teams_waitlist":        bson.M{"$literal": []any{}}, // Placeholder
			"teams_count":           1,
			"teams_config":          1,
			"comments":              1,
			"visibility":            1,
			"external_link":         1,
			"is_sensitive":          1,
			"created_at":            1,
			"updated_at":            1,
			"cancelled_at":          1,
			"recurrence_config":     1,
		},
	}

	return bson.A{
		posterLookupPipeline,
		posterAuthLookupPipeline,
		createPosterSnippetPipeline,
		participantsLookupPipeline,
		participantUsersLookupPipeline,
		participantAuthLookupPipeline,
		mapParticipantsUsersPipeline,
		participantsWaitlistPipeline,
		filterRegularParticipantsPipeline,
		teamsLookupPipeline,
		commentsLookupPipeline,
		commentUsersLookupPipeline,
		commentAuthLookupPipeline,
		mapCommentsUsersPipeline,
		projectPipeline,
	}
}

// Builds a pipeline for filtering and returning multiple events
func BuildEventsAggregation(
	userID *string,
	sports *[]string,
	location *models.GeoJSON,
	venues *[]primitive.ObjectID,
	clubs *[]primitive.ObjectID,
	orgs *[]primitive.ObjectID,
	radius int,
	limit int,
	skip int,
) bson.A {
	// Get current time as primitive.DateTime for comparison
	currentTime := primitive.NewDateTimeFromTime(time.Now())

	// Pipeline to get events that have not ended yet (future events)
	timePipeline := bson.M{
		"$match": bson.M{
			"stop_time": bson.M{
				"$gte": currentTime,
			},
		},
	}

	// Handle cancelled events - exclude them
	cancelledPipeline := bson.M{
		"$match": bson.M{
			"$or": bson.A{
				bson.M{"cancelled_at": bson.M{"$exists": false}},
				bson.M{"cancelled_at": nil},
			},
		},
	}

	// Build filter pipeline based on provided parameters
	filterConditions := bson.A{}

	// Add sports filter if provided
	if sports != nil && len(*sports) > 0 {
		filterConditions = append(filterConditions, bson.M{
			"sports": bson.M{
				"$in": *sports,
			},
		})
	}

	// Build venue location filter if either venues or location is provided
	venueConditions := bson.A{}

	if venues != nil && len(*venues) > 0 {
		venueConditions = append(venueConditions, bson.M{
			"venues._id": bson.M{
				"$exists": true,
				"$in":     *venues,
			},
		})
	}

	if location != nil {
		venueConditions = append(venueConditions, bson.M{
			"venues.location": bson.M{
				"$geoWithin": bson.M{
					"$center": bson.A{
						location.Coordinates,
						radius,
					},
				},
			},
		})
	}

	// Add venue conditions if any exist
	if len(venueConditions) > 0 {
		filterConditions = append(filterConditions, bson.M{
			"$or": venueConditions,
		})
	}

	// Create filter pipeline if we have any conditions
	var filterPipeline bson.M
	if len(filterConditions) > 0 {
		filterPipeline = bson.M{
			"$match": bson.M{
				"$and": filterConditions,
			},
		}
	} else {
		// If no filters, just create an empty match to maintain pipeline structure
		filterPipeline = bson.M{
			"$match": bson.M{},
		}
	}

	// Build visibility pipeline
	visibilityConditions := bson.A{
		// Public events are always visible
		bson.M{
			"visibility": 0,
		},
	}

	// Club/Org member events
	if clubs != nil && len(*clubs) > 0 {
		visibilityConditions = append(visibilityConditions, bson.M{
			"visibility": 1,
			"organizers._id": bson.M{
				"$in": *clubs,
			},
		})
	}

	if orgs != nil && len(*orgs) > 0 {
		visibilityConditions = append(visibilityConditions, bson.M{
			"visibility": 1,
			"organizers._id": bson.M{
				"$in": *orgs,
			},
		})
	}

	// Private events where user is participant (only if userID is provided)
	if userID != nil && *userID != "" {
		visibilityConditions = append(visibilityConditions, bson.M{
			"visibility":           2,
			"participants.user_id": *userID,
		})
	}

	visibilityPipeline := bson.M{
		"$match": bson.M{
			"$or": visibilityConditions,
		},
	}

	// Pagination pipelines
	skipPipeline := bson.M{
		"$skip": skip,
	}

	limitPipeline := bson.M{
		"$limit": limit,
	}

	// Get the core pipeline stages
	corePipeline := BuildEventCorePipeline()

	// Complete pipeline with filtering and pagination
	// First add the pre-lookup filters
	completePipeline := bson.A{
		timePipeline,
		cancelledPipeline,
		filterPipeline,
	}

	// Then add the core lookups
	completePipeline = append(completePipeline, corePipeline...)

	// Finally add the post-lookup stages (visibility filtering and pagination)
	completePipeline = append(completePipeline,
		visibilityPipeline,
		skipPipeline,
		limitPipeline,
	)

	return completePipeline
}
