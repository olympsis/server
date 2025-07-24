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

func (s *Service) PinPost(clubID *primitive.ObjectID, postID *primitive.ObjectID) bool {

	// update club's pinned post
	filter := bson.M{"_id": clubID}
	update := bson.M{"$set": bson.M{"pinned_post_id": postID}}
	err := s.UpdateClub(context.TODO(), filter, update)
	if err != nil {
		s.Logger.Error("Failed to update club: " + err.Error())
		return false
	}

	return true
}

func (s *Service) UnpinPost(clubID *primitive.ObjectID) bool {

	// remove club's pinned post
	filter := bson.M{"_id": clubID}
	update := bson.M{"$unset": bson.M{"pinned_post_id": 1}}
	err := s.UpdateClub(context.TODO(), filter, update)
	if err != nil {
		s.Logger.Error("Failed to update club: " + err.Error())
		return false
	}

	return true
}
