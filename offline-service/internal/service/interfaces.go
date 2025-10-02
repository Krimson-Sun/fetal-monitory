package service

import (
	"context"
	"offline-service/pkg/models"
)

type CacheRepository interface {
	SaveSession(ctx context.Context, sessionID string, data *models.MedicalSession) error
	GetSession(ctx context.Context, sessionID string) (*models.MedicalSession, error)
	DeleteSession(ctx context.Context, sessionID string) error
	GetStats() map[string]interface{}
	CheckConnection(ctx context.Context) error
	Close() error
}

type DBRepository interface {
	SaveSession(ctx context.Context, session *models.MedicalSession) error
}
