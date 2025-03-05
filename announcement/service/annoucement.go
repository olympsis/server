package service

import (
	"context"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Insert new announcement into database
func (s *Service) InsertAnnouncement(ctx context.Context, announcement *models.AnnouncementDao) (*primitive.ObjectID, error) {
	resp, err := s.Database.AnnouncementCol.InsertOne(ctx, announcement)
	if err != nil {
		return nil, err
	}
	id := resp.InsertedID.(primitive.ObjectID)
	return &id, err
}

// Get announcement from database
func (s *Service) FindAnnouncement(ctx context.Context, filter interface{}) (*models.Announcement, error) {
	var announcement models.Announcement
	err := s.Database.AnnouncementCol.FindOne(ctx, filter).Decode(&announcement)
	if err != nil {
		return nil, err
	}
	return &announcement, nil
}

// get announcements from database
func (s *Service) FindAnnouncements(ctx context.Context, filter interface{}) (*[]models.Announcement, error) {
	var announcements []models.Announcement
	cursor, err := s.Database.AnnouncementCol.Find(ctx, filter)
	if err != nil {
		return nil, err
	}

	for cursor.Next(context.TODO()) {
		var announcement models.Announcement
		err := cursor.Decode(&announcement)
		if err != nil {
			return nil, err
		}
		announcements = append(announcements, announcement)
	}
	return &announcements, nil
}

// update announcement in database
func (s *Service) ModifyAnnouncement(ctx context.Context, filter interface{}, update interface{}) error {
	// update announcement
	_, err := s.Database.AnnouncementCol.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	return nil
}

// delete announcement in database
func (s *Service) DeleteAnnouncement(ctx context.Context, filter interface{}) error {
	// delete announcement
	_, err := s.Database.AnnouncementCol.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}
