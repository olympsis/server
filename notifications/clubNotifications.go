package notifications

import (
	"time"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (n *Service) NewApplication(id *primitive.ObjectID, application *models.ClubApplicationDao) error {
	// Load club info
	club, err := n.findClub(*id)
	if err != nil {
		return err
	}

	user, err := n.findUser(*application.Applicant)
	if err != nil {
		return err
	}

	clubID := id.Hex()
	name := *club.Name
	timestamp := primitive.NewDateTimeFromTime(time.Now())
	note := models.PushNotification{
		ID:       primitive.NewObjectID(),
		Title:    "New Club Application!",
		Body:     name,
		Type:     "push",
		Category: "events",
		Data: map[string]any{
			"type":           models.NewClubApplicationType,
			"group_type":     "club",
			"group_id":       clubID,
			"group_name":     name,
			"group_logo_url": club.Logo,
			"username":       user.UserName,
			"image_url":      user.ImageURL,
		},
		CreatedAt: timestamp,
	}
	request := models.NotificationPushRequest{
		Topic:        &clubID,
		Notification: note,
	}
	return n.carousel.AddJob(1, request)
}

func (n *Service) ApplicationUpdate(id primitive.ObjectID, app *models.ClubApplicationDao) error {
	// Load club info
	club, err := n.findClub(id)
	if err != nil {
		return err
	}

	clubID := id.Hex()
	name := *club.Name
	users := []string{*app.Applicant}
	timestamp := primitive.NewDateTimeFromTime(time.Now())
	note := models.PushNotification{
		ID:       primitive.NewObjectID(),
		Title:    "Your application was approved!",
		Body:     name,
		Type:     "push",
		Category: "events",
		Data: map[string]any{
			"type":           models.ClubApplicationUpdateType,
			"group_type":     "club",
			"group_id":       clubID,
			"group_name":     name,
			"group_logo_url": club.Logo,
		},
		CreatedAt: timestamp,
	}
	request := models.NotificationPushRequest{
		Users:        &users,
		Notification: note,
	}
	return n.carousel.AddJob(1, request)
}

func (n *Service) ChangedRole(id primitive.ObjectID, userID string, previous models.MemberRole, new models.MemberRole) error {
	// Load club info
	club, err := n.findClub(id)
	if err != nil {
		return err
	}

	clubID := id.Hex()
	name := *club.Name
	users := []string{userID}
	timestamp := primitive.NewDateTimeFromTime(time.Now())
	note := models.PushNotification{
		ID:       primitive.NewObjectID(),
		Title:    "Your Rank has changed!",
		Body:     name,
		Type:     "push",
		Category: "events",
		Data: map[string]any{
			"type":           models.RankingChangeType,
			"group_type":     "club",
			"group_id":       clubID,
			"group_name":     name,
			"group_logo_url": club.Logo,
		},
		CreatedAt: timestamp,
	}
	request := models.NotificationPushRequest{
		Users:        &users,
		Notification: note,
	}
	return n.carousel.AddJob(1, request)
}

func (n *Service) Suspended(id *primitive.ObjectID, member *models.MemberDao) error {
	return n.carousel.AddJob(1, models.NotificationPushRequest{})
}

func (n *Service) Kicked(id *primitive.ObjectID, member *models.MemberDao) error {
	// Load club info
	club, err := n.findClub(*id)
	if err != nil {
		return err
	}

	clubID := id.Hex()
	name := *club.Name
	users := []string{member.UserID}
	timestamp := primitive.NewDateTimeFromTime(time.Now())
	note := models.PushNotification{
		ID:       primitive.NewObjectID(),
		Title:    "You've been kicked from the club!",
		Body:     name,
		Type:     "push",
		Category: "events",
		Data: map[string]any{
			"type":           models.ExpulsionType,
			"group_type":     "club",
			"group_id":       clubID,
			"group_name":     name,
			"group_logo_url": club.Logo,
		},
		CreatedAt: timestamp,
	}
	request := models.NotificationPushRequest{
		Users:        &users,
		Notification: note,
	}
	return n.carousel.AddJob(1, request)
}
