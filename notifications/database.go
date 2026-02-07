package notifications

import (
	"context"
	"time"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func (n *Service) createNotificationLog(log *models.NotificationLog) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := n.database.NotificationLogsCollection.InsertOne(ctx, log)
	return err
}

func (n *Service) createPushNotification(note *models.PushNotification) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := n.database.PushNotificationsCollection.InsertOne(ctx, note)
	return err
}

func (n *Service) createUserNotification(note *models.UserNotification) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := n.database.UserNotificationsCollection.InsertOne(ctx, note)
	return err
}

// func (n *Service) updateUserNotification(filter bson.M, updates bson.M) error {
// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	defer cancel()

// 	_, err := n.database.UserNotificationsCollection.UpdateMany(ctx, filter, updates)
// 	return err
// }

// func (n *Service) deleteUserNotification(filter bson.M) error {
// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	defer cancel()

// 	_, err := n.database.UserNotificationsCollection.DeleteMany(ctx, filter)
// 	return err
// }

func (n *Service) createNotificationTopic(topic *models.NotificationTopicDao) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := n.database.NotificationTopicsCollection.InsertOne(ctx, topic)
	return err
}

func (n *Service) findNotificationTopic(filter bson.M) (*models.NotificationTopic, error) {
	var topic models.NotificationTopic
	err := n.database.NotificationTopicsCollection.FindOne(context.Background(), filter).Decode(&topic)
	if err != nil {
		return nil, err
	}

	return &topic, nil
}

func (n *Service) findNotificationTopics(filter bson.M) ([]models.NotificationTopic, error) {
	var topics []models.NotificationTopic
	cursor, err := n.database.NotificationTopicsCollection.Find(context.Background(), filter)
	if err != nil {
		return nil, err
	}

	err = cursor.All(context.Background(), &topics)
	if err != nil {
		return nil, err
	}

	return topics, nil
}

func (n *Service) updateNotificationTopic(filter bson.M, users []string, add bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	timestamp := bson.NewDateTimeFromTime(time.Now())

	var updates bson.M
	if add { // Handle add users operation
		updates = bson.M{
			"$push": bson.M{
				"users": bson.M{
					"$each": users,
				},
			},
			"$set": bson.M{
				"updated_at": timestamp,
			},
		}
	} else { // Handle remove users operation
		updates = bson.M{
			"$pull": bson.M{
				"users": bson.M{
					"$in": users,
				},
			},
			"$set": bson.M{
				"updated_at": timestamp,
			},
		}
	}

	_, err := n.database.NotificationTopicsCollection.UpdateOne(ctx, filter, updates)
	return err
}

func (n *Service) disableNotificationTopic(filter bson.M) error {
	update := bson.M{
		"$set": bson.M{
			"is_active": false,
		},
	}
	_, err := n.database.NotificationTopicsCollection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		return err
	}

	return nil
}

func (n *Service) deleteNotificationTopic(filter bson.M) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := n.database.NotificationTopicsCollection.DeleteOne(ctx, filter)
	return err
}
