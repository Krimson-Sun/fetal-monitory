package models

import (
	"errors"
	"time"
)

type MedicalRecord struct {
	FetalHeartRate      MetricRecord `json:"bpm"`
	UterineContractions MetricRecord `json:"uc"`
}

type MetricRecord struct {
	TimeSec []float64 `json:"time_sec"`
	Value   []float64 `json:"value"`
}

type MedicalSession struct {
	SessionID  string        `json:"session_id"`
	Records    MedicalRecord `json:"records"`
	Prediction float64       `json:"prediction"`
	CreatedAt  time.Time     `json:"created_at"`
	Status     string        `json:"status"`
}

type Acceleration struct {
	StartTime float64 `json:"start"`
	EndTime   float64 `json:"end"`
	Amplitude float64 `json:"amplitude"`
	Duration  float64 `json:"duration"`
}

type Deceleration struct {
	StartTime float64 `json:"start"`
	EndTime   float64 `json:"end"`
	Amplitude float64 `json:"amplitude"`
	Duration  float64 `json:"duration"`
	IsLate    bool    `json:"is_late"`
}

type Contraction struct {
	StartTime float64 `json:"start"`
	EndTime   float64 `json:"end"`
	Amplitude float64 `json:"amplitude"`
	Duration  float64 `json:"duration"`
}

type CTGAnalysis struct {
	STV                   float64        `json:"stv"`
	LTV                   float64        `json:"ltv"`
	BaselineHeartRate     float64        `json:"baseline_heart_rate"`
	Accelerations         []Acceleration `json:"accelerations"`
	Decelerations         []Deceleration `json:"decelerations"`
	Contractions          []Contraction  `json:"contractions"`
	STVs                  []float64      `json:"stvs"`
	STVsWindowDuration    float64        `json:"stvs_window_duration"`
	LTVs                  []float64      `json:"ltvs"`
	LTVsWindowDuration    float64        `json:"ltvs_window_duration"`
	TotalDecelerations    int64          `json:"total_decelerations"`
	LateDecelerations     int64          `json:"late_decelerations"`
	LateDecelerationRatio float64        `json:"late_deceleration_ratio"`
	TotalAccelerations    int64          `json:"total_accelerations"`
	AccelDecelRatio       float64        `json:"accel_decel_ratio"`
	TotalContractions     int64          `json:"total_contractions"`
	STVTrend              float64        `json:"stv_trend"`
	BPMTrend              float64        `json:"bpm_trend"`
	DataPoints            int64          `json:"data_points"`
	TimeSpanSec           float64        `json:"time_span_sec"`
	FilteredBMPBatch      MetricRecord   `json:"filtered_bpm_batch"`
	FilteredUterusBatch   MetricRecord   `json:"filtered_uterus_batch"`
}

type SaveDecision struct {
	SessionID string `json:"session_id"`
	Save      bool   `json:"save"`
}

// Новые структуры для ответов
type UploadResponse struct {
	SessionID  string      `json:"session_id"`
	Status     string      `json:"status"`
	Records    CTGAnalysis `json:"records"`
	Prediction float64     `json:"prediction"`
	Message    string      `json:"message,omitempty"`
}

type DecisionResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// Ошибки
var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session expired")
)
