package service

import (
	"context"
	"olympsis-server/database"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func generateNewPostNotification(id string, title string, username string) models.PushNotification {
	return models.PushNotification{
		Title:    title,
		Body:     username + " created a new post!",
		Type:     "push",
		Category: "club",
		Data: map[string]interface{}{
			"type": "new_post",
			"id":   id,
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
			"type": "new_post",
			"id":   id,
		},
	}
}

func findClubMembers(ctx *context.Context, id *primitive.ObjectID, db *database.Database) (*[]models.MemberDao, error) {
	var members []models.MemberDao
	cur, err := db.ClubMembersCollection.Find(*ctx, bson.M{"club_id": id})
	if err != nil {
		return nil, err
	}

	err = cur.All(*ctx, members)
	if err != nil {
		return nil, err
	}

	return &members, nil
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
