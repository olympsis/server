package service

import (
	"context"
	"olympsis-server/database"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func FindPost(id primitive.ObjectID, database *database.Database) (*models.Post, error) {

	idPipeline := bson.M{
		"$match": bson.M{
			"_id": id,
		},
	}

	posterPipeline := bson.D{
		{Key: "$lookup",
			Value: bson.D{
				{Key: "from", Value: "users"},
				{Key: "localField", Value: "poster"},
				{Key: "foreignField", Value: "uuid"},
				{Key: "as", Value: "metadata"},
			},
		},
	}

	addPipeline := bson.D{
		{Key: "$addFields",
			Value: bson.D{
				{Key: "poster",
					Value: bson.D{
						{Key: "$arrayElemAt",
							Value: bson.A{
								"$metadata",
								0,
							},
						},
					},
				},
			},
		},
	}

	posterCleanPipeline := bson.D{
		{Key: "$project",
			Value: bson.D{
				{Key: "metadata", Value: 0},
				{Key: "poster._id", Value: 0},
				{Key: "poster.clubs", Value: 0},
				{Key: "poster.sports", Value: 0},
				{Key: "poster.visibility", Value: 0},
				{Key: "poster.device_token", Value: 0},
				{Key: "poster.organizations", Value: 0},
			},
		},
	}

	commentsPipeline := bson.D{
		{Key: "$lookup",
			Value: bson.D{
				{Key: "from", Value: "users"},
				{Key: "localField", Value: "comments.uuid"},
				{Key: "foreignField", Value: "uuid"},
				{Key: "as", Value: "commentUsers"},
			},
		},
	}

	commentsAddPipeline := bson.D{
		{Key: "$addFields",
			Value: bson.D{
				{Key: "comments",
					Value: bson.D{
						{Key: "$map",
							Value: bson.D{
								{Key: "input", Value: "$comments"},
								{Key: "as", Value: "comment"},
								{Key: "in",
									Value: bson.D{
										{Key: "$mergeObjects",
											Value: bson.A{
												"$$comment",
												bson.D{
													{Key: "user",
														Value: bson.D{
															{Key: "$arrayElemAt",
																Value: bson.A{
																	bson.D{
																		{Key: "$filter",
																			Value: bson.D{
																				{Key: "input", Value: "$commentUsers"},
																				{Key: "as", Value: "cu"},
																				{Key: "cond",
																					Value: bson.D{
																						{Key: "$eq",
																							Value: bson.A{
																								"$$cu.uuid",
																								"$$comment.uuid",
																							},
																						},
																					},
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
								},
							},
						},
					},
				},
			},
		},
	}

	commentsCleanPipeline := bson.M{
		"$project": bson.M{
			"comments.uuid":               0,
			"comments.user._id":           0,
			"comments.user.clubs":         0,
			"comments.user.sports":        0,
			"comments.user.visibility":    0,
			"comments.user.device_token":  0,
			"comments.user.organizations": 0,
			"commentUsers":                0,
		},
	}

	pipeline := bson.A{
		idPipeline,
		posterPipeline,
		addPipeline,
		posterCleanPipeline,
		commentsPipeline,
		commentsAddPipeline,
		commentsCleanPipeline,
	}

	cur, err := database.PostCol.Aggregate(context.TODO(), pipeline)
	if err != nil {
		return nil, err
	}

	var post models.Post
	if cur.Next(context.Background()) {
		err = cur.Decode(&post)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, mongo.ErrNoDocuments
	}

	return &post, nil
}

func FindPosts(groupIDs []primitive.ObjectID, database *database.Database, limit int) (*[]models.Post, error) {

	groupsFilter := bson.A{}
	for i := range groupIDs {
		groupsFilter = append(groupsFilter, bson.M{"group_id": bson.M{"$eq": groupIDs[i]}})
	}

	groupPipeline := bson.M{
		"$match": bson.M{
			"$or": groupsFilter,
		},
	}

	posterPipeline := bson.D{
		{Key: "$lookup",
			Value: bson.D{
				{Key: "from", Value: "users"},
				{Key: "localField", Value: "poster"},
				{Key: "foreignField", Value: "uuid"},
				{Key: "as", Value: "metadata"},
			},
		},
	}

	addPipeline := bson.D{
		{Key: "$addFields",
			Value: bson.D{
				{Key: "poster",
					Value: bson.D{
						{Key: "$arrayElemAt",
							Value: bson.A{
								"$metadata",
								0,
							},
						},
					},
				},
			},
		},
	}

	posterCleanPipeline := bson.M{
		"$project": bson.M{
			"metadata":             0,
			"poster._id":           0,
			"poster.clubs":         0,
			"poster.sports":        0,
			"poster.visibility":    0,
			"poster.device_token":  0,
			"poster.organizations": 0,
		},
	}

	commentsPipeline := bson.D{
		{Key: "$lookup",
			Value: bson.D{
				{Key: "from", Value: "users"},
				{Key: "localField", Value: "comments.uuid"},
				{Key: "foreignField", Value: "uuid"},
				{Key: "as", Value: "commentUsers"},
			},
		},
	}

	commentsAddPipeline := bson.D{
		{Key: "$addFields",
			Value: bson.D{
				{Key: "comments",
					Value: bson.D{
						{Key: "$map",
							Value: bson.D{
								{Key: "input", Value: "$comments"},
								{Key: "as", Value: "comment"},
								{Key: "in",
									Value: bson.D{
										{Key: "$mergeObjects",
											Value: bson.A{
												"$$comment",
												bson.D{
													{Key: "user",
														Value: bson.D{
															{Key: "$arrayElemAt",
																Value: bson.A{
																	bson.D{
																		{Key: "$filter",
																			Value: bson.D{
																				{Key: "input", Value: "$commentUsers"},
																				{Key: "as", Value: "cu"},
																				{Key: "cond",
																					Value: bson.D{
																						{Key: "$eq",
																							Value: bson.A{
																								"$$cu.uuid",
																								"$$comment.uuid",
																							},
																						},
																					},
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
								},
							},
						},
					},
				},
			},
		},
	}

	commentsCleanPipeline := bson.M{
		"$project": bson.M{
			"comments.uuid":               0,
			"comments.user._id":           0,
			"comments.user.clubs":         0,
			"comments.user.sports":        0,
			"comments.user.visibility":    0,
			"comments.user.device_token":  0,
			"comments.user.organizations": 0,
			"commentUsers":                0,
		},
	}

	sortPipeline := bson.M{
		"$sort": bson.M{
			"created_at": -1,
		},
	}

	limitPipeline := bson.M{
		"$limit": limit,
	}

	pipeline := bson.A{
		groupPipeline,
		sortPipeline,
		limitPipeline,
		posterPipeline,
		addPipeline,
		posterCleanPipeline,
		commentsPipeline,
		commentsAddPipeline,
		commentsCleanPipeline,
	}

	cur, err := database.PostCol.Aggregate(context.TODO(), pipeline)

	if err != nil {
		return nil, err
	}

	var response []models.Post
	for cur.Next(context.TODO()) {
		var post models.Post
		err := cur.Decode(&post)
		if err != nil {
			database.Logger.Error(err)
		}

		response = append(response, post)
	}

	return &response, nil
}
