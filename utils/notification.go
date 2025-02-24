package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/olympsis/models"
	"github.com/sirupsen/logrus"
)

type NotificationInterface struct {
	ServiceURL string
	Status     string
	Client     http.Client
	Logger     *logrus.Logger
}

func NewNotificationInterface(u string, logger *logrus.Logger) *NotificationInterface {
	return &NotificationInterface{
		ServiceURL: u,
		Status:     "good",
		Client:     http.Client{},
		Logger:     logger,
	}
}

// Send a notification - handles both token and topic notifications
func (n *NotificationInterface) SendNotification(authToken string, notification models.NotificationPushRequest) error {
	// convert notification to json
	data, err := json.Marshal(notification)
	if err != nil {
		n.Logger.Errorf("Failed to marshal notification: %v", err)
		return err
	}

	// craft http request
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/notifications", n.ServiceURL), bytes.NewBuffer(data))
	if err != nil {
		n.Logger.Errorf("Failed to create request: %v", err)
		return err
	}

	// Set the request headers
	req.Header.Set("Content-Type", "application/json")
	if authToken != "" {
		req.Header.Set("Authorization", authToken)
	}

	// Send the request asynchronously
	go func() {
		resp, err := n.Client.Do(req)
		if err != nil {
			n.Logger.Errorf("Failed to send notification: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			n.Logger.Errorf("Notification service returned non-success status code: %d", resp.StatusCode)
		}
	}()

	return nil
}

// This is the single, unified function for sending notifications
// The backward compatibility functions have been removed

// Create a new notification topic
func (n *NotificationInterface) CreateNotificationTopic(name string) error {
	topic := models.NotificationTopicDao{
		Name: &name,
	}
	return n.CreateTopic("", topic)
}

// Delete a notification topic
func (n *NotificationInterface) DeleteNotificationTopic(name string) error {
	return n.DeleteTopic("", name)
}

// Add a user to a topic
func (n *NotificationInterface) AddTokenToTopic(topic string, user string) error {
	updateReq := models.NotificationTopicUpdateRequest{
		Action: "add",
		Users:  []string{user},
	}
	return n.ModifyTopic("", topic, updateReq)
}

// Remove a user from a topic
func (n *NotificationInterface) RemoveTokenFromTopic(topic string, user string) error {
	updateReq := models.NotificationTopicUpdateRequest{
		Action: "remove",
		Users:  []string{user},
	}
	return n.ModifyTopic("", topic, updateReq)
}

// Create a notification topic
func (n *NotificationInterface) CreateTopic(authToken string, topic models.NotificationTopicDao) error {
	// convert topic to json
	data, err := json.Marshal(topic)
	if err != nil {
		n.Logger.Errorf("Failed to marshal topic: %v", err)
		return err
	}

	// craft http request
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/notifications/topics", n.ServiceURL), bytes.NewBuffer(data))
	if err != nil {
		n.Logger.Errorf("Failed to create request: %v", err)
		return err
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if authToken != "" {
		req.Header.Set("Authorization", authToken)
	}

	// Send the request
	resp, err := n.Client.Do(req)
	if err != nil {
		n.Logger.Errorf("Failed to create topic: %v", err)
		return err
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		n.Logger.Errorf("Notification service returned non-200/201 status code: %d", resp.StatusCode)
		return fmt.Errorf("notification service returned error status code: %d", resp.StatusCode)
	}

	return nil
}

// Modify a notification topic (add/remove users)
func (n *NotificationInterface) ModifyTopic(authToken string, topicName string, update models.NotificationTopicUpdateRequest) error {
	// convert update to json
	data, err := json.Marshal(update)
	if err != nil {
		n.Logger.Errorf("Failed to marshal topic update: %v", err)
		return err
	}

	// craft http request
	var endpoint string
	if update.Action == "add" {
		endpoint = fmt.Sprintf("%s/v1/notifications/topics/%s/add", n.ServiceURL, topicName)
	} else {
		endpoint = fmt.Sprintf("%s/v1/notifications/topics/%s/remove", n.ServiceURL, topicName)
	}

	req, err := http.NewRequest("PUT", endpoint, bytes.NewBuffer(data))
	if err != nil {
		n.Logger.Errorf("Failed to create request: %v", err)
		return err
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if authToken != "" {
		req.Header.Set("Authorization", authToken)
	}

	// Send the request
	resp, err := n.Client.Do(req)
	if err != nil {
		n.Logger.Errorf("Failed to modify topic: %v", err)
		return err
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		n.Logger.Errorf("Notification service returned non-200 status code: %d", resp.StatusCode)
		return fmt.Errorf("notification service returned error status code: %d", resp.StatusCode)
	}

	return nil
}

// Delete a notification topic
func (n *NotificationInterface) DeleteTopic(authToken string, name string) error {
	// craft http request
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/v1/notifications/topics/%s", n.ServiceURL, name), nil)
	if err != nil {
		n.Logger.Errorf("Failed to create request: %v", err)
		return err
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if authToken != "" {
		req.Header.Set("Authorization", authToken)
	}

	// Send the request
	resp, err := n.Client.Do(req)
	if err != nil {
		n.Logger.Errorf("Failed to delete topic: %v", err)
		return err
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		n.Logger.Errorf("Notification service returned non-200 status code: %d", resp.StatusCode)
		return fmt.Errorf("notification service returned error status code: %d", resp.StatusCode)
	}

	return nil
}
