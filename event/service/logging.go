package service

import (
	"context"

	"github.com/olympsis/models"
)

func (s *Service) createEventLog(log *models.EventAuditLog) error {
	_, err := s.Database.EventLogsCollection.InsertOne(context.TODO(), log)
	if err != nil {
		return err
	}
	return nil
}
