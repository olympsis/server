package service

import (
	"context"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
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

// Get announcement from database using DAO
func (s *Service) FindAnnouncementDao(ctx context.Context, filter bson.M) (*models.AnnouncementDao, error) {
	var announcement models.AnnouncementDao
	err := s.Database.AnnouncementCol.FindOne(ctx, filter).Decode(&announcement)
	if err != nil {
		return nil, err
	}
	return &announcement, nil
}

// Find multiple announcement DAOs
func (s *Service) FindAnnouncements(ctx context.Context, filter bson.M, opts *options.FindOptions) ([]models.AnnouncementDao, error) {
	var announcements []models.AnnouncementDao
	cursor, err := s.Database.AnnouncementCol.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}

	defer cursor.Close(ctx)
	if err = cursor.All(ctx, &announcements); err != nil {
		return nil, err
	}

	return announcements, nil
}

// Update announcement in database
func (s *Service) ModifyAnnouncement(ctx context.Context, filter bson.M, update bson.M) error {
	_, err := s.Database.AnnouncementCol.UpdateOne(ctx, filter, update)
	return err
}

// Update multiple announcements in database
func (s *Service) ModifyAnnouncements(ctx context.Context, filter bson.M, update bson.M) error {
	_, err := s.Database.AnnouncementCol.UpdateMany(ctx, filter, update)
	return err
}

// Delete announcement from database
func (s *Service) RemoveAnnouncement(ctx context.Context, filter bson.M) error {
	_, err := s.Database.AnnouncementCol.DeleteOne(ctx, filter)
	return err
}
