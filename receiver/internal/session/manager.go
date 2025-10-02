package session

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	featureextractorv1 "github.com/Krimson/fetal-monitory/proto/feature_extractor"
	"github.com/google/uuid"
)

// Manager управляет сессиями мониторинга (Application Layer)
type Manager struct {
	cache      CacheStore
	repository Repository

	mu             sync.RWMutex
	activeSessions map[string]*Session // Кэш активных сессий в памяти
}

// NewManager создает новый менеджер сессий
func NewManager(cache CacheStore, repository Repository) *Manager {
	return &Manager{
		cache:          cache,
		repository:     repository,
		activeSessions: make(map[string]*Session),
	}
}

// CreateSession создает новую сессию
func (m *Manager) CreateSession(ctx context.Context, req *CreateSessionRequest) (*Session, error) {
	sessionID := uuid.New().String()

	session := &Session{
		ID:        sessionID,
		Status:    SessionStatusActive,
		StartedAt: time.Now(),
		Metadata: Metadata{
			PatientID:   req.PatientID,
			DoctorID:    req.DoctorID,
			FacilityID:  req.FacilityID,
			Notes:       req.Notes,
			CustomData:  req.CustomData,
			CreatedFrom: req.CreatedFrom,
		},
	}

	// Сохраняем в Redis
	if err := m.cache.SetSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to save session to cache: %w", err)
	}

	// Добавляем в активные сессии
	m.mu.Lock()
	m.activeSessions[sessionID] = session
	m.mu.Unlock()

	log.Printf("[SESSION] Created new session: %s", sessionID)
	return session, nil
}

// GetSession получает сессию по ID
func (m *Manager) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	// Сначала проверяем в памяти
	m.mu.RLock()
	if session, ok := m.activeSessions[sessionID]; ok {
		m.mu.RUnlock()
		return session, nil
	}
	m.mu.RUnlock()

	// Проверяем в Redis
	session, err := m.cache.GetSession(ctx, sessionID)
	if err == nil {
		return session, nil
	}

	// Проверяем в PostgreSQL
	return m.repository.GetSession(ctx, sessionID)
}

// StopSession останавливает сессию
func (m *Manager) StopSession(ctx context.Context, sessionID string) error {
	session, err := m.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	if session.Status != SessionStatusActive {
		return fmt.Errorf("session is not active: %s", session.Status)
	}

	now := time.Now()
	session.Status = SessionStatusStopped
	session.StoppedAt = &now
	session.TotalDurationMs = now.Sub(session.StartedAt).Milliseconds()

	// Обновляем в Redis
	if err := m.cache.SetSession(ctx, session); err != nil {
		return fmt.Errorf("failed to update session in cache: %w", err)
	}

	// Удаляем из активных сессий
	m.mu.Lock()
	delete(m.activeSessions, sessionID)
	m.mu.Unlock()

	log.Printf("[SESSION] Stopped session: %s, duration: %dms", sessionID, session.TotalDurationMs)
	return nil
}

// SaveSession сохраняет сессию в PostgreSQL
func (m *Manager) SaveSession(ctx context.Context, sessionID string, notes string) error {
	// Получаем все данные из Redis
	sessionData, err := m.cache.GetSessionData(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session data from cache: %w", err)
	}

	// Обновляем метаданные
	if notes != "" {
		sessionData.Session.Metadata.Notes = notes
	}

	now := time.Now()
	sessionData.Session.Status = SessionStatusSaved
	sessionData.Session.SavedAt = &now

	// Сохраняем в PostgreSQL
	if err := m.repository.SaveSessionData(ctx, sessionData); err != nil {
		return fmt.Errorf("failed to save session to database: %w", err)
	}

	// Обновляем статус в Redis
	if err := m.cache.SetSession(ctx, sessionData.Session); err != nil {
		log.Printf("[WARN] Failed to update session status in cache: %v", err)
	}

	// Опционально: удаляем из Redis (можно оставить с TTL)
	// if err := m.cache.DeleteSession(ctx, sessionID); err != nil {
	// 	log.Printf("[WARN] Failed to delete session from cache: %v", err)
	// }

	log.Printf("[SESSION] Saved session to database: %s", sessionID)
	return nil
}

// ListSessions возвращает список сессий
func (m *Manager) ListSessions(ctx context.Context, limit, offset int) ([]*Session, error) {
	return m.repository.ListSessions(ctx, limit, offset)
}

// DeleteSession удаляет сессию
func (m *Manager) DeleteSession(ctx context.Context, sessionID string) error {
	// Удаляем из памяти
	m.mu.Lock()
	delete(m.activeSessions, sessionID)
	m.mu.Unlock()

	// Удаляем из Redis
	if err := m.cache.DeleteSession(ctx, sessionID); err != nil {
		log.Printf("[WARN] Failed to delete session from cache: %v", err)
	}

	// Удаляем из PostgreSQL
	if err := m.repository.DeleteSession(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to delete session from database: %w", err)
	}

	log.Printf("[SESSION] Deleted session: %s", sessionID)
	return nil
}

// ProcessFeatureBatch обрабатывает батч от feature extractor
func (m *Manager) ProcessFeatureBatch(ctx context.Context, response *featureextractorv1.ProcessBatchResponse) error {
	sessionID := response.SessionId

	// Получаем или создаем сессию
	session, err := m.getOrCreateSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get or create session: %w", err)
	}

	if session.Status != SessionStatusActive {
		log.Printf("[WARN] Received batch for non-active session: %s (status: %s)", sessionID, session.Status)
		return nil // Не возвращаем ошибку, просто игнорируем
	}

	// 1. Обновляем агрегированные метрики
	metrics := ConvertFromFeatureResponse(response)
	if err := m.cache.SetMetrics(ctx, metrics); err != nil {
		return fmt.Errorf("failed to save metrics: %w", err)
	}

	// 2. Добавляем новые события (только если они еще не существуют)
	if err := m.processEvents(ctx, sessionID, response); err != nil {
		log.Printf("[WARN] Failed to process events: %v", err)
	}

	// 3. Добавляем новые значения временных рядов
	if err := m.processTimeSeries(ctx, sessionID, response); err != nil {
		log.Printf("[WARN] Failed to process time series: %v", err)
	}

	// 4. Обновляем отфильтрованные данные
	if err := m.processFilteredData(ctx, sessionID, response); err != nil {
		log.Printf("[WARN] Failed to process filtered data: %v", err)
	}

	// 5. Обновляем счетчик точек данных в сессии
	session.TotalDataPoints += int64(response.DataPoints)
	if err := m.cache.SetSession(ctx, session); err != nil {
		log.Printf("[WARN] Failed to update session: %v", err)
	}

	log.Printf("[SESSION] Processed batch for session %s: stv=%.2f ltv=%.2f points=%d",
		sessionID, response.Stv, response.Ltv, response.DataPoints)

	return nil
}

// processEvents обрабатывает события из батча
func (m *Manager) processEvents(ctx context.Context, sessionID string, response *featureextractorv1.ProcessBatchResponse) error {
	var newEvents []SessionEvent

	// Обрабатываем акселерации
	for _, acc := range response.Accelerations {
		exists, err := m.cache.EventExists(ctx, sessionID, EventTypeAcceleration, acc.Start)
		if err != nil || exists {
			continue
		}
		newEvents = append(newEvents, SessionEvent{
			SessionID: sessionID,
			Type:      EventTypeAcceleration,
			StartTime: acc.Start,
			EndTime:   acc.End,
			Duration:  acc.Duration,
			Amplitude: acc.Amplitude,
			CreatedAt: time.Now(),
		})
	}

	// Обрабатываем децелерации
	for _, dec := range response.Decelerations {
		exists, err := m.cache.EventExists(ctx, sessionID, EventTypeDeceleration, dec.Start)
		if err != nil || exists {
			continue
		}
		newEvents = append(newEvents, SessionEvent{
			SessionID: sessionID,
			Type:      EventTypeDeceleration,
			StartTime: dec.Start,
			EndTime:   dec.End,
			Duration:  dec.Duration,
			Amplitude: dec.Amplitude,
			IsLate:    dec.IsLate,
			CreatedAt: time.Now(),
		})
	}

	// Обрабатываем сокращения
	for _, cont := range response.Contractions {
		exists, err := m.cache.EventExists(ctx, sessionID, EventTypeContraction, cont.Start)
		if err != nil || exists {
			continue
		}
		newEvents = append(newEvents, SessionEvent{
			SessionID: sessionID,
			Type:      EventTypeContraction,
			StartTime: cont.Start,
			EndTime:   cont.End,
			Duration:  cont.Duration,
			Amplitude: cont.Amplitude,
			CreatedAt: time.Now(),
		})
	}

	if len(newEvents) > 0 {
		if err := m.cache.AppendEvents(ctx, sessionID, newEvents); err != nil {
			return err
		}
		log.Printf("[SESSION] Added %d new events to session %s", len(newEvents), sessionID)
	}

	return nil
}

// processTimeSeries обрабатывает временные ряды из батча
func (m *Manager) processTimeSeries(ctx context.Context, sessionID string, response *featureextractorv1.ProcessBatchResponse) error {
	// Обрабатываем STV
	currentSTVCount, err := m.cache.GetTimeSeriesCount(ctx, sessionID, TimeSeriesTypeSTV)
	if err != nil {
		currentSTVCount = 0
	}

	if len(response.Stvs) > currentSTVCount {
		newSTVs := response.Stvs[currentSTVCount:]
		var stvPoints []TimeSeriesPoint
		for i, value := range newSTVs {
			stvPoints = append(stvPoints, TimeSeriesPoint{
				SessionID:      sessionID,
				Type:           TimeSeriesTypeSTV,
				TimeIndex:      currentSTVCount + i,
				Value:          value,
				WindowDuration: response.StvsWindowDuration,
			})
		}
		if err := m.cache.AppendTimeSeries(ctx, sessionID, TimeSeriesTypeSTV, stvPoints); err != nil {
			return err
		}
	}

	// Обрабатываем LTV
	currentLTVCount, err := m.cache.GetTimeSeriesCount(ctx, sessionID, TimeSeriesTypeLTV)
	if err != nil {
		currentLTVCount = 0
	}

	if len(response.Ltvs) > currentLTVCount {
		newLTVs := response.Ltvs[currentLTVCount:]
		var ltvPoints []TimeSeriesPoint
		for i, value := range newLTVs {
			ltvPoints = append(ltvPoints, TimeSeriesPoint{
				SessionID:      sessionID,
				Type:           TimeSeriesTypeLTV,
				TimeIndex:      currentLTVCount + i,
				Value:          value,
				WindowDuration: response.LtvsWindowDuration,
			})
		}
		if err := m.cache.AppendTimeSeries(ctx, sessionID, TimeSeriesTypeLTV, ltvPoints); err != nil {
			return err
		}
	}

	return nil
}

// processFilteredData обрабатывает отфильтрованные данные из батча
func (m *Manager) processFilteredData(ctx context.Context, sessionID string, response *featureextractorv1.ProcessBatchResponse) error {
	// Обновляем BPM данные
	if len(response.FilteredBpmBatch) > 0 {
		bpmPoints := ConvertFilteredData(response.FilteredBpmBatch)
		if err := m.cache.UpdateFilteredData(ctx, sessionID, MetricTypeBPM, bpmPoints); err != nil {
			return fmt.Errorf("failed to update BPM data: %w", err)
		}
	}

	// Обновляем Uterus данные
	if len(response.FilteredUterusBatch) > 0 {
		uterusPoints := ConvertFilteredData(response.FilteredUterusBatch)
		if err := m.cache.UpdateFilteredData(ctx, sessionID, MetricTypeUterus, uterusPoints); err != nil {
			return fmt.Errorf("failed to update Uterus data: %w", err)
		}
	}

	return nil
}

// GetSessionMetrics получает текущие метрики сессии
func (m *Manager) GetSessionMetrics(ctx context.Context, sessionID string) (*SessionMetrics, error) {
	return m.cache.GetMetrics(ctx, sessionID)
}

// GetSessionData получает все данные сессии
func (m *Manager) GetSessionData(ctx context.Context, sessionID string) (*SessionData, error) {
	return m.cache.GetSessionData(ctx, sessionID)
}

// IsSessionActive проверяет, активна ли сессия
func (m *Manager) IsSessionActive(sessionID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.activeSessions[sessionID]
	return exists
}

// getOrCreateSession получает существующую сессию или создает новую
// Используется для автоматического создания сессий при получении данных от устройств
func (m *Manager) getOrCreateSession(ctx context.Context, sessionID string) (*Session, error) {
	// Сначала проверяем в памяти (быстро)
	m.mu.RLock()
	if session, exists := m.activeSessions[sessionID]; exists {
		m.mu.RUnlock()
		return session, nil
	}
	m.mu.RUnlock()

	// Проверяем в кэше (Redis)
	session, err := m.cache.GetSession(ctx, sessionID)
	if err == nil {
		// Найдена в Redis, добавляем в память
		m.mu.Lock()
		m.activeSessions[sessionID] = session
		m.mu.Unlock()
		return session, nil
	}

	// Проверяем в PostgreSQL (возможно, остановленная сессия)
	session, err = m.repository.GetSession(ctx, sessionID)
	if err == nil {
		// Найдена в БД, загружаем в кэш
		log.Printf("[SESSION] Loaded existing session from database: %s (status: %s)", sessionID, session.Status)
		if err := m.cache.SetSession(ctx, session); err != nil {
			log.Printf("[WARN] Failed to cache session: %v", err)
		}
		if session.Status == SessionStatusActive {
			m.mu.Lock()
			m.activeSessions[sessionID] = session
			m.mu.Unlock()
		}
		return session, nil
	}

	// Сессия не найдена нигде - создаем новую
	log.Printf("[SESSION] Auto-creating new session from incoming data: %s", sessionID)

	session = &Session{
		ID:        sessionID,
		Status:    SessionStatusActive,
		StartedAt: time.Now(),
		Metadata: Metadata{
			CreatedFrom: "auto-created",
			Notes:       "Automatically created from device/emulator data",
		},
	}

	// Сохраняем в Redis
	if err := m.cache.SetSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to save auto-created session to cache: %w", err)
	}

	// Добавляем в активные сессии
	m.mu.Lock()
	m.activeSessions[sessionID] = session
	m.mu.Unlock()

	log.Printf("[SESSION] Successfully auto-created session: %s", sessionID)
	return session, nil
}
