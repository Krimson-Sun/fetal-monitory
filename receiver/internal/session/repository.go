package session

import (
	"context"
)

// Repository определяет интерфейс для работы с хранилищем сессий (Domain Layer)
type Repository interface {
	// Управление сессиями
	CreateSession(ctx context.Context, session *Session) error
	GetSession(ctx context.Context, sessionID string) (*Session, error)
	UpdateSession(ctx context.Context, session *Session) error
	ListSessions(ctx context.Context, limit, offset int) ([]*Session, error)
	DeleteSession(ctx context.Context, sessionID string) error

	// Работа с метриками
	SaveMetrics(ctx context.Context, metrics *SessionMetrics) error
	GetMetrics(ctx context.Context, sessionID string) (*SessionMetrics, error)

	// Работа с событиями
	SaveEvents(ctx context.Context, events []SessionEvent) error
	GetEvents(ctx context.Context, sessionID string) ([]SessionEvent, error)

	// Работа с временными рядами
	SaveTimeSeries(ctx context.Context, points []TimeSeriesPoint) error
	GetTimeSeries(ctx context.Context, sessionID string, seriesType TimeSeriesType) ([]TimeSeriesPoint, error)

	// Сохранение полных данных сессии
	SaveSessionData(ctx context.Context, data *SessionData) error
}

// CacheStore определяет интерфейс для работы с кэшем (Redis)
type CacheStore interface {
	// Управление сессиями в кэше
	SetSession(ctx context.Context, session *Session) error
	GetSession(ctx context.Context, sessionID string) (*Session, error)
	DeleteSession(ctx context.Context, sessionID string) error

	// Метрики (перезаписываются целиком)
	SetMetrics(ctx context.Context, metrics *SessionMetrics) error
	GetMetrics(ctx context.Context, sessionID string) (*SessionMetrics, error)

	// События (append-only)
	AppendEvents(ctx context.Context, sessionID string, events []SessionEvent) error
	GetEvents(ctx context.Context, sessionID string, eventType EventType) ([]SessionEvent, error)
	GetAllEvents(ctx context.Context, sessionID string) ([]SessionEvent, error)
	EventExists(ctx context.Context, sessionID string, eventType EventType, startTime float64) (bool, error)

	// Временные ряды (append-only)
	AppendTimeSeries(ctx context.Context, sessionID string, seriesType TimeSeriesType, points []TimeSeriesPoint) error
	GetTimeSeries(ctx context.Context, sessionID string, seriesType TimeSeriesType) ([]TimeSeriesPoint, error)
	GetTimeSeriesCount(ctx context.Context, sessionID string, seriesType TimeSeriesType) (int, error)

	// Отфильтрованные данные (обновляются через Sorted Set)
	UpdateFilteredData(ctx context.Context, sessionID string, metricType MetricType, points []FilteredDataPoint) error
	GetFilteredData(ctx context.Context, sessionID string, metricType MetricType) ([]FilteredDataPoint, error)
	GetFilteredDataCount(ctx context.Context, sessionID string, metricType MetricType) (int, error)

	// Получение всех данных сессии
	GetSessionData(ctx context.Context, sessionID string) (*SessionData, error)

	// Утилиты
	SessionExists(ctx context.Context, sessionID string) (bool, error)
	SetSessionTTL(ctx context.Context, sessionID string, ttl int) error
}
