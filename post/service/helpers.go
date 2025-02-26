package service

import "github.com/olympsis/models"

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
