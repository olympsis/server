package notifications

import (
	"errors"
	"time"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// New event created
func (n *Service) NewEvent(id *primitive.ObjectID, event *models.EventDao) error {
	// Lets notify all of the group organizers and their members
	if event.Organizers == nil {
		return errors.New("no event organizers found")
	}

	// Find the topics
	var organizerTopics []string
	organizers := *event.Organizers
	for idx := range organizers {
		organizerTopics = append(organizerTopics, organizers[idx].ID.Hex())
	}

	filter := bson.M{
		"name": bson.M{
			"$in": organizerTopics,
		},
	}
	topics, err := n.findNotificationTopics(filter)
	if err != nil {
		return err
	}

	// Grab all of the users from the topics and use a map to prevent duplicate user IDs
	var users []string
	usersSet := make(map[string]struct{})
	for i := range topics {
		for j := range topics[i].Users {
			if topics[i].Users[j] != *event.PosterID {
				usersSet[topics[i].Users[j]] = struct{}{}
			}
		}
	}

	// Move users set to array
	for k := range usersSet {
		users = append(users, k)
	}

	// Fetch poster data
	user, err := n.findUser(*event.PosterID)
	if err != nil {
		return err
	}

	timestamp := primitive.NewDateTimeFromTime(time.Now())
	note := models.PushNotification{
		ID:       primitive.NewObjectID(),
		Title:    "New event created!",
		Body:     *event.Title,
		Type:     "push",
		Category: "events",
		Data: map[string]interface{}{
			"type":            models.NewEventType,
			"event_id":        id.Hex(),
			"event_name":      event.Title,
			"event_media_url": event.MediaURL,
			"username":        user.UserName,
			"image_url":       user.ImageURL,
		},
		CreatedAt: timestamp,
	}

	request := models.NotificationPushRequest{
		Users:        &users,
		Notification: note,
	}

	return n.carousel.AddJob(1, request)
}

// Cancels an event
func (n *Service) CancelEvent(id *primitive.ObjectID, actor string) error {
	// Fetch event data
	event, err := n.findEvent(*id)
	if err != nil {
		return err
	}
	// Lets notify all of the group organizers and their members
	if event.Organizers == nil {
		return errors.New("no event organizers found")
	}

	// Find the topics
	var organizerTopics []string
	organizers := *event.Organizers
	for idx := range organizers {
		organizerTopics = append(organizerTopics, organizers[idx].ID.Hex())
	}

	filter := bson.M{
		"name": bson.M{
			"$in": organizerTopics,
		},
	}
	topics, err := n.findNotificationTopics(filter)
	if err != nil {
		return err
	}

	// Grab all of the users from the topics and use a map to prevent duplicate user IDs
	var users []string
	usersSet := make(map[string]struct{})
	for i := range topics {
		for j := range topics[i].Users {
			if topics[i].Users[j] != actor {
				usersSet[topics[i].Users[j]] = struct{}{}
			}
		}
	}

	// Move users set to array
	for k := range usersSet {
		users = append(users, k)
	}

	timestamp := primitive.NewDateTimeFromTime(time.Now())
	note := models.PushNotification{
		ID:       primitive.NewObjectID(),
		Title:    "Event has been cancelled!",
		Body:     *event.Title,
		Type:     "push",
		Category: "events",
		Data: map[string]interface{}{
			"type":            models.EventCancellation,
			"event_id":        id.Hex(),
			"event_name":      event.Title,
			"event_media_url": event.MediaURL,
		},
		CreatedAt: timestamp,
	}

	request := models.NotificationPushRequest{
		Users:        &users,
		Notification: note,
	}

	return n.carousel.AddJob(1, request)
}

// Someone commented
func (n *Service) NewEventComment(id primitive.ObjectID, comment string) error {
	// Fetch event data
	event, err := n.findEvent(id)
	if err != nil {
		return err
	}

	// Lets notify all of the group organizers and their members
	if event.Organizers == nil {
		return errors.New("no event organizers found")
	}

	// Find the topics
	var organizerTopics []string
	organizers := *event.Organizers
	for idx := range organizers {
		organizerTopics = append(organizerTopics, organizers[idx].ID.Hex())
	}

	filter := bson.M{
		"name": bson.M{
			"$in": organizerTopics,
		},
	}
	topics, err := n.findNotificationTopics(filter)
	if err != nil {
		return err
	}

	// Grab all of the users from the topics and use a map to prevent duplicate user IDs
	var users []string
	usersSet := make(map[string]struct{})
	for i := range topics {
		for j := range topics[i].Users {
			if topics[i].Users[j] != *event.PosterID {
				usersSet[topics[i].Users[j]] = struct{}{}
			}
		}
	}

	// Move users set to array
	for k := range usersSet {
		users = append(users, k)
	}

	// Fetch poster data
	user, err := n.findUser(*event.PosterID)
	if err != nil {
		return err
	}

	// Create notification object
	timestamp := primitive.NewDateTimeFromTime(time.Now())
	note := models.PushNotification{
		ID:       primitive.NewObjectID(),
		Title:    "New Event Comment!",
		Body:     *event.Title,
		Type:     "push",
		Category: "events",
		Data: map[string]any{
			"type":            models.NewEventCommentType,
			"event_id":        id.Hex(),
			"event_name":      event.Title,
			"event_media_url": event.MediaURL,
			"event_comment":   comment,
			"username":        user.UserName,
			"image_url":       user.ImageURL,
		},
		CreatedAt: timestamp,
	}

	request := models.NotificationPushRequest{
		Users:        &users,
		Notification: note,
	}

	return n.carousel.AddJob(1, request)
}

// Event starts soon
func (n *Service) EventReminder(id string) error {
	// Fetch event data
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	event, err := n.findEvent(oid)
	if err != nil {
		return err
	}

	// Create notification object
	timestamp := primitive.NewDateTimeFromTime(time.Now())
	note := models.PushNotification{
		ID:       primitive.NewObjectID(),
		Title:    "Event starts in 30 minutes!",
		Body:     *event.Title,
		Type:     "push",
		Category: "events",
		Data: map[string]any{
			"type":            models.EventReminderType,
			"event_id":        id,
			"event_name":      event.Title,
			"event_media_url": event.MediaURL,
		},
		CreatedAt: timestamp,
	}

	request := models.NotificationPushRequest{
		Topic:        &id,
		Notification: note,
	}

	return n.carousel.AddJob(1, request)
}

// Event participant kicked
func (n *Service) ParticipantKick(event *models.EventDao, participant *models.ParticipantDao) error {
	// Create notification object
	timestamp := primitive.NewDateTimeFromTime(time.Now())
	note := models.PushNotification{
		ID:       primitive.NewObjectID(),
		Title:    "You've been kicked from the participants list.",
		Body:     *event.Title,
		Type:     "push",
		Category: "events",
		Data: map[string]any{
			"type":            models.EventParticipantKickType,
			"event_id":        event.ID,
			"event_name":      event.Title,
			"event_media_url": event.MediaURL,
		},
		CreatedAt: timestamp,
	}

	users := []string{*participant.UserID}
	request := models.NotificationPushRequest{
		Users:        &users,
		Notification: note,
	}

	return n.carousel.AddJob(1, request)
}

// Event participant waitlist promotion
func (n *Service) WaitlistPromotion(event *models.EventDao, participant *models.ParticipantDao) error {
	// Create notification object
	timestamp := primitive.NewDateTimeFromTime(time.Now())
	note := models.PushNotification{
		ID:       primitive.NewObjectID(),
		Title:    "You've been promoted from the waitlist.",
		Body:     *event.Title,
		Type:     "push",
		Category: "events",
		Data: map[string]any{
			"type":            models.EventParticipantWaitlistUpgradeType,
			"event_id":        event.ID,
			"event_name":      event.Title,
			"event_media_url": event.MediaURL,
		},
		CreatedAt: timestamp,
	}

	users := []string{*participant.UserID}
	request := models.NotificationPushRequest{
		Users:        &users,
		Notification: note,
	}

	return n.carousel.AddJob(1, request)
}
