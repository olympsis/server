package notifications

import (
	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func (n *Service) NewPost(id *bson.ObjectID, post *models.PostDao) error {
	return n.carousel.AddJob(1, models.NotificationPushRequest{})
}

func (n *Service) NewAnnouncement(id *bson.ObjectID, post *models.PostDao) error {
	return n.carousel.AddJob(1, models.NotificationPushRequest{})
}
