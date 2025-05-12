package aggregations

import (
	"context"
	"olympsis-server/database"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func AggregateUser(uuid *string, database *database.Database) (*models.UserData, error) {

	ctx := context.Background()

	// find user auth object
	match := bson.M{
		"$match": bson.M{
			"uuid": *uuid,
		},
	}

	// grab user metadata
	lookup := bson.M{
		"$lookup": bson.M{
			"from":         "users",
			"localField":   "uuid",
			"foreignField": "uuid",
			"as":           "metadata",
		},
	}

	// move result into document
	metadata := bson.M{
		"$addFields": bson.M{
			"metadata": bson.M{
				"$arrayElemAt": bson.A{
					"$metadata",
					0,
				},
			},
		},
	}

	// move data from result object into document
	order := bson.M{
		"$set": bson.M{
			"username":                "$metadata.username",
			"bio":                     "$metadata.bio",
			"sports":                  "$metadata.sports",
			"image_url":               "$metadata.image_url",
			"visibility":              "$metadata.visibility",
			"clubs":                   "$metadata.clubs",
			"organizations":           "$metadata.organizations",
			"accepted_eula":           "$metadata.accepted_eula",
			"has_onboarded":           "$metadata.has_onboarded",
			"blocked_users":           "$metadata.blocked_users",
			"hometown":                "$metadata.hometown",
			"last_location":           "$metadata.last_location",
			"notification_devices":    "$metadata.notification_devices",
			"notification_preference": "$metadata.notification_preference",
		},
	}

	// cleanup unnecessary properties
	cleanup := bson.M{
		"$project": bson.M{
			"_id":        0,
			"metadata":   0,
			"created_at": 0,
		},
	}

	pipeline := bson.A{
		match,
		lookup,
		metadata,
		order,
		cleanup,
	}

	cur, err := database.AuthCol.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}

	var user models.UserData
	if cur.Next(ctx) {
		err = cur.Decode(&user)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, mongo.ErrNoDocuments
	}

	return &user, nil
}
