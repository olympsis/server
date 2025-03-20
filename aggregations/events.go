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

/*
Find an event's data
*/
func AggregateEvent(id primitive.ObjectID, database *database.Database) (*models.Event, error) {

	ctx := context.Background()

	// filter out all docs by our ID
	idPipeline := bson.M{
		"$match": bson.M{
			"_id": id,
		},
	}

	// find poster data
	posterPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "users",
			"localField":   "poster",
			"foreignField": "uuid",
			"as":           "_poster",
		},
	}

	// add poster data correctly to document
	addPosterPipeline := bson.M{
		"$addFields": bson.M{
			"poster": bson.M{
				"$arrayElemAt": bson.A{
					"$_poster",
					0,
				},
			},
		},
	}

	// find participants data
	participantsPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "users",
			"localField":   "participants.uuid",
			"foreignField": "uuid",
			"as":           "_participants",
		},
	}

	// add participants data to document correctly
	addParticipantsPipeline := bson.M{
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
									"$arrayElemAt": bson.A{
										bson.M{
											"$filter": bson.M{
												"input": "$_participants",
												"as":    "pa",
												"cond": bson.M{
													"$eq": bson.A{
														"$$pa.uuid",
														"$$participant.uuid",
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

	// New wait_list pipelines
	waitListPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "users",
			"localField":   "wait_list.uuid",
			"foreignField": "uuid",
			"as":           "_wait_list",
		},
	}

	// Add wait-listed participants data to document
	addWaitListPipeline := bson.M{
		"$addFields": bson.M{
			"wait_list": bson.M{
				"$map": bson.M{
					"input": "$wait_list",
					"as":    "wait_list_participant",
					"in": bson.M{
						"$mergeObjects": bson.A{
							"$$wait_list_participant",
							bson.M{
								"user": bson.M{
									"$arrayElemAt": bson.A{
										bson.M{
											"$filter": bson.M{
												"input": "$_wait_list",
												"as":    "wl",
												"cond": bson.M{
													"$eq": bson.A{
														"$$wl.uuid",
														"$$wait_list_participant.uuid",
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

	// find field data
	fieldPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "fields",
			"localField":   "field._id",
			"foreignField": "_id",
			"as":           "field_data",
		},
	}

	// add field data to document correctly
	addFieldPipeline := bson.M{
		"$addFields": bson.M{
			"field_data": bson.M{
				"$arrayElemAt": bson.A{
					"$field_data",
					0,
				},
			},
		},
	}

	// find clubs data
	clubsPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "clubs",
			"localField":   "organizers._id",
			"foreignField": "_id",
			"as":           "clubs",
		},
	}

	// find orgs data
	orgsPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "organizations",
			"localField":   "organizers._id",
			"foreignField": "_id",
			"as":           "organizations",
		},
	}

	// clean up data
	projectPipeline := bson.M{
		"$project": bson.M{
			"_poster":                         0,
			"poster._id":                      0,
			"poster.clubs":                    0,
			"poster.sports":                   0,
			"poster.visibility":               0,
			"poster.device_token":             0,
			"poster.organizations":            0,
			"participants.uuid":               0,
			"participants.user._id":           0,
			"participants.user.clubs":         0,
			"participants.user.sports":        0,
			"participants.user.visibility":    0,
			"participants.user.device_token":  0,
			"participants.user.organizations": 0,
			"_participants":                   0,
			"clubs.members":                   0,
			"organizations.managers":          0,
		},
	}

	// complete pipeline
	pipeline := bson.A{
		idPipeline,
		posterPipeline,
		addPosterPipeline,
		participantsPipeline,
		addParticipantsPipeline,
		waitListPipeline,
		addWaitListPipeline,
		fieldPipeline,
		addFieldPipeline,
		clubsPipeline,
		orgsPipeline,
		projectPipeline,
	}

	cur, err := database.EventsCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}

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

/*
Find event data by field id
*/
func AggregateEventsByField(id primitive.ObjectID, limit int, database *database.Database) (*[]models.Event, error) {

	ctx := context.Background()

	// filter out all docs by our ID
	idPipeline := bson.M{
		"$match": bson.M{
			"venues._id": id,
		},
	}

	// find poster data
	posterPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "users",
			"localField":   "poster",
			"foreignField": "uuid",
			"as":           "_poster",
		},
	}

	// add poster data correctly to document
	addPosterPipeline := bson.M{
		"$addFields": bson.M{
			"poster": bson.M{
				"$arrayElemAt": bson.A{
					"$_poster",
					0,
				},
			},
		},
	}

	// find participants data
	participantsPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "users",
			"localField":   "participants.uuid",
			"foreignField": "uuid",
			"as":           "_participants",
		},
	}

	// add participants data to document correctly
	addParticipantsPipeline := bson.M{
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
									"$arrayElemAt": bson.A{
										bson.M{
											"$filter": bson.M{
												"input": "$_participants",
												"as":    "pa",
												"cond": bson.M{
													"$eq": bson.A{
														"$$pa.uuid",
														"$$participant.uuid",
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

	// New wait_list pipelines
	waitListPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "users",
			"localField":   "wait_list.uuid",
			"foreignField": "uuid",
			"as":           "_wait_list",
		},
	}

	// Add wait-listed participants data to document
	addWaitListPipeline := bson.M{
		"$addFields": bson.M{
			"wait_list": bson.M{
				"$map": bson.M{
					"input": "$wait_list",
					"as":    "wait_list_participant",
					"in": bson.M{
						"$mergeObjects": bson.A{
							"$$wait_list_participant",
							bson.M{
								"user": bson.M{
									"$arrayElemAt": bson.A{
										bson.M{
											"$filter": bson.M{
												"input": "$_wait_list",
												"as":    "wl",
												"cond": bson.M{
													"$eq": bson.A{
														"$$wl.uuid",
														"$$wait_list_participant.uuid",
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

	// find field data
	fieldPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "fields",
			"localField":   "field._id",
			"foreignField": "_id",
			"as":           "field_data",
		},
	}

	// add field data to document correctly
	addFieldPipeline := bson.M{
		"$addFields": bson.M{
			"field_data": bson.M{
				"$arrayElemAt": bson.A{
					"$field_data",
					0,
				},
			},
		},
	}

	// find clubs data
	clubsPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "clubs",
			"localField":   "organizers._id",
			"foreignField": "_id",
			"as":           "clubs",
		},
	}

	// find orgs data
	orgsPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "organizations",
			"localField":   "organizers._id",
			"foreignField": "_id",
			"as":           "organizations",
		},
	}

	// clean up data
	projectPipeline := bson.M{
		"$project": bson.M{
			"_poster":                         0,
			"poster._id":                      0,
			"poster.clubs":                    0,
			"poster.sports":                   0,
			"poster.visibility":               0,
			"poster.device_token":             0,
			"poster.organizations":            0,
			"participants.uuid":               0,
			"participants.user._id":           0,
			"participants.user.clubs":         0,
			"participants.user.sports":        0,
			"participants.user.visibility":    0,
			"participants.user.device_token":  0,
			"participants.user.organizations": 0,
			"_participants":                   0,
			"clubs.members":                   0,
			"organizations.managers":          0,
		},
	}

	// limit documents returned
	limitPipeline := bson.M{
		"$limit": limit,
	}

	// complete pipeline
	pipeline := bson.A{
		idPipeline,
		limitPipeline,
		posterPipeline,
		addPosterPipeline,
		participantsPipeline,
		addParticipantsPipeline,
		waitListPipeline,
		addWaitListPipeline,
		fieldPipeline,
		addFieldPipeline,
		clubsPipeline,
		orgsPipeline,
		projectPipeline,
	}

	cur, err := database.EventsCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}

	var response []models.Event
	for cur.Next(context.TODO()) {
		var event models.Event
		err := cur.Decode(&event)
		if err != nil {
			database.Logger.Error(err)
		}

		response = append(response, event)
	}

	return &response, nil
}

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

	// Lookup for participants - separate collection now
	participantsLookupPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "eventParticipants",
			"localField":   "_id",
			"foreignField": "event_id",
			"as":           "event_participants",
		},
	}

	// Lookup for teams - separate collection
	teamsLookupPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "eventTeams",
			"localField":   "_id",
			"foreignField": "event_id",
			"as":           "event_teams",
		},
	}

	// Calculate counts and add to document
	addCountsPipeline := bson.M{
		"$addFields": bson.M{
			"participants_count": bson.M{"$size": "$event_participants"},
			"teams_count":        bson.M{"$size": "$event_teams"},
		},
	}

	// Private events where user is participant (only if userID is provided)
	if userID != nil && *userID != "" {
		visibilityConditions = append(visibilityConditions, bson.M{
			"visibility":                 2,
			"event_participants.user_id": *userID,
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

	// Enhanced project pipeline to shape the final document
	projectPipeline := bson.M{
		"$project": bson.M{
			"_id":                 1,
			"poster_id":           1,
			"organizers":          1,
			"venues":              1,
			"media_url":           1,
			"media_type":          1,
			"title":               1,
			"body":                1,
			"sports":              1,
			"format_config":       1,
			"start_time":          1,
			"stop_time":           1,
			"participants_count":  1,
			"participants_config": 1,
			"teams_count":         1,
			"teams_config":        1,
			"visibility":          1,
			"external_link":       1,
			"is_sensitive":        1,
			"created_at":          1,
			"updated_at":          1,
			"cancelled_at":        1,
			"recurrence_config":   1,
		},
	}

	// complete pipeline
	pipeline := bson.A{
		timePipeline,
		cancelledPipeline,
		filterPipeline,
		participantsLookupPipeline, // Add participants lookup early
		teamsLookupPipeline,        // Add teams lookup early
		addCountsPipeline,          // Calculate counts
		visibilityPipeline,         // Apply visibility filter after lookups
		skipPipeline,
		limitPipeline,
		projectPipeline,
	}

	return pipeline
}
