package service

import (
	"context"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/bson"
)

func (s *Service) FindCountries(ctx context.Context, filter bson.M) (*[]models.Country, error) {
	var countries []models.Country
	cursor, err := s.Database.CountriesCol.Find(ctx, filter)
	if err != nil {
		return nil, err
	}

	for cursor.Next(context.TODO()) {
		var country models.Country
		err := cursor.Decode(&country)
		if err != nil {
			return nil, err
		}
		countries = append(countries, country)
	}
	return &countries, nil
}

func (s *Service) FindAdministrativeAreas(ctx context.Context, filter bson.M) (*[]models.AdministrativeArea, error) {
	var adminAreas []models.AdministrativeArea
	cursor, err := s.Database.AdminAreasCol.Find(ctx, filter)
	if err != nil {
		return nil, err
	}

	for cursor.Next(context.TODO()) {
		var area models.AdministrativeArea
		err := cursor.Decode(&area)
		if err != nil {
			return nil, err
		}
		adminAreas = append(adminAreas, area)
	}
	return &adminAreas, nil
}

func (s *Service) FindSubAdministrativeAreas(ctx context.Context, filter bson.M) (*[]models.SubAdministrativeArea, error) {
	var subAdminAreas []models.SubAdministrativeArea
	cursor, err := s.Database.SubAdminAreasCol.Find(ctx, filter)
	if err != nil {
		return nil, err
	}

	for cursor.Next(context.TODO()) {
		var area models.SubAdministrativeArea
		err := cursor.Decode(&area)
		if err != nil {
			return nil, err
		}
		subAdminAreas = append(subAdminAreas, area)
	}
	return &subAdminAreas, nil
}
