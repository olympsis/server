package notifications

import (
	"time"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func (n *Service) NewApplication(id *bson.ObjectID, application *models.ClubApplicationDao) error {
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
	timestamp := bson.NewDateTimeFromTime(time.Now())
	note := models.PushNotification{
		ID:       bson.NewObjectID(),
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

func (n *Service) ApplicationUpdate(id bson.ObjectID, app *models.ClubApplicationDao) error {
	// Load club info
	club, err := n.findClub(id)
	if err != nil {
		return err
	}

	clubID := id.Hex()
	name := *club.Name
	users := []string{*app.Applicant}
	timestamp := bson.NewDateTimeFromTime(time.Now())
	note := models.PushNotification{
		ID:       bson.NewObjectID(),
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

func (n *Service) ChangedRole(id bson.ObjectID, userID string, previous models.MemberRole, new models.MemberRole) error {
	// Load club info
	club, err := n.findClub(id)
	if err != nil {
		return err
	}

	clubID := id.Hex()
	name := *club.Name
	users := []string{userID}
	timestamp := bson.NewDateTimeFromTime(time.Now())
	note := models.PushNotification{
		ID:       bson.NewObjectID(),
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

func (n *Service) Suspended(id *bson.ObjectID, member *models.MemberDao) error {
	return n.carousel.AddJob(1, models.NotificationPushRequest{})
}

func (n *Service) Kicked(id *bson.ObjectID, member *models.MemberDao) error {
	// Load club info
	club, err := n.findClub(*id)
	if err != nil {
		return err
	}

	clubID := id.Hex()
	name := *club.Name
	users := []string{member.UserID}
	timestamp := bson.NewDateTimeFromTime(time.Now())
	note := models.PushNotification{
		ID:       bson.NewObjectID(),
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
