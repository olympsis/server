package search

import (
	"olympsis-server/database"

	"github.com/sirupsen/logrus"
)

type Service struct {
	Database *database.Database
	Log      *logrus.Logger
}

func NewSearchService(l *logrus.Logger, d *database.Database) *Service {
	return &Service{Log: l, Database: d}
}
