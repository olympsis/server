package notifications

import (
	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (n *Service) NewApplication(id primitive.ObjectID, application models.ClubApplicationDao) error {
	return n.carousel.AddJob(1, models.NotificationPushRequest{})
}

func (n *Service) ApplicationUpdate(id primitive.ObjectID, appID primitive.ObjectID) error {
	return n.carousel.AddJob(1, models.NotificationPushRequest{})
}

func (n *Service) ChangedRole(id primitive.ObjectID, userID string, previous models.MemberRole, new models.MemberRole) error {
	return n.carousel.AddJob(1, models.NotificationPushRequest{})
}

func (n *Service) Kicked(id *primitive.ObjectID, member *models.MemberDao) error {
	return n.carousel.AddJob(1, models.NotificationPushRequest{})
}
