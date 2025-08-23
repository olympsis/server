package notifications

import (
	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (n *Service) NewPost(id *primitive.ObjectID, post *models.PostDao) error {
	return n.carousel.AddJob(1, models.NotificationPushRequest{})
}

func (n *Service) NewAnnouncement(id *primitive.ObjectID, post *models.PostDao) error {
	return n.carousel.AddJob(1, models.NotificationPushRequest{})
}
