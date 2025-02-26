package service

import (
	"fmt"

	"github.com/olympsis/models"
)

func generateNewReportNotification(id string, name string, repID string) models.PushNotification {
	return models.PushNotification{
		Title:    "New Report!",
		Body:     fmt.Sprintf("A member of %s created a report.", name),
		Type:     "push",
		Category: "groups",
		Data: map[string]interface{}{
			"type":      "new_report",
			"id":        id,
			"report_id": repID,
		},
	}
}

func generateNewApplicationNotification(id string, appID string) models.PushNotification {
	return models.PushNotification{
		Title:    "New Application",
		Body:     "You have a new club application!",
		Type:     "push",
		Category: "groups",
		Data: map[string]interface{}{
			"type":           "new_application",
			"id":             id,
			"application_id": appID,
		},
	}
}

func generateUpdateApplicationNotification(id string, name string, appID string, status string) models.PushNotification {
	return models.PushNotification{
		Title:    "Application Status",
		Body:     fmt.Sprintf("%s accepted your application!", name),
		Type:     "push",
		Category: "groups",
		Data: map[string]interface{}{
			"type":           "application_update",
			"id":             id,
			"application_id": appID,
		},
	}
}
