package service

import (
	"bytes"
	"encoding/json"
	"net/http"
)

/*
Sends Notifications to topic

  - notifices all devices listening to a topic

Returns:

	err - if there is an error
*/
func (e *Service) SendNotificationToTopic(t string, b string, tpc string) error {
	client := &http.Client{}

	request := NotificationRequest{
		Title: t,
		Body:  b,
		Topic: tpc,
	}

	data, err := json.Marshal(request)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", "http://192.168.0.109/v1/pushnote/topic", bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	return nil
}

/*
Fetch User

  - grab user data

Returns:

	LookupUser
		- user information
*/
func (e *Service) SubscribeToEventTopic(tpc string, tks []string) error {
	client := &http.Client{}

	request := NotificationRequest{
		Topic:  tpc,
		Tokens: tks,
	}

	data, err := json.Marshal(request)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", "http://192.168.0.109/v1/pushnote/topic", bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	return nil
}

/*
Fetch User

  - grab user data

Returns:

	LookupUser
		- user information
*/
func (e *Service) UnsubscribeFromEventTopic(tpc string, tks []string) error {
	client := &http.Client{}

	request := NotificationRequest{
		Topic:  tpc,
		Tokens: tks,
	}

	data, err := json.Marshal(request)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("DELETE", "http://192.168.0.109/v1/pushnote/topic", bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	return nil
}
