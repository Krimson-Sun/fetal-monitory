package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"offline-service/pkg/models"
)

// RedisStub - заглушка Redis репозитория
type RedisStub struct {
	sessions map[string][]byte
	mutex    sync.RWMutex
	ttl      time.Duration
}

func NewRedisStub(ttl time.Duration) *RedisStub {
	return &RedisStub{
		sessions: make(map[string][]byte),
		ttl:      ttl,
	}
}

func (r *RedisStub) SaveSession(ctx context.Context, sessionID string, session *models.MedicalSession) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	r.sessions["session:"+sessionID] = data

	// Имитация TTL - запускаем горутину для удаления через TTL
	go func() {
		time.Sleep(r.ttl)
		r.mutex.Lock()
		delete(r.sessions, "session:"+sessionID)
		r.mutex.Unlock()
		fmt.Printf("Session %s expired after TTL\n", sessionID)
	}()

	fmt.Printf("✅ RedisStub: Session %s saved (TTL: %v)\n", sessionID, r.ttl)
	fmt.Printf("   Records: %d FHR points, %d UC points, Prediction: %f\n", len(session.Records.FetalHeartRate.Time),
		len(session.Records.UterineContractions.Time), session.Prediction)
	return nil
}

func (r *RedisStub) GetSession(ctx context.Context, sessionID string) (*models.MedicalSession, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	data, exists := r.sessions["session:"+sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	var session models.MedicalSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	fmt.Printf("✅ RedisStub: Session %s retrieved\n", sessionID)
	fmt.Printf("   Records: %d FHR points, %d UC points, Prediction: %f\n", len(session.Records.FetalHeartRate.Time),
		len(session.Records.UterineContractions.Time), session.Prediction)
	return &session, nil
}

func (r *RedisStub) DeleteSession(ctx context.Context, sessionID string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	key := "session:" + sessionID
	if _, exists := r.sessions[key]; !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	delete(r.sessions, key)
	fmt.Printf("✅ RedisStub: Session %s deleted\n", sessionID)
	return nil
}

func (r *RedisStub) Close() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	fmt.Printf("✅ RedisStub: Closing connection, active sessions: %d\n", len(r.sessions))
	r.sessions = make(map[string][]byte)
	return nil
}

// Дополнительный метод для отладки
func (r *RedisStub) GetStats() map[string]interface{} {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return map[string]interface{}{
		"active_sessions": len(r.sessions),
		"session_ids":     r.getSessionKeys(),
	}
}

func (r *RedisStub) getSessionKeys() []string {
	var keys []string
	for key := range r.sessions {
		// Убираем префикс "session:"
		if len(key) > 8 {
			keys = append(keys, key[8:])
		}
	}
	return keys
}
