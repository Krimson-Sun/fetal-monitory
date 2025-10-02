package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"offline-service/pkg/models"
)

// PostgresStub - заглушка PostgreSQL репозитория
type PostgresStub struct {
	sessions map[string]*models.MedicalSession
	mutex    sync.RWMutex
}

func NewPostgresStub() *PostgresStub {
	return &PostgresStub{
		sessions: make(map[string]*models.MedicalSession),
	}
}

func (p *PostgresStub) SaveMedicalRecord(ctx context.Context, session *models.MedicalSession) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Создаем копию сессии для хранения
	sessionCopy := *session
	sessionCopy.Status = "saved"

	p.sessions[session.SessionID] = &sessionCopy

	// Логируем данные для отладки
	recordsJSON, _ := json.Marshal(session.Records)
	fmt.Printf("✅ PostgresStub: Session %s saved to database\n", session.SessionID)
	fmt.Printf("   Records: %d FHR points, %d UC points, Prediction: %f\n", len(session.Records.FetalHeartRate.TimeSec),
		len(session.Records.UterineContractions.TimeSec), session.Prediction)
	fmt.Printf("   First 3 records: %s\n", string(recordsJSON)[:min(100, len(string(recordsJSON)))])

	return nil
}

func (p *PostgresStub) GetMedicalData(ctx context.Context, sessionID string) (*models.MedicalSession, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	session, exists := p.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found in database: %s", sessionID)
	}

	fmt.Printf("✅ PostgresStub: Session %s retrieved from database\n", sessionID)
	return session, nil
}

func (p *PostgresStub) Close() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	fmt.Printf("✅ PostgresStub: Closing connection, saved sessions: %d\n", len(p.sessions))
	return nil
}

// Дополнительные методы для отладки и тестирования

// GetAllSessions возвращает все сохраненные сессии
func (p *PostgresStub) GetAllSessions() []*models.MedicalSession {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	var sessions []*models.MedicalSession
	for _, session := range p.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}

// GetStats возвращает статистику по сохраненным данным
func (p *PostgresStub) GetStats() map[string]interface{} {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	totalRecords := 0
	for _, session := range p.sessions {
		totalRecords += len(session.Records.FetalHeartRate.TimeSec) + len(session.Records.UterineContractions.TimeSec)
	}

	return map[string]interface{}{
		"total_sessions": len(p.sessions),
		"total_records":  totalRecords,
		"session_ids":    p.getSessionKeys(),
	}
}

// Clear очищает все данные (для тестирования)
func (p *PostgresStub) Clear() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.sessions = make(map[string]*models.MedicalSession)
	fmt.Println("✅ PostgresStub: All data cleared")
}

func (p *PostgresStub) getSessionKeys() []string {
	var keys []string
	for key := range p.sessions {
		keys = append(keys, key)
	}
	return keys
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
