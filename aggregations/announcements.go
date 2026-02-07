package aggregations

import (
	"context"
	"olympsis-server/database"
	"olympsis-server/utils"
	"time"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Aggregate a single announcement with creator details
func AggregateAnnouncement(ctx context.Context, id bson.ObjectID, db *database.Database) (*models.Announcement, error) {
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"_id": id,
			},
		},
		{
			"$lookup": bson.M{
				"from":         "users",
				"localField":   "creator",
				"foreignField": "uuid",
				"as":           "creator_user",
			},
		},
		{
			"$unwind": bson.M{
				"path":                       "$creator_user",
				"preserveNullAndEmptyArrays": true,
			},
		},
		{
			"$project": bson.M{
				"_id":            1,
				"title":          1,
				"subtitle":       1,
				"text_emphasis":  1,
				"title_style":    1,
				"subtitle_style": 1,
				"media_url":      1,
				"media_type":     1,
				"action_button":  1,
				"position":       1,
				"scope":          1,
				"location":       1,
				"status":         1,
				"active_date":    1,
				"expiry_date":    1,
				"created_at":     1,
				"updated_at":     1,
				"creator": bson.M{
					"uuid":      "$creator",
					"username":  "$creator_user.username",
					"image_url": "$creator_user.image_url",
				},
			},
		},
	}

	cursor, err := db.AnnouncementCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var announcements []models.Announcement
	if err = cursor.All(ctx, &announcements); err != nil {
		return nil, err
	}

	if len(announcements) == 0 {
		return nil, mongo.ErrNoDocuments
	}

	// Apply text emphasis defaults to ensure consistent styling
	announcement := &announcements[0]
	utils.ApplyTextEmphasisDefaults(announcement)

	return announcement, nil
}

// Aggregate multiple announcements with creator details
func AggregateAnnouncements(ctx context.Context, filter bson.M, opts *options.AggregateOptionsBuilder, db *database.Database) ([]models.Announcement, error) {
	pipeline := []bson.M{
		{
			"$match": filter,
		},
		{
			"$lookup": bson.M{
				"from":         "users",
				"localField":   "creator",
				"foreignField": "uuid",
				"as":           "creator_user",
			},
		},
		{
			"$unwind": bson.M{
				"path":                       "$creator_user",
				"preserveNullAndEmptyArrays": true,
			},
		},
		{
			"$project": bson.M{
				"_id":            1,
				"title":          1,
				"subtitle":       1,
				"text_emphasis":  1,
				"title_style":    1,
				"subtitle_style": 1,
				"media_url":      1,
				"media_type":     1,
				"action_button":  1,
				"position":       1,
				"scope":          1,
				"location":       1,
				"status":         1,
				"active_date":    1,
				"expiry_date":    1,
				"created_at":     1,
				"updated_at":     1,
				"creator": bson.M{
					"uuid":      "$creator",
					"username":  "$creator_user.username",
					"image_url": "$creator_user.image_url",
				},
			},
		},
		{
			"$sort": bson.M{
				"created_at": -1, // Newest first
			},
		},
	}

	cursor, err := db.AnnouncementCollection.Aggregate(ctx, pipeline, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var announcements []models.Announcement
	if err = cursor.All(ctx, &announcements); err != nil {
		return nil, err
	}

	// Apply text emphasis defaults to all announcements
	for i := range announcements {
		utils.ApplyTextEmphasisDefaults(&announcements[i])
	}

	return announcements, nil
}

// Get active announcements with location filtering and aggregation
func GetActiveAnnouncements(ctx context.Context, location *models.Location, limit int, db *database.Database) ([]models.Announcement, error) {
	currentTime := time.Now().Unix()

	// Base filter for active announcements
	baseFilter := bson.M{
		"status":      models.StatusActive,
		"active_date": bson.M{"$lte": currentTime},
		"expiry_date": bson.M{"$gt": currentTime},
	}

	// Combined filters for different scopes
	filter := bson.M{
		"$or": []bson.M{
			// Global announcements
			{
				"scope": models.ScopeGlobal,
			},
			// Application-wide announcements
			{
				"scope": models.ScopeGlobal,
			},
		},
	}

	// Add location-specific filter if location is provided
	if location != nil {
		localFilter := bson.M{
			"scope":            models.ScopeLocal,
			"location.country": location.Country,
		}

		// Optional: refine by more specific location fields
		if location.AdminArea != "" {
			localFilter["location.admin_area"] = location.AdminArea
		}
		if location.SubAdminArea != "" {
			localFilter["location.sub_admin_area"] = location.SubAdminArea
		}

		// Add local filter to the $or array
		filter["$or"] = append(filter["$or"].([]bson.M), localFilter)
	}

	// Combine with base filter using $and
	combinedFilter := bson.M{
		"$and": []bson.M{baseFilter, filter},
	}

	// Set up aggregation options (SetMaxTime removed in v2; use context deadlines instead)
	opts := options.Aggregate()

	// Use the aggregation pipeline to get full announcements with creator info
	announcements, err := AggregateAnnouncements(ctx, combinedFilter, opts, db)
	if err != nil {
		return nil, err
	}

	// Apply limit after aggregation
	if len(announcements) > limit {
		announcements = announcements[:limit]
	}

	return announcements, nil
}
