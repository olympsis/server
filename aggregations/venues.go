package aggregations

import (
	"context"
	"olympsis-server/database"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// AggregateVenue gets a single venue by ID with all related data
// (transit lines and venue units looked up and embedded).
func AggregateVenue(id bson.ObjectID, database *database.Database) (*models.Venue, error) {
	ctx := context.Background()

	idPipeline := bson.M{
		"$match": bson.M{
			"_id": id,
		},
	}

	corePipeline := BuildVenueCorePipeline()

	completePipeline := append(bson.A{idPipeline}, corePipeline...)

	cur, err := database.VenuesCollection.Aggregate(ctx, completePipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var venue models.Venue
	if cur.Next(ctx) {
		err = cur.Decode(&venue)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, mongo.ErrNoDocuments
	}

	return &venue, nil
}

// AggregateVenues fetches multiple venues using a pre-built query pipeline
// (from generateVenuesQuery) and appends the core lookups + pagination.
func AggregateVenues(
	queryPipeline bson.A,
	limit int,
	skip int,
	database *database.Database,
) (*[]models.Venue, error) {
	ctx := context.Background()

	corePipeline := BuildVenueCorePipeline()

	// Start with the query/filter stages (geo, sports, transit from generateVenuesQuery)
	pipeline := make(bson.A, 0, len(queryPipeline)+len(corePipeline)+2)
	pipeline = append(pipeline, queryPipeline...)

	// Append the core lookup stages
	pipeline = append(pipeline, corePipeline...)

	// Pagination
	pipeline = append(pipeline,
		bson.M{"$skip": skip},
		bson.M{"$limit": limit},
	)

	cur, err := database.VenuesCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	response := make([]models.Venue, 0, limit)
	err = cur.All(ctx, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// BuildVenueCorePipeline returns the common aggregation pipeline stages shared
// by both single-venue and multi-venue queries. It resolves the ObjectID
// references on a venue into their full documents:
//
//   - transit_lines  → looked up from the transitLines collection
//   - units          → looked up from the venueUnits collection
func BuildVenueCorePipeline() bson.A {
	// Resolve transit_lines ObjectID array into full TransitLine documents
	transitLookup := bson.M{
		"$lookup": bson.M{
			"from":         "transitLines",
			"localField":   "transit_lines",
			"foreignField": "_id",
			"as":           "transit_lines",
		},
	}

	// Resolve units ObjectID array into full VenueUnit documents
	unitsLookup := bson.M{
		"$lookup": bson.M{
			"from":         "venueUnits",
			"localField":   "units",
			"foreignField": "_id",
			"as":           "units",
		},
	}

	return bson.A{
		transitLookup,
		unitsLookup,
	}
}
