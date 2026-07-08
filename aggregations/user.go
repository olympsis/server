package aggregations

import (
	"context"
	"olympsis-server/database"
	"regexp"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// AggregateUser resolves a user's full profile (auth + metadata) for the given
// uuid. The caller passes its request context so the aggregation honors client
// cancellation and request deadlines instead of running detached on
// context.Background().
func AggregateUser(ctx context.Context, uuid *string, database *database.Database) (*models.UserData, error) {

	// find user auth object
	match := bson.M{
		"$match": bson.M{
			"user_id": *uuid,
		},
	}

	// grab user metadata
	lookup := bson.M{
		"$lookup": bson.M{
			"from":         "users",
			"localField":   "user_id",
			"foreignField": "user_id",
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
			"gender":                  "$metadata.gender",
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

	cur, err := database.AuthCollection.Aggregate(ctx, pipeline)
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

// AggregateUsersByUsername searches the users collection for usernames matching
// the given query (case-insensitive substring) and returns up to `limit` trimmed
// user snippets. First/last name live on the auth collection rather than the
// users collection, so we $lookup into "auth" and merge those two fields onto
// each match. Driving the whole search from a single pipeline (instead of a
// per-match auth lookup) keeps it to one round-trip to Mongo.
func AggregateUsersByUsername(ctx context.Context, username string, limit int64, database *database.Database) ([]models.UserSnippet, error) {

	// case-insensitive substring match on username. QuoteMeta escapes any regex
	// metacharacters in the raw client input so it's matched as a literal string.
	match := bson.M{
		"$match": bson.M{
			"username": bson.Regex{Pattern: regexp.QuoteMeta(username), Options: "i"},
		},
	}

	// deterministic ordering so the "top k" slice is stable across calls
	sort := bson.M{
		"$sort": bson.M{"username": 1},
	}

	// cap the result set at the requested limit (top k)
	limitStage := bson.M{
		"$limit": limit,
	}

	// grab first/last name from the auth collection (keyed on user_id)
	lookup := bson.M{
		"$lookup": bson.M{
			"from":         "auth",
			"localField":   "user_id",
			"foreignField": "user_id",
			"as":           "_auth",
		},
	}

	// merge the looked-up name fields onto the top-level document
	addNames := bson.M{
		"$addFields": bson.M{
			"first_name": bson.M{"$arrayElemAt": bson.A{"$_auth.first_name", 0}},
			"last_name":  bson.M{"$arrayElemAt": bson.A{"$_auth.last_name", 0}},
		},
	}

	// keep only the snippet fields
	project := bson.M{
		"$project": bson.M{
			"_id":        0,
			"user_id":    1,
			"username":   1,
			"image_url":  1,
			"first_name": 1,
			"last_name":  1,
		},
	}

	pipeline := bson.A{match, sort, limitStage, lookup, addNames, project}

	cur, err := database.UserCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	// non-nil default so an empty result encodes as [] rather than null
	users := []models.UserSnippet{}
	if err := cur.All(ctx, &users); err != nil {
		return nil, err
	}

	return users, nil
}
