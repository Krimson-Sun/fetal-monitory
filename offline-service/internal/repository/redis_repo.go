package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"offline-service/pkg/models"
)

type RedisRepository struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisRepository(addr, password string, db int, ttl time.Duration) *RedisRepository {
	return &RedisRepository{
		client: redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
		}),
		ttl: ttl,
	}
}

func (r *RedisRepository) SaveSession(ctx context.Context, sessionID string, session *models.MedicalSession) error {
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	if err := r.client.Set(ctx, "session:"+sessionID, data, r.ttl).Err(); err != nil {
		return fmt.Errorf("failed to save session to Redis: %w", err)
	}

	fmt.Printf("Session %s saved to Redis with TTL %v\n", sessionID, r.ttl)
	return nil
}

func (r *RedisRepository) GetSession(ctx context.Context, sessionID string) (*models.MedicalSession, error) {
	data, err := r.client.Get(ctx, "session:"+sessionID).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get session from Redis: %w", err)
	}

	var session models.MedicalSession
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	fmt.Printf("Session %s retrieved from Redis with %d FHR points and %d UC points\n", session.SessionID,
		len(session.Records.FetalHeartRate.Time), len(session.Records.UterineContractions.Time))
	return &session, nil
}

func (r *RedisRepository) DeleteSession(ctx context.Context, sessionID string) error {
	if err := r.client.Del(ctx, "session:"+sessionID).Err(); err != nil {
		return fmt.Errorf("failed to delete session from Redis: %w", err)
	}

	fmt.Printf("Session %s deleted from Redis\n", sessionID)
	return nil
}
