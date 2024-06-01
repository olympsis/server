package service

import (
	"context"
	"olympsis-server/database"
	"time"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// EVENT AGGREGATIONS

/*
Find an event's data
*/
func FindEvent(id primitive.ObjectID, database *database.Database) (*models.Event, error) {

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

/*
Find events data by location
*/
func FindEvents(uuid string, sports []string, fieldIDs []primitive.ObjectID, location models.GeoJSON, radius int, limit int, database *database.Database) (*[]models.Event, error) {

	ctx := context.Background()
	// match events by the field ids and a geo location if they have one
	filterPipeline := bson.M{
		"$match": bson.M{
			"$or": bson.A{
				bson.M{
					"venue._id": bson.M{
						"$exists": true,
						"$in":     fieldIDs,
					},
				},
				bson.M{
					"venue.location": bson.M{
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
	}

	// filter out events by the sport
	sportsPipeline := bson.M{
		"$match": bson.M{
			"sport": bson.M{
				"$in": sports,
			},
		},
	}

	// make sure events have not been stopped or have a stop time that was at least 2hrs ago
	timePipeline := bson.M{
		"$match": bson.M{
			"$or": bson.A{
				bson.M{ // event with stop time
					"actual_stop_time": bson.M{
						"$exists": false,
					},
					"stop_time": bson.M{
						"$gte": time.Now().Add(-time.Hour * 2).Unix(),
					},
				},
				bson.M{ //event with no stop time
					"type": bson.M{
						"$eq": "pickup",
					},
					"actual_stop_time": bson.M{
						"$exists": false,
					},
					"stop_time": bson.M{
						"$exists": false,
					},
					"start_time": bson.M{
						"$gte": time.Now().Add(-time.Hour * 3).Unix(),
					},
				},
			},
		},
	}

	// only events that are public, or that the user is a club member of or is a participant of
	visibilityPipeline := bson.M{
		"$match": bson.M{
			"$or": bson.A{
				bson.M{ // events that anyone can see
					"visibility": "public",
				},
				bson.M{
					"visibility":         "club",
					"clubs.members.uuid": uuid,
				},
				bson.M{ // private events that the user joined
					"visibility":             "private",
					"participants.user.uuid": uuid,
				},
			},
		},
	}

	// sort the documents by the start time
	sortPipeline := bson.M{
		"$sort": bson.M{
			"start_time": -1,
		},
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

	// find organizations data
	orgsPipeline := bson.M{
		"$lookup": bson.M{
			"from":         "organizations",
			"localField":   "organizers._id",
			"foreignField": "_id",
			"as":           "organizations",
		},
	}

	// remove unecessary data
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
		filterPipeline,
		sportsPipeline,
		timePipeline,
		sortPipeline,
		posterPipeline,
		addPosterPipeline,
		participantsPipeline,
		addParticipantsPipeline,
		fieldPipeline,
		addFieldPipeline,
		clubsPipeline,
		orgsPipeline,
		visibilityPipeline,
		limitPipeline,
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
			database.Logger.Error("failed to decode event", err)
		}
		response = append(response, event)
	}

	return &response, nil
}

/*
Find event data by field id
*/
func FindEventsByField(id primitive.ObjectID, limit int, database *database.Database) (*[]models.Event, error) {

	ctx := context.Background()

	// filter out all docs by our ID
	idPipeline := bson.M{
		"$match": bson.M{
			"venue._id": id,
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
