package service

import (
	"encoding/json"
	"net/http"

	"context"
	"fmt"
	"log"

	"github.com/gorilla/mux"
	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/token"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
)

/*
Create new field service struct
*/
func NewNotificationService(l *logrus.Logger, r *mux.Router) *Service {
	return &Service{Logger: l, Router: r}
}

/*
Create apns client from p8 file, keyid and teamid
*/
func (n *Service) CreateNewClient() {
	authKey, err := token.AuthKeyFromFile("./files/AuthKey_B9S6C6UY9C.p8")
	if err != nil {
		n.Logger.Fatal("token error:", err)
	}

	token := &token.Token{
		AuthKey: authKey,
		// KeyID from developer account (Certificates, Identifiers & Profiles -> Keys)
		KeyID: "B9S6C6UY9C",
		// TeamID from developer account (View Account -> Membership)
		TeamID: "5A6H49Q85D",
	}

	n.client = apns2.NewTokenClient(token)
}

/*
Firebase app to handle notifications in the meantime
*/
func (n *Service) CreateFirebaseApp() {
	opt := option.WithCredentialsFile("./files/diesel-nova-366902-firebase-adminsdk-4b4gi-451ebcb17f.json")
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		n.Logger.Error("error initializing app: " + err.Error())
		return
	}
	n.fApp = app
}

func (n *Service) SendPushNotification() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		var req NotificationRequest

		// decode request
		json.NewDecoder(r.Body).Decode(&req)

		if len(req.Tokens) < 2 {
			n.sendToToken(req)
		} else {
			n.sendMulticastAndHandleErrors(req)
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
	}
}

func (n *Service) SendPushNotificationToTopic() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		var req NotificationRequest

		// decode request
		json.NewDecoder(r.Body).Decode(&req)

		n.sendToTopic(req)

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
	}
}

func (n *Service) SubscribeToTopic() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		var req NotificationRequest

		// decode request
		json.NewDecoder(r.Body).Decode(&req)

		n.subscribeToTopic(req)

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
	}
}

func (n *Service) UnSubscribeFromTopic() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		var req NotificationRequest

		// decode request
		json.NewDecoder(r.Body).Decode(&req)

		n.unsubscribeFromTopic(req)

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
	}
}

func (n *Service) sendToToken(req NotificationRequest) {

	ctx := context.Background()
	client, err := n.fApp.Messaging(ctx)
	if err != nil {
		n.Logger.Error("error getting Messaging client: %v\n", err)
	}

	registrationToken := req.Tokens[0]

	message := &messaging.Message{
		Notification: &messaging.Notification{
			Title: req.Title,
			Body:  req.Body,
		},
		Token: registrationToken,
	}

	response, err := client.Send(ctx, message)
	if err != nil {
		log.Fatalln(err)
	}

	n.Logger.Info("Successfully sent message:", response)
}

func (n *Service) sendToTopic(req NotificationRequest) {

	topic := req.Topic
	ctx := context.Background()
	client, err := n.fApp.Messaging(ctx)
	if err != nil {
		log.Fatalf("error getting Messaging client: %v\n", err)
	}

	message := &messaging.Message{
		Notification: &messaging.Notification{
			Title: req.Title,
			Body:  req.Body,
		},
		Topic: topic,
	}

	response, err := client.Send(ctx, message)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println("Successfully sent message:", response)
}

func (n *Service) sendMulticastAndHandleErrors(req NotificationRequest) {

	registrationTokens := req.Tokens

	ctx := context.Background()
	client, err := n.fApp.Messaging(ctx)
	if err != nil {
		log.Fatalf("error getting Messaging client: %v\n", err)
	}

	message := &messaging.MulticastMessage{
		Notification: &messaging.Notification{
			Title: req.Title,
			Body:  req.Body,
		},
		Tokens: registrationTokens,
	}

	br, err := client.SendMulticast(context.Background(), message)
	if err != nil {
		log.Fatalln(err)
	}

	if br.FailureCount > 0 {
		var failedTokens []string
		for idx, resp := range br.Responses {
			if !resp.Success {
				// The order of responses corresponds to the order of the registration tokens.
				failedTokens = append(failedTokens, registrationTokens[idx])
			}
		}

		fmt.Printf("List of tokens that caused failures: %v\n", failedTokens)
	}
}

func (n *Service) subscribeToTopic(req NotificationRequest) {
	topic := req.Topic

	registrationTokens := req.Tokens

	ctx := context.Background()
	client, err := n.fApp.Messaging(ctx)
	if err != nil {
		log.Fatalf("error getting Messaging client: %v\n", err)
	}

	response, err := client.SubscribeToTopic(ctx, registrationTokens, topic)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(response.SuccessCount, "tokens were subscribed successfully")
}

func (n *Service) unsubscribeFromTopic(req NotificationRequest) {
	topic := req.Topic

	registrationTokens := req.Tokens

	ctx := context.Background()
	client, err := n.fApp.Messaging(ctx)
	if err != nil {
		log.Fatalf("error getting Messaging client: %v\n", err)
	}

	response, err := client.UnsubscribeFromTopic(ctx, registrationTokens, topic)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(response.SuccessCount, "tokens were unsubscribed successfully")
}
