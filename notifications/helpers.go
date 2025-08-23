package notifications

import (
	"context"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (n *Service) findUsers(arr []string) ([]models.User, error) {
	filter := bson.M{
		"uuid": bson.M{
			"$in": arr,
		},
	}
	cursor, err := n.database.UserCol.Find(context.Background(), filter)
	if err != nil {
		return []models.User{}, err
	}

	var users []models.User
	for cursor.Next(context.Background()) {
		var user models.User
		err := cursor.Decode(&user)
		if err != nil {
			n.logger.Error("Failed to decode user!")
		}
		users = append(users, user)
	}

	return users, nil
}

func (n *Service) findUser(userID string) (*models.User, error) {
	filter := bson.M{
		"uuid": userID,
	}

	var user models.User
	err := n.database.UserCol.FindOne(context.Background(), filter).Decode(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (n *Service) findEvent(id primitive.ObjectID) (*models.EventDao, error) {
	filter := bson.M{
		"_id": id,
	}

	var event models.EventDao
	err := n.database.EventsCollection.FindOne(context.Background(), filter).Decode(&event)
	if err != nil {
		return nil, err
	}

	return &event, nil
}

func (n *Service) findClub(id primitive.ObjectID) (*models.ClubDao, error) {
	filter := bson.M{
		"_id": id,
	}

	var club models.ClubDao
	err := n.database.ClubCol.FindOne(context.Background(), filter).Decode(&club)
	if err != nil {
		return nil, err
	}

	return &club, err
}
