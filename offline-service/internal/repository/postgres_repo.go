package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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

	// Создаем таблицу если не существует
	createTableSQL := `
    CREATE TABLE IF NOT EXISTS medical_sessions (
        session_id TEXT PRIMARY KEY,
        records JSONB NOT NULL,
        prediction FLOAT NOT NULL,
        created_at TIMESTAMP NOT NULL,
        status TEXT NOT NULL
    );
    
    CREATE INDEX IF NOT EXISTS idx_session_id ON medical_sessions(session_id);
    CREATE INDEX IF NOT EXISTS idx_created_at ON medical_sessions(created_at);
    `

	if _, err := db.Exec(createTableSQL); err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	return &PostgreSQLRepository{db: db}, nil
}

func (r *PostgreSQLRepository) SaveMedicalData(ctx context.Context, session *models.MedicalSession) error {
	recordsJSON, err := json.Marshal(session.Records)
	if err != nil {
		return fmt.Errorf("failed to marshal records: %w", err)
	}

	query := `
    INSERT INTO medical_sessions (session_id, records, prediction, created_at, status)
    VALUES ($1, $2, $3, $4, $5)
    ON CONFLICT (session_id) 
    DO UPDATE SET records = $2, prediction = $3, status = $5
    `

	_, err = r.db.ExecContext(ctx, query,
		session.SessionID,
		recordsJSON,
		session.Prediction,
		session.CreatedAt,
		session.Status,
	)

	if err != nil {
		return fmt.Errorf("failed to insert medical session: %w", err)
	}

	fmt.Printf("Session %s saved to PostgreSQL with %d FHR points and %d UC points\n", session.SessionID,
		len(session.Records.FetalHeartRate.Time), len(session.Records.UterineContractions.Time))
	return nil
}
