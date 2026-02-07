package service

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// POST MANAGEMENT

func (s *Service) PinPost(orgID *string, postID *string) bool {

	// convert club hex id string to object id
	cid, err := bson.ObjectIDFromHex(*orgID)
	if err != nil {
		s.Logger.Error("Failed to create org object id: " + err.Error())
		return false
	}

	// convert post hex id string to object id
	pid, err := bson.ObjectIDFromHex(*postID)
	if err != nil {
		s.Logger.Error("Failed to create post object id: " + err.Error())
		return false
	}

	// update club's pinned post
	filter := bson.M{"_id": cid}
	update := bson.M{"$set": bson.M{"pinned_post_id": pid}}
	err = s.UpdateAnOrganization(context.TODO(), filter, update)
	if err != nil {
		s.Logger.Error("Failed to update club: " + err.Error())
		return false
	}

	return true
}

func (s *Service) UnpinPost(orgID *string) bool {

	// convert club hex id string to object id
	cid, err := bson.ObjectIDFromHex(*orgID)
	if err != nil {
		s.Logger.Error("Failed to create club object id: " + err.Error())
		return false
	}

	// remove club's pinned post
	filter := bson.M{"_id": cid}
	update := bson.M{"$unset": bson.M{"pinned_post_id": 1}}
	err = s.UpdateAnOrganization(context.TODO(), filter, update)
	if err != nil {
		s.Logger.Error("Failed to update club: " + err.Error())
		return false
	}

	return true
}
