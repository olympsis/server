package interfaces

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/olympsis/models"
)

type NotificationInterface struct {
	ServiceHost string
	Status      int
	Client      http.Client
}

func NewNotificationInterface() (*NotificationInterface, error) {
	url := os.Getenv("NOTIF_URL")
	if url == "" {
		return nil, errors.New("no notification service url provided")
	}
	return &NotificationInterface{
		ServiceHost: url,
		Status:      1,
		Client:      http.Client{},
	}, nil
}

// Check the connection status to the service
func (n *NotificationInterface) CheckStatus() int { return 1 }

// Get notification topic from notification service using it's name
func (n *NotificationInterface) GetNotificationTopic(topic string) (*models.NotificationTopic, error) {
	// Create new request and set headers
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/v1/notifications/topics/%s", n.ServiceHost, topic), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	// Send the newly created request
	resp, err := n.Client.Do(req)
	if err != nil {
		return nil, err
	}

	// Check http status
	if resp.StatusCode != 200 {
		return nil, errors.New("status code not ok")
	}

	// Decode response
	var value models.NotificationTopic
	err = json.NewDecoder(resp.Body).Decode(&value)
	if err != nil {
		return nil, errors.New("failed to decode response")
	}

	return &value, nil
}

// Create a new notification topic
func (n *NotificationInterface) CreateNotificationTopic(topic string, topicType string, users []string) error {
	isActive := true
	timeStamp := time.Now().Unix()
	newTopic := models.NotificationTopicDao{
		Name:      &topic,
		Type:      &topicType,
		Users:     &users,
		IsActive:  &isActive,
		CreatedAt: &timeStamp,
	}

	// convert notification to json
	data, err := json.Marshal(newTopic)
	if err != nil {
		return err
	}

	// Create new request and set headers
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/notifications/topics", n.ServiceHost), bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	// Send the newly created request
	resp, err := n.Client.Do(req)
	if err != nil {
		return err
	}

	// Check http status
	if resp.StatusCode != 201 {
		return errors.New("status code not created")
	}

	return nil
}

// Update a notification topic
func (n *NotificationInterface) UpdateNotificationTopic(topic string, update *models.NotificationTopicUpdateRequest) error {

	// convert notification to json
	data, err := json.Marshal(update)
	if err != nil {
		return err
	}

	// Create new request and set headers
	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/v1/notifications/topics/%s", n.ServiceHost, topic), bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	// Send the newly created request
	resp, err := n.Client.Do(req)
	if err != nil {
		return err
	}

	// Check http status
	if resp.StatusCode != 200 {
		return errors.New("status code not ok")
	}

	return nil
}

// Delete a notification topic
func (n *NotificationInterface) DeleteNotificationTopic(topic string) error {
	// Create new request and set headers
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/v1/notifications/topics/%s", n.ServiceHost, topic), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	// Send the newly created request
	resp, err := n.Client.Do(req)
	if err != nil {
		return err
	}

	// Check http status
	if resp.StatusCode != 200 {
		return errors.New("status code not ok")
	}

	return nil
}

// Send a notification
func (n *NotificationInterface) SendNotification(request *models.NotificationPushRequest) error {

	// convert request to json
	data, err := json.Marshal(request)
	if err != nil {
		return err
	}

	// Create new request and set headers
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/notifications", n.ServiceHost), bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	// Send the newly created request
	resp, err := n.Client.Do(req)
	if err != nil {
		return err
	}

	// Check http status
	if resp.StatusCode != 201 {
		return errors.New("status code not created")
	}

	return nil
}
