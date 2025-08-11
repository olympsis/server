package service

import (
	"context"
	"fmt"
	"net/http"
	"olympsis-server/database"
	"strconv"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Query parameters structure for cleaner handling
type PostQueryParams struct {
	GroupID  string
	ParentID *string
	Skip     int
	Limit    int
}

func parsePostQueryParams(r *http.Request) (*PostQueryParams, error) {
	query := r.URL.Query()
	params := &PostQueryParams{}

	groupID := query.Get("groupID")
	parentID := query.Get("parentID")

	if groupID == "" {
		return nil, fmt.Errorf("Group ID required in query")
	}
	params.GroupID = groupID

	if parentID != "" {
		params.ParentID = &parentID
	}

	// Parse skip with default
	skipStr := query.Get("skip")
	if skipStr != "" {
		skip, err := strconv.ParseInt(skipStr, 10, 32)
		if err != nil {
			params.Skip = 0
		} else {
			params.Skip = int(skip)
		}
	} else {
		params.Skip = 0
	}

	// Parse limit with default
	limitStr := query.Get("limit")
	if limitStr != "" {
		limit, err := strconv.ParseInt(limitStr, 10, 32)
		if err != nil {
			params.Limit = 50
		} else {
			params.Limit = int(limit)
		}
	} else {
		params.Limit = 50
	}

	return params, nil
}

func generateNewPostNotification(id string, title string, username string) models.PushNotification {
	return models.PushNotification{
		Title:    fmt.Sprintf("[%s] New Post", title),
		Body:     username + " created a new post!",
		Type:     "push",
		Category: "club",
		Data: map[string]interface{}{
			"type":    models.ClubNewPostType,
			"club_id": id,
		},
	}
}

func generateNewAnnouncementNotification(id string, title string) models.PushNotification {
	return models.PushNotification{
		Title:    title,
		Body:     "New Announcement!",
		Type:     "push",
		Category: "organization",
		Data: map[string]interface{}{
			"type":   models.OrganizationNewAnnouncementType,
			"org_id": id,
		},
	}
}

func findOrganizationMembers(ctx *context.Context, id *primitive.ObjectID, db *database.Database) (*[]models.MemberDao, error) {
	var members []models.MemberDao
	cur, err := db.ClubMembersCollection.Find(*ctx, bson.M{"org_id": id})
	if err != nil {
		return nil, err
	}

	err = cur.All(*ctx, members)
	if err != nil {
		return nil, err
	}

	return &members, nil
}
