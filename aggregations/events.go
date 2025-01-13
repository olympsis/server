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

	cur, err := database.EventCol.Aggregate(ctx, pipeline)
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
	uuid string,
	sports []string,
	location models.GeoJSON,
	venues []primitive.ObjectID,
	clubs []primitive.ObjectID,
	orgs []primitive.ObjectID,
	radius int,
	limit int,
	skip int,
	status int,
	database *database.Database,
) (*[]models.Event, error) {

	ctx := context.TODO()

	/**
	EVENT STATUS PIPELINE

	0 - completed
	1 - pending/live
	*/
	var timePipeline bson.M
	switch status {
	case 0:
		timePipeline = bson.M{
			"$match": bson.M{
				"stop_time": bson.M{
					"$lte": time.Now().Unix(),
				},
			},
		}
	default:
		timePipeline = bson.M{
			"$match": bson.M{
				"stop_time": bson.M{
					"$gte": time.Now().Unix(),
				},
			},
		}
	}

	// venues & sports filter
	filterPipeline := bson.M{
		"$match": bson.M{
			"$and": bson.A{
				bson.M{ // sports filter
					"sports": bson.M{
						"$in": sports,
					},
				},
				bson.M{ // venue location
					"$or": bson.A{
						bson.M{
							"venues._id": bson.M{
								"$exists": true,
								"$in":     venues,
							},
						},
						bson.M{
							"venues.location": bson.M{
								"$geoWithin": bson.M{
									"$center": bson.A{
										location.Coordinates,
										radius,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// only events that are public, or that the user is a club member of or is a participant of
	visibilityPipeline := bson.M{
		"$match": bson.M{
			"$or": bson.A{
				// Public events
				bson.M{
					"visibility": 0,
				},
				// Club/Org member events
				bson.M{
					"visibility": 1,
					"organizers._id": bson.M{
						"$in": clubs,
					},
				},
				bson.M{
					"visibility": 1,
					"organizers._id": bson.M{
						"$in": orgs,
					},
				},
				// Private events where user is participant
				bson.M{
					"visibility":        2,
					"participants.uuid": uuid,
				},
			},
		},
	}

	// skip documents
	skipPipeline := bson.M{
		"$skip": skip,
	}

	// limit documents returned
	limitPipeline := bson.M{
		"$limit": limit,
	}

	// find the poster data
	posterPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "users",
			"localField":   "poster",
			"foreignField": "uuid",
			"as":           "_poster",
		},
	}

	// add poster data to document correctly
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

	// remove unnecessary data
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
		timePipeline,
		filterPipeline,
		visibilityPipeline,
		skipPipeline,
		limitPipeline,
		posterPipeline,
		addPosterPipeline,
		participantsPipeline,
		addParticipantsPipeline,
		waitListPipeline,
		addWaitListPipeline,
		projectPipeline,
	}

	cur, err := database.EventCol.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

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

	cur, err := database.EventCol.Aggregate(ctx, pipeline)
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
