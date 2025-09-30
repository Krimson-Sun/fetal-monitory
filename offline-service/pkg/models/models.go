package models

import (
	"time"
)

type MedicalRecord struct {
	FetalHeartRate      MetricRecord `json:"bpm"`
	UterineContractions MetricRecord `json:"uc"`
}

type MetricRecord struct {
	Time  []float64 `json:"time"`
	Value []float64 `json:"value"`
}

type MedicalSession struct {
	SessionID  string        `json:"session_id"`
	Records    MedicalRecord `json:"records"`
	Prediction float64       `json:"prediction"`
	CreatedAt  time.Time     `json:"created_at"`
	Status     string        `json:"status"`
}

type SaveDecision struct {
	SessionID string `json:"session_id"`
	Save      bool   `json:"save"`
}

// Новые структуры для ответов
type UploadResponse struct {
	SessionID  string        `json:"session_id"`
	Status     string        `json:"status"`
	Records    MedicalRecord `json:"records"`
	Prediction float64       `json:"prediction"`
	Message    string        `json:"message,omitempty"`
}

type DecisionResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}
