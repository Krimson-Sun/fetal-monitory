package service

import (
	"context"
	"offline-service/pkg/models"
)

type CacheRepository interface {
	SaveSession(ctx context.Context, sessionID string, data *models.MedicalSession) error
	GetSession(ctx context.Context, sessionID string) (*models.MedicalSession, error)
	DeleteSession(ctx context.Context, sessionID string) error
}

type DBRepository interface {
	SaveMedicalRecord(ctx context.Context, data *models.MedicalSession) error
}
