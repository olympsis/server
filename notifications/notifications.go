package notifications

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"olympsis-server/database"
	"sync"
	"time"

	"github.com/olympsis/models"
	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/payload"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Carousel struct {
	priorityQueue PriorityQueue
	mu            sync.Mutex
	cond          *sync.Cond
	logger        *logrus.Logger
	onProcessJob  func(*models.NotificationPushRequest) error
}

type Service struct {
	client   *apns2.Client
	logger   *logrus.Logger
	database *database.Database
	carousel *Carousel
}

func New(c *apns2.Client, l *logrus.Logger, db *database.Database) *Service {
	service := Service{
		client:   c,
		logger:   l,
		database: db,
	}
	service.carousel = NewCarousel(l, service.processPushRequest)
	service.carousel.Start()
	return &service
}

func (n *Service) CreateTopic(name string, users []string) error {
	isActive := true
	timestamp := bson.NewDateTimeFromTime(time.Now())
	return n.createNotificationTopic(&models.NotificationTopicDao{
		Name:      &name,
		Users:     &users,
		IsActive:  &isActive,
		CreatedAt: &timestamp,
	})
}

func (n *Service) AddUsersToTopic(name string, users []string) error {
	return n.updateNotificationTopic(bson.M{"name": name}, users, true)
}

func (n *Service) RemoveUsersFromTopic(name string, users []string) error {
	return n.updateNotificationTopic(bson.M{"name": name}, users, false)
}

func (n *Service) RemoveTopic(name string) error {
	return n.deleteNotificationTopic(bson.M{"name": name})
}

func (n *Service) DisableTopic(name string) error {
	return n.disableNotificationTopic(bson.M{"name": name})
}

func (n *Service) AddNoteToCarousel(priority int, request *models.NotificationPushRequest) error {
	return n.carousel.AddJob(1, *request)
}

func (n *Service) processPushRequest(request *models.NotificationPushRequest) error {
	// Create a new PushNotification record
	pushNotif := &models.PushNotification{
		ID:        bson.NewObjectID(),
		Title:     request.Notification.Title,
		Body:      request.Notification.Body,
		Type:      request.Notification.Type,
		Category:  request.Notification.Category,
		Data:      request.Notification.Data,
		CreatedAt: bson.NewDateTimeFromTime(time.Now()),
	}
	if err := n.createPushNotification(pushNotif); err != nil {
		return err
	}

	// Get users from topic if specified
	var users []string
	if request.Topic != nil {
		topic, err := n.findNotificationTopic(bson.M{"name": *request.Topic})
		if err != nil {
			return err
		}
		users = topic.Users
	} else if request.Users != nil {
		users = *request.Users
	}

	// Send to all users
	userDetails, err := n.findUsers(users)
	if err != nil {
		return err
	}

	for _, user := range userDetails {
		// Create UserNotification for each user
		userNotif := &models.UserNotification{
			ID:             bson.NewObjectID(),
			UUID:           user.UUID,
			NotificationID: pushNotif.ID,
			IsRead:         false,
			CreatedAt:      bson.NewDateTimeFromTime(time.Now()),
		}

		if err := n.createUserNotification(userNotif); err != nil {
			n.logger.Error("Failed to create user notification:", err)
			continue
		}

		// Send push notification to each device
		if user.NotificationDevices != nil && user.NotificationPreference != nil {
			for _, device := range *user.NotificationDevices {

				// Object log the notification attempt
				log := &models.NotificationLog{
					ID:             bson.NewObjectID(),
					NotificationID: pushNotif.ID,
					Platform:       device.DeviceInfo.Platform,
					Status:         "sent",
					CreatedAt:      bson.NewDateTimeFromTime(time.Now()),
				}

				switch device.DeviceInfo.Platform {
				case "ios":
					if user.NotificationPreference.Types["push"] {
						err := n.pushAppleNote(&request.Notification, device.Token)
						if err != nil {
							errStr := err.Error()
							log.Status = "failed"
							log.Error = &errStr
						}

						if err := n.createNotificationLog(log); err != nil {
							n.logger.Error("Failed to create notification log:", err)
						}
					}
				case "android":
					if user.NotificationPreference.Types["android"] {
						n.logger.Error("Android Push Notifications not supported yet")
					}
				case "web":
					if user.NotificationPreference.Types["web"] {
						n.logger.Error("Web Push Notifications not supported yet")
					}
				default:
					n.logger.Error("Invalid device platform")
				}
			}
		}
	}

	return nil
}

func (n *Service) pushAppleNote(note *models.PushNotification, token string) error {
	notification := &apns2.Notification{}
	notification.DeviceToken = token
	notification.Topic = "com.olympsis.client"
	notification.Priority = 5

	payload := payload.NewPayload()
	payload.AlertTitle(note.Title)
	payload.AlertBody(note.Body)
	payload.Badge(1)

	// custom notification data
	if note.Data != nil {
		for key, value := range note.Data {
			payload.Custom(key, value)
		}
	}

	notification.Payload = payload
	notification.Priority = 5

	res, err := n.client.Push(notification)
	if err != nil {
		return err
	}

	if res.Sent() {
		n.logger.Debug("Sent:", res.ApnsID)
		return nil
	} else {
		errString := fmt.Sprintf("Not Sent: %v %v %v\n", res.StatusCode, res.ApnsID, res.Reason)
		return errors.New(errString)
	}
}

func (n *Service) HandleNotificationRequest() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		var request models.NotificationPushRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			n.logger.Errorf("Failed to decode request. Error: %s", err.Error())
			http.Error(rw, `{"msg": "bad request"}`, http.StatusBadRequest)
		}

		if err := n.AddNoteToCarousel(1, &request); err != nil {
			n.logger.Errorf("Failed to add note to carousel. Error: %s", err.Error())
			http.Error(rw, `{"msg": "failed to handle notif"}`, http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{"msg": "OK"}`))
	}
}
