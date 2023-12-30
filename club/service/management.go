package service

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MEMBER MANAGEMENT

func (s *Service) InviteMember(clubID *string, uuid *string) bool {
	return false
}

func (s *Service) PromoteMember(clubID *string, memberID *string, role *string) bool {
	return false
}

func (s *Service) DemoteMember(clubID *string, memberID *string, role *string) bool {
	return false
}

func (s *Service) KickAMember(clubID *string, memberID *string) bool {
	return false
}

// POST MANAGEMENT

func (s *Service) PinPost(clubID *string, postID *string) bool {

	// convert club hex id string to object id
	cid, err := primitive.ObjectIDFromHex(*clubID)
	if err != nil {
		s.Logger.Error("Failed to create club object id: " + err.Error())
		return false
	}

	// convert post hex id string to object id
	pid, err := primitive.ObjectIDFromHex(*postID)
	if err != nil {
		s.Logger.Error("Failed to create post object id: " + err.Error())
		return false
	}

	// update club's pinned post
	filter := bson.M{"_id": cid}
	update := bson.M{"$set": bson.M{"pinned_post_id": pid}}
	err = s.UpdateAClub(context.TODO(), filter, update)
	if err != nil {
		s.Logger.Error("Failed to update club: " + err.Error())
		return false
	}

	return true
}

func (s *Service) UnpinPost(clubID *string) bool {

	// convert club hex id string to object id
	cid, err := primitive.ObjectIDFromHex(*clubID)
	if err != nil {
		s.Logger.Error("Failed to create club object id: " + err.Error())
		return false
	}

	// remove club's pinned post
	filter := bson.M{"_id": cid}
	update := bson.M{"$unset": bson.M{"pinned_post_id": 1}}
	err = s.UpdateAClub(context.TODO(), filter, update)
	if err != nil {
		s.Logger.Error("Failed to update club: " + err.Error())
		return false
	}

	return true
}
