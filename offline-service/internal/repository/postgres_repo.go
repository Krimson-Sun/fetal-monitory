package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
	"offline-service/pkg/models"
)

type PostgreSQLRepository struct {
	db *sql.DB
}

func NewPostgreSQLRepository(connStr string) (*PostgreSQLRepository, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	// Настройка пула соединений
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	log.Println("✅ Connected to PostgreSQL database")
	return &PostgreSQLRepository{db: db}, nil
}

// SaveSession - сохраняет сессию в вашу реальную схему
func (r *PostgreSQLRepository) SaveSession(ctx context.Context, session *models.MedicalSession) error {
	// Начинаем транзакцию
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Сохраняем в таблицу sessions
	totalDataPoints := len(session.Records.FetalHeartRate.TimeSec) + len(session.Records.UterineContractions.TimeSec)

	_, err = tx.ExecContext(ctx, `
		INSERT INTO sessions (id, status, started_at, total_data_points, metadata)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) 
		DO UPDATE SET 
			status = $2,
			total_data_points = $4,
			metadata = $5`,
		session.SessionID,
		session.Status,
		session.CreatedAt,
		totalDataPoints,
		`{"source": "offline-service", "prediction": `+fmt.Sprintf("%.4f", session.Prediction)+`}`,
	)
	if err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	// 2. Сохраняем сырые данные в session_raw_data
	if err := r.saveRawData(ctx, tx, session); err != nil {
		return err
	}

	log.Printf("✅ Session %s saved to PostgreSQL", session.SessionID)

	// Коммитим транзакцию
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *PostgreSQLRepository) saveRawData(ctx context.Context, tx *sql.Tx, session *models.MedicalSession) error {
	// Удаляем старые сырые данные
	_, err := tx.ExecContext(ctx, "DELETE FROM session_raw_data WHERE session_id = $1", session.SessionID)
	if err != nil {
		return fmt.Errorf("failed to delete old raw data: %w", err)
	}

	// Сохраняем BPM данные
	bpmData, err := json.Marshal(struct {
		TimeSec []float64 `json:"time_sec"`
		Value   []float64 `json:"value"`
	}{
		TimeSec: session.Records.FetalHeartRate.TimeSec,
		Value:   session.Records.FetalHeartRate.Value,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal BPM data: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO session_raw_data (session_id, batch_ts_ms, metric_type, data)
		VALUES ($1, $2, 'FHR', $3)`,
		session.SessionID,
		time.Now().UnixMilli(),
		bpmData,
	)
	if err != nil {
		return fmt.Errorf("failed to insert BPM raw data: %w", err)
	}

	// Сохраняем UC данные
	ucData, err := json.Marshal(struct {
		TimeSec []float64 `json:"time_sec"`
		Value   []float64 `json:"value"`
	}{
		TimeSec: session.Records.UterineContractions.TimeSec,
		Value:   session.Records.UterineContractions.Value,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal UC data: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO session_raw_data (session_id, batch_ts_ms, metric_type, data)
		VALUES ($1, $2, 'UC', $3)`,
		session.SessionID,
		time.Now().UnixMilli()+1,
		ucData,
	)
	if err != nil {
		return fmt.Errorf("failed to insert UC raw data: %w", err)
	}

	return nil
}

// CheckConnection - проверяет подключение
func (r *PostgreSQLRepository) CheckConnection(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

// Close - закрывает соединение
func (r *PostgreSQLRepository) Close() error {
	return r.db.Close()
}

// GetStats - получает статистику (только количество сессий)
func (r *PostgreSQLRepository) GetStats() map[string]interface{} {
	var sessionCount int
	err := r.db.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&sessionCount)

	if err != nil {
		return map[string]interface{}{
			"active_sessions": 0,
			"session_ids":     []string{},
			"error":           err.Error(),
		}
	}

	return map[string]interface{}{
		"active_sessions": sessionCount,
		"session_ids":     []string{}, // Пока не реализовано получение ID
	}
}

// Заглушки для остальных методов (чтобы удовлетворять интерфейсу)

func (r *PostgreSQLRepository) GetSession(ctx context.Context, sessionID string) (*models.MedicalSession, error) {
	// Временная заглушка
	return nil, fmt.Errorf("GetSession not implemented yet")
}

func (r *PostgreSQLRepository) DeleteSession(ctx context.Context, sessionID string) error {
	// Временная заглушка
	return fmt.Errorf("DeleteSession not implemented yet")
}
