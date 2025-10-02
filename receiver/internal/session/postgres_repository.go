package session

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// PostgresRepository реализует Repository для PostgreSQL (Infrastructure Layer)
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository создает новый экземпляр PostgresRepository
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{
		db: db,
	}
}

// NewPostgresRepositoryFromDSN создает репозиторий из строки подключения
func NewPostgresRepositoryFromDSN(dsn string) (*PostgresRepository, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Проверяем соединение
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Настройки пула соединений
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return &PostgresRepository{db: db}, nil
}

// Close закрывает соединение с БД
func (r *PostgresRepository) Close() error {
	return r.db.Close()
}

// ===== Управление сессиями =====

func (r *PostgresRepository) CreateSession(ctx context.Context, session *Session) error {
	metadataJSON, err := json.Marshal(session.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO sessions (id, status, started_at, stopped_at, saved_at, total_duration_ms, total_data_points, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err = r.db.ExecContext(ctx, query,
		session.ID,
		session.Status,
		session.StartedAt,
		session.StoppedAt,
		session.SavedAt,
		session.TotalDurationMs,
		session.TotalDataPoints,
		metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	query := `
		SELECT id, status, started_at, stopped_at, saved_at, total_duration_ms, total_data_points, metadata
		FROM sessions
		WHERE id = $1
	`

	var session Session
	var metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, sessionID).Scan(
		&session.ID,
		&session.Status,
		&session.StartedAt,
		&session.StoppedAt,
		&session.SavedAt,
		&session.TotalDurationMs,
		&session.TotalDataPoints,
		&metadataJSON,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found: %s", sessionID)
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if err := json.Unmarshal(metadataJSON, &session.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &session, nil
}

func (r *PostgresRepository) UpdateSession(ctx context.Context, session *Session) error {
	metadataJSON, err := json.Marshal(session.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		UPDATE sessions
		SET status = $2, stopped_at = $3, saved_at = $4, total_duration_ms = $5, total_data_points = $6, metadata = $7
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		session.ID,
		session.Status,
		session.StoppedAt,
		session.SavedAt,
		session.TotalDurationMs,
		session.TotalDataPoints,
		metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("session not found: %s", session.ID)
	}

	return nil
}

func (r *PostgresRepository) ListSessions(ctx context.Context, limit, offset int) ([]*Session, error) {
	query := `
		SELECT id, status, started_at, stopped_at, saved_at, total_duration_ms, total_data_points, metadata
		FROM sessions
		ORDER BY started_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*Session

	for rows.Next() {
		var session Session
		var metadataJSON []byte

		err := rows.Scan(
			&session.ID,
			&session.Status,
			&session.StartedAt,
			&session.StoppedAt,
			&session.SavedAt,
			&session.TotalDurationMs,
			&session.TotalDataPoints,
			&metadataJSON,
		)

		if err != nil {
			continue // Пропускаем поврежденные записи
		}

		if err := json.Unmarshal(metadataJSON, &session.Metadata); err == nil {
			sessions = append(sessions, &session)
		}
	}

	return sessions, nil
}

func (r *PostgresRepository) DeleteSession(ctx context.Context, sessionID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Удаляем связанные данные (каскадное удаление должно работать через FK, но для надежности делаем явно)
	queries := []string{
		"DELETE FROM session_raw_data WHERE session_id = $1",
		"DELETE FROM session_timeseries WHERE session_id = $1",
		"DELETE FROM session_events WHERE session_id = $1",
		"DELETE FROM session_metrics WHERE session_id = $1",
		"DELETE FROM sessions WHERE id = $1",
	}

	for _, query := range queries {
		if _, err := tx.ExecContext(ctx, query, sessionID); err != nil {
			return fmt.Errorf("failed to delete session data: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ===== Метрики =====

func (r *PostgresRepository) SaveMetrics(ctx context.Context, metrics *SessionMetrics) error {
	query := `
		INSERT INTO session_metrics (
			session_id, stv, ltv, baseline_heart_rate,
			total_accelerations, total_decelerations, late_decelerations, late_deceleration_ratio,
			total_contractions, accel_decel_ratio, stv_trend, bpm_trend,
			data_points, time_span_sec, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (session_id) DO UPDATE SET
			stv = EXCLUDED.stv,
			ltv = EXCLUDED.ltv,
			baseline_heart_rate = EXCLUDED.baseline_heart_rate,
			total_accelerations = EXCLUDED.total_accelerations,
			total_decelerations = EXCLUDED.total_decelerations,
			late_decelerations = EXCLUDED.late_decelerations,
			late_deceleration_ratio = EXCLUDED.late_deceleration_ratio,
			total_contractions = EXCLUDED.total_contractions,
			accel_decel_ratio = EXCLUDED.accel_decel_ratio,
			stv_trend = EXCLUDED.stv_trend,
			bpm_trend = EXCLUDED.bpm_trend,
			data_points = EXCLUDED.data_points,
			time_span_sec = EXCLUDED.time_span_sec,
			updated_at = EXCLUDED.updated_at
	`

	_, err := r.db.ExecContext(ctx, query,
		metrics.SessionID,
		metrics.STV,
		metrics.LTV,
		metrics.BaselineHeartRate,
		metrics.TotalAccelerations,
		metrics.TotalDecelerations,
		metrics.LateDecelerations,
		metrics.LateDecelerationRatio,
		metrics.TotalContractions,
		metrics.AccelDecelRatio,
		metrics.STVTrend,
		metrics.BPMTrend,
		metrics.DataPoints,
		metrics.TimeSpanSec,
		metrics.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save metrics: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetMetrics(ctx context.Context, sessionID string) (*SessionMetrics, error) {
	query := `
		SELECT session_id, stv, ltv, baseline_heart_rate,
			total_accelerations, total_decelerations, late_decelerations, late_deceleration_ratio,
			total_contractions, accel_decel_ratio, stv_trend, bpm_trend,
			data_points, time_span_sec, updated_at
		FROM session_metrics
		WHERE session_id = $1
	`

	var metrics SessionMetrics

	err := r.db.QueryRowContext(ctx, query, sessionID).Scan(
		&metrics.SessionID,
		&metrics.STV,
		&metrics.LTV,
		&metrics.BaselineHeartRate,
		&metrics.TotalAccelerations,
		&metrics.TotalDecelerations,
		&metrics.LateDecelerations,
		&metrics.LateDecelerationRatio,
		&metrics.TotalContractions,
		&metrics.AccelDecelRatio,
		&metrics.STVTrend,
		&metrics.BPMTrend,
		&metrics.DataPoints,
		&metrics.TimeSpanSec,
		&metrics.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("metrics not found for session: %s", sessionID)
		}
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	return &metrics, nil
}

// ===== События =====

func (r *PostgresRepository) SaveEvents(ctx context.Context, events []SessionEvent) error {
	if len(events) == 0 {
		return nil
	}

	query := `
		INSERT INTO session_events (session_id, event_type, start_time, end_time, duration, amplitude, is_late, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, event := range events {
		_, err := stmt.ExecContext(ctx,
			event.SessionID,
			event.Type,
			event.StartTime,
			event.EndTime,
			event.Duration,
			event.Amplitude,
			event.IsLate,
			event.CreatedAt,
		)

		if err != nil {
			return fmt.Errorf("failed to insert event: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetEvents(ctx context.Context, sessionID string) ([]SessionEvent, error) {
	query := `
		SELECT id, session_id, event_type, start_time, end_time, duration, amplitude, is_late, created_at
		FROM session_events
		WHERE session_id = $1
		ORDER BY start_time ASC
	`

	rows, err := r.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}
	defer rows.Close()

	var events []SessionEvent

	for rows.Next() {
		var event SessionEvent

		err := rows.Scan(
			&event.ID,
			&event.SessionID,
			&event.Type,
			&event.StartTime,
			&event.EndTime,
			&event.Duration,
			&event.Amplitude,
			&event.IsLate,
			&event.CreatedAt,
		)

		if err != nil {
			continue
		}

		events = append(events, event)
	}

	return events, nil
}

// ===== Временные ряды =====

func (r *PostgresRepository) SaveTimeSeries(ctx context.Context, points []TimeSeriesPoint) error {
	if len(points) == 0 {
		return nil
	}

	query := `
		INSERT INTO session_timeseries (session_id, metric_type, time_index, value, window_duration)
		VALUES ($1, $2, $3, $4, $5)
	`

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, point := range points {
		_, err := stmt.ExecContext(ctx,
			point.SessionID,
			point.Type,
			point.TimeIndex,
			point.Value,
			point.WindowDuration,
		)

		if err != nil {
			return fmt.Errorf("failed to insert time series point: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetTimeSeries(ctx context.Context, sessionID string, seriesType TimeSeriesType) ([]TimeSeriesPoint, error) {
	query := `
		SELECT session_id, metric_type, time_index, value, window_duration
		FROM session_timeseries
		WHERE session_id = $1 AND metric_type = $2
		ORDER BY time_index ASC
	`

	rows, err := r.db.QueryContext(ctx, query, sessionID, seriesType)
	if err != nil {
		return nil, fmt.Errorf("failed to get time series: %w", err)
	}
	defer rows.Close()

	var points []TimeSeriesPoint

	for rows.Next() {
		var point TimeSeriesPoint

		err := rows.Scan(
			&point.SessionID,
			&point.Type,
			&point.TimeIndex,
			&point.Value,
			&point.WindowDuration,
		)

		if err != nil {
			continue
		}

		points = append(points, point)
	}

	return points, nil
}

// ===== Сохранение полных данных сессии =====

func (r *PostgresRepository) SaveSessionData(ctx context.Context, data *SessionData) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Сохраняем/обновляем сессию
	if err := r.CreateSession(ctx, data.Session); err != nil {
		// Если сессия уже существует, обновляем
		if err := r.UpdateSession(ctx, data.Session); err != nil {
			return fmt.Errorf("failed to save session: %w", err)
		}
	}

	// 2. Сохраняем метрики
	if data.Metrics != nil {
		if err := r.SaveMetrics(ctx, data.Metrics); err != nil {
			return fmt.Errorf("failed to save metrics: %w", err)
		}
	}

	// 3. Сохраняем события
	if len(data.Events) > 0 {
		if err := r.SaveEvents(ctx, data.Events); err != nil {
			return fmt.Errorf("failed to save events: %w", err)
		}
	}

	// 4. Сохраняем временные ряды
	allTimeSeries := append(data.TimeSeriesSTV, data.TimeSeriesLTV...)
	if len(allTimeSeries) > 0 {
		if err := r.SaveTimeSeries(ctx, allTimeSeries); err != nil {
			return fmt.Errorf("failed to save time series: %w", err)
		}
	}

	// 5. Опционально: сохраняем отфильтрованные данные как raw data
	if len(data.FilteredBPMData) > 0 || len(data.FilteredUterusData) > 0 {
		if err := r.saveFilteredDataAsRaw(ctx, data.Session.ID, data.FilteredBPMData, data.FilteredUterusData); err != nil {
			return fmt.Errorf("failed to save filtered data: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// saveFilteredDataAsRaw сохраняет отфильтрованные данные в таблицу raw_data
func (r *PostgresRepository) saveFilteredDataAsRaw(ctx context.Context, sessionID string, bpmData, uterusData []FilteredDataPoint) error {
	query := `
		INSERT INTO session_raw_data (session_id, batch_ts_ms, metric_type, data, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	// Сохраняем BPM данные
	if len(bpmData) > 0 {
		dataJSON, err := json.Marshal(bpmData)
		if err != nil {
			return fmt.Errorf("failed to marshal bpm data: %w", err)
		}

		_, err = r.db.ExecContext(ctx, query,
			sessionID,
			time.Now().UnixMilli(),
			"FHR",
			dataJSON,
			time.Now(),
		)

		if err != nil {
			return fmt.Errorf("failed to save bpm data: %w", err)
		}
	}

	// Сохраняем Uterus данные
	if len(uterusData) > 0 {
		dataJSON, err := json.Marshal(uterusData)
		if err != nil {
			return fmt.Errorf("failed to marshal uterus data: %w", err)
		}

		_, err = r.db.ExecContext(ctx, query,
			sessionID,
			time.Now().UnixMilli(),
			"UC",
			dataJSON,
			time.Now(),
		)

		if err != nil {
			return fmt.Errorf("failed to save uterus data: %w", err)
		}
	}

	return nil
}

