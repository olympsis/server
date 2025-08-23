package service

import (
	"context"

	"github.com/olympsis/models"
)

// adds invitation document to database
func (s *Service) InsertAnInvitation(ctx context.Context, invitation *models.Invitation) error {
	s.Database.OrgInvitationCollection.InsertOne(ctx, invitation)
	return nil
}

// finds and returns an invitation from database
func (s *Service) FindAnInvitation(ctx context.Context, filter interface{}, invitation *models.Invitation) error {
	err := s.Database.OrgInvitationCollection.FindOne(ctx, filter).Decode(&invitation)
	if err != nil {
		return err
	}
	return nil
}

// finds multiple invitations from database
func (s *Service) FindInvitations(ctx context.Context, filter interface{}, invitations *[]models.Invitation) error {

	cursor, err := s.Database.OrgInvitationCollection.Find(ctx, filter)
	if err != nil {
		return err
	}

	for cursor.Next(context.TODO()) {
		var invite models.Invitation
		err := cursor.Decode(&invite)
		if err != nil {
			return err
		}
		*invitations = append(*invitations, invite)
	}
	return nil
}

// updates in invitation in the database
func (s *Service) UpdateAnInvitation(ctx context.Context, filter interface{}, update interface{}, invitation *models.Invitation) error {
	// update user
	_, err := s.Database.OrgInvitationCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	// find and return updated user
	err = s.FindAnInvitation(ctx, filter, invitation)
	if err != nil {
		return err
	}

	return nil
}

// updates multiple invitations in the database
func (s *Service) UpdateInvitations(ctx context.Context, filter interface{}, update interface{}, invitations *[]models.Invitation) error {

	// update event
	_, err := s.Database.OrgInvitationCollection.UpdateMany(ctx, filter, update)
	if err != nil {
		return err
	}

	// find updated users
	err = s.FindInvitations(ctx, filter, invitations)
	if err != nil {
		return err
	}

	return nil
}

// deletes an invitation from the database
func (s *Service) DeleteAnInvitation(ctx context.Context, filter interface{}) error {

	// delete user
	_, err := s.Database.OrgInvitationCollection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}

// delete invitations in database
func (s *Service) DeleteInvitations(ctx context.Context, filter interface{}) error {

	// delete users
	_, err := s.Database.OrgInvitationCollection.DeleteMany(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}
