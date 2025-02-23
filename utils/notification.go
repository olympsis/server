package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/olympsis/models"
)

// Contact the notification service and send a notification to a token belonging to a device
func SendNotificationToToken(token string, notification *models.Notification) error {

	// convert notification to json
	data, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	// craft http request
	url := os.Getenv("NOTIF_URL")
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/notifications/%s", url, token), bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	// Set the request headers
	req.Header.Set("Content-Type", "application/json")

	// Create a new HTTP client and send the request
	client := &http.Client{}
	go func() {
		_, err := client.Do(req)
		if err != nil {
			fmt.Sprintln("Notification error: " + err.Error())
		}
	}()

	return nil
}

// Contact the notification service and send a notification to a topic
func SendNotificationToTopic(notification *models.Notification) error {

	// convert notification to json
	data, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	// craft http request
	url := os.Getenv("NOTIF_URL")
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/notifications", url), bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	// Set the request headers
	req.Header.Set("Content-Type", "application/json")

	// Create a new HTTP client and send the request
	client := &http.Client{}
	go func() {
		_, err := client.Do(req)
		if err != nil {
			fmt.Sprintln("Notification error: " + err.Error())
		}
	}()

	return nil
}

// Contact the notification service and create a new topic
func CreateNotificationTopic(name string) error {
	// data, err := json.Marshal(models.Topic{Name: name})
	// if err != nil {
	// 	return err
	// }

	// // craft http request
	// url := os.Getenv("NOTIF_URL")
	// req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/notifications/topics", url), bytes.NewBuffer(data))
	// if err != nil {
	// 	return err
	// }

	// // Set the request headers
	// req.Header.Set("Content-Type", "application/json")

	// // Create a new HTTP client and send the request
	// client := &http.Client{}
	// resp, err := client.Do(req)
	// if err != nil {
	// 	return err
	// }

	// if resp.StatusCode != 200 {
	// 	return errors.New("status code not ok")
	// }

	return nil
}

// Contact the notification service and delete a topic
func DeleteNotificationTopic(name string) error {
	// // craft http request
	// url := os.Getenv("NOTIF_URL")
	// req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/v1/notifications/topics/%s", url, name), &bytes.Buffer{})
	// if err != nil {
	// 	return err
	// }

	// // Set the request headers
	// req.Header.Set("Content-Type", "application/json")

	// // Create a new HTTP client and send the request
	// client := &http.Client{}
	// resp, err := client.Do(req)
	// if err != nil {
	// 	return err
	// }

	// if resp.StatusCode != 200 {
	// 	return errors.New("status code not ok")
	// }

	return nil
}

// Contact the notification service and add a user to a topic
func AddTokenToTopic(topic string, user string) error {
	// data, err := json.Marshal(models.ModifyTopic{User: user})
	// if err != nil {
	// 	return err
	// }

	// // craft http request
	// url := os.Getenv("NOTIF_URL")
	// req, err := http.NewRequest("PUT", fmt.Sprintf("%s/v1/notifications/topics/%s/add", url, topic), bytes.NewBuffer(data))
	// if err != nil {
	// 	return err
	// }

	// // Set the request headers
	// req.Header.Set("Content-Type", "application/json")

	// // Create a new HTTP client and send the request
	// client := &http.Client{}
	// resp, err := client.Do(req)
	// if err != nil {
	// 	return err
	// }

	// if resp.StatusCode != 200 {
	// 	return errors.New("status code not ok")
	// }

	return nil
}

func GetUserNotifications(token string, uuid string) *models.NotificationListResponse {
	return nil
}

func SendNotification(token string, note models.NotificationPushRequest) {
	return
}
func _CreateNotificationTopic(token string, topic models.NotificationTopicDao) {
	return
}
func _ModifyNotificationTopic(token string, topic models.NotificationTopicUpdateRequest) {
	return
}
func _DeleteNotificationTopic(token string, name string) bool {
	return true
}
