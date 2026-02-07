package database

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Checks if the collection exists
func (d *Database) collectionExists(db *mongo.Database, name string) bool {
	collections, err := db.ListCollectionNames(context.Background(), bson.M{"name": name})
	if err != nil {
		// If there's an error, assume the collection doesn't exist
		return false
	}
	return len(collections) > 0
}

// Creates a new collection
func (d *Database) createCollection(db *mongo.Database, name string) error {
	err := db.CreateCollection(context.Background(), name)
	if err != nil {
		return fmt.Errorf("could not create collection %s: %v", name, err)
	}
	return nil
}

// Creates a new time-series collection
func (d *Database) createTimeSeriesCollection(db *mongo.Database, name string, timeField string) error {
	opts := options.CreateCollection().SetTimeSeriesOptions(
		options.TimeSeries().
			SetTimeField(timeField).
			SetGranularity("seconds"),
	)
	err := db.CreateCollection(context.Background(), name, opts)
	if err != nil {
		return fmt.Errorf("could not create time series collection %s: %v", name, err)
	}
	return nil
}

// getIndexName extracts the index name from an IndexOptionsBuilder.
// In v2, IndexModel.Options is a builder with setter functions, so we
// resolve them into an IndexOptions struct to read the Name field.
func getIndexName(builder *options.IndexOptionsBuilder) *string {
	if builder == nil {
		return nil
	}
	resolved := &options.IndexOptions{}
	for _, fn := range builder.Opts {
		if err := fn(resolved); err != nil {
			return nil
		}
	}
	return resolved.Name
}

// Safely creates the indexes for a collection
func createIndexes(collection *mongo.Collection, indexes []mongo.IndexModel, collectionName string) error {
	// First, list existing indexes
	cursor, err := collection.Indexes().List(context.Background())
	if err != nil {
		return fmt.Errorf("could not list existing indexes for %s: %v", collectionName, err)
	}
	defer cursor.Close(context.Background())

	// Extract existing index names
	existingIndexes := make(map[string]bool)
	var indexDoc bson.M
	for cursor.Next(context.Background()) {
		if err := cursor.Decode(&indexDoc); err != nil {
			return fmt.Errorf("could not decode index document: %v", err)
		}
		if name, exists := indexDoc["name"].(string); exists {
			existingIndexes[name] = true
		}
	}

	if err := cursor.Err(); err != nil {
		return fmt.Errorf("error during index cursor iteration: %v", err)
	}

	// Filter out indexes that already exist
	var newIndexes []mongo.IndexModel
	for _, idx := range indexes {
		name := getIndexName(idx.Options)
		if name == nil {
			// No name specified, keep the index
			newIndexes = append(newIndexes, idx)
			continue
		}

		if existingIndexes[*name] {
			// Skip this index as it already exists
			continue
		}
		newIndexes = append(newIndexes, idx)
	}

	// Create only new indexes
	if len(newIndexes) > 0 {
		_, err := collection.Indexes().CreateMany(context.Background(), newIndexes)
		if err != nil {
			return fmt.Errorf("could not create indexes for %s: %v", collectionName, err)
		}
	}

	return nil
}
