package push

import (
	"context"
	"time"

	"olympsis-server/database"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// repo is the push package's data layer. It reads events/topics/users and writes
// the notification audit trail, against the SAME collections the legacy
// notifications package uses (so the in-app inbox stays consistent).
type repo struct{ db *database.Database }

// dbTimeout bounds each write so a slow Mongo call can't wedge the dispatcher's
// worker (which would also stall graceful-shutdown drain).
const dbTimeout = 5 * time.Second

func (r repo) findEvent(id bson.ObjectID) (*models.EventDao, error) {
	var event models.EventDao
	err := r.db.EventsCollection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&event)
	if err != nil {
		return nil, err
	}
	return &event, nil
}

// findTopic resolves a single topic by name (used for the event-id topic, whose
// members are the event's participants).
func (r repo) findTopic(name string) (*models.NotificationTopic, error) {
	var topic models.NotificationTopic
	err := r.db.NotificationTopicsCollection.FindOne(context.Background(), bson.M{"name": name}).Decode(&topic)
	if err != nil {
		return nil, err
	}
	return &topic, nil
}

// findTopics resolves several topics by name in one query (used for an event's
// organizer topics).
func (r repo) findTopics(names []string) ([]models.NotificationTopic, error) {
	filter := bson.M{"name": bson.M{"$in": names}}
	cursor, err := r.db.NotificationTopicsCollection.Find(context.Background(), filter)
	if err != nil {
		return nil, err
	}
	var topics []models.NotificationTopic
	if err := cursor.All(context.Background(), &topics); err != nil {
		return nil, err
	}
	return topics, nil
}

func (r repo) findUsers(ids []string) ([]models.User, error) {
	filter := bson.M{"user_id": bson.M{"$in": ids}}
	cursor, err := r.db.UserCollection.Find(context.Background(), filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var users []models.User
	if err := cursor.All(context.Background(), &users); err != nil {
		return nil, err
	}
	return users, nil
}

func (r repo) createPushNotification(note *models.PushNotification) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	_, err := r.db.PushNotificationsCollection.InsertOne(ctx, note)
	return err
}

func (r repo) createUserNotification(note *models.UserNotification) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	_, err := r.db.UserNotificationsCollection.InsertOne(ctx, note)
	return err
}

func (r repo) createNotificationLog(log *models.NotificationLog) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	_, err := r.db.NotificationLogsCollection.InsertOne(ctx, log)
	return err
}

// deactivateDevice flips Active=false on the user's device whose token is dead,
// so the dispatcher stops sending to it. The positional `$` updates the matching
// element of the notification_devices array.
func (r repo) deactivateDevice(userID, token string) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	filter := bson.M{"user_id": userID, "notification_devices.token": token}
	update := bson.M{"$set": bson.M{"notification_devices.$.active": false}}
	_, err := r.db.UserCollection.UpdateOne(ctx, filter, update)
	return err
}
