package session

import (
	"time"

	featureextractorv1 "github.com/Krimson/fetal-monitory/proto/feature_extractor"
)

// SessionStatus представляет статус сессии
type SessionStatus string

const (
	SessionStatusActive  SessionStatus = "ACTIVE"
	SessionStatusStopped SessionStatus = "STOPPED"
	SessionStatusSaved   SessionStatus = "SAVED"
)

// Session представляет мониторинговую сессию
type Session struct {
	ID              string        `json:"id"`
	Status          SessionStatus `json:"status"`
	StartedAt       time.Time     `json:"started_at"`
	StoppedAt       *time.Time    `json:"stopped_at,omitempty"`
	SavedAt         *time.Time    `json:"saved_at,omitempty"`
	TotalDurationMs int64         `json:"total_duration_ms"`
	TotalDataPoints int64         `json:"total_data_points"`
	Metadata        Metadata      `json:"metadata,omitempty"`
}

// Metadata содержит дополнительную информацию о сессии
type Metadata struct {
	PatientID   string                 `json:"patient_id,omitempty"`
	DoctorID    string                 `json:"doctor_id,omitempty"`
	FacilityID  string                 `json:"facility_id,omitempty"`
	Notes       string                 `json:"notes,omitempty"`
	CustomData  map[string]interface{} `json:"custom_data,omitempty"`
	CreatedFrom string                 `json:"created_from,omitempty"` // "web", "mobile", "emulator"
}

// SessionMetrics содержит агрегированные метрики сессии
type SessionMetrics struct {
	SessionID             string    `json:"session_id"`
	STV                   float64   `json:"stv"`
	LTV                   float64   `json:"ltv"`
	BaselineHeartRate     float64   `json:"baseline_heart_rate"`
	TotalAccelerations    int32     `json:"total_accelerations"`
	TotalDecelerations    int32     `json:"total_decelerations"`
	LateDecelerations     int32     `json:"late_decelerations"`
	LateDecelerationRatio float64   `json:"late_deceleration_ratio"`
	TotalContractions     int32     `json:"total_contractions"`
	AccelDecelRatio       float64   `json:"accel_decel_ratio"`
	STVTrend              float64   `json:"stv_trend"`
	BPMTrend              float64   `json:"bpm_trend"`
	DataPoints            int32     `json:"data_points"`
	TimeSpanSec           float64   `json:"time_span_sec"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// EventType представляет тип события
type EventType string

const (
	EventTypeAcceleration EventType = "acceleration"
	EventTypeDeceleration EventType = "deceleration"
	EventTypeContraction  EventType = "contraction"
)

// SessionEvent представляет событие в сессии
type SessionEvent struct {
	ID        int64     `json:"id"`
	SessionID string    `json:"session_id"`
	Type      EventType `json:"type"`
	StartTime float64   `json:"start_time"`
	EndTime   float64   `json:"end_time"`
	Duration  float64   `json:"duration"`
	Amplitude float64   `json:"amplitude"`
	IsLate    bool      `json:"is_late,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// TimeSeriesType представляет тип временного ряда
type TimeSeriesType string

const (
	TimeSeriesTypeSTV TimeSeriesType = "stv"
	TimeSeriesTypeLTV TimeSeriesType = "ltv"
)

// TimeSeriesPoint представляет точку временного ряда
type TimeSeriesPoint struct {
	SessionID      string         `json:"session_id"`
	Type           TimeSeriesType `json:"type"`
	TimeIndex      int            `json:"time_index"`
	Value          float64        `json:"value"`
	WindowDuration float64        `json:"window_duration"`
}

// FilteredDataPoint представляет отфильтрованную точку данных
type FilteredDataPoint struct {
	TimeSec float64 `json:"time_sec"`
	Value   float64 `json:"value"`
}

// MetricType представляет тип метрики
type MetricType string

const (
	MetricTypeBPM    MetricType = "bpm"
	MetricTypeUterus MetricType = "uterus"
)

// SessionData представляет все данные сессии для хранения
type SessionData struct {
	Session            *Session            `json:"session"`
	Metrics            *SessionMetrics     `json:"metrics"`
	Events             []SessionEvent      `json:"events"`
	TimeSeriesSTV      []TimeSeriesPoint   `json:"time_series_stv"`
	TimeSeriesLTV      []TimeSeriesPoint   `json:"time_series_ltv"`
	FilteredBPMData    []FilteredDataPoint `json:"filtered_bpm_data"`
	FilteredUterusData []FilteredDataPoint `json:"filtered_uterus_data"`
}

// CreateSessionRequest представляет запрос на создание сессии
type CreateSessionRequest struct {
	PatientID   string                 `json:"patient_id,omitempty"`
	DoctorID    string                 `json:"doctor_id,omitempty"`
	FacilityID  string                 `json:"facility_id,omitempty"`
	Notes       string                 `json:"notes,omitempty"`
	CustomData  map[string]interface{} `json:"custom_data,omitempty"`
	CreatedFrom string                 `json:"created_from,omitempty"`
}

// SessionResponse представляет ответ с информацией о сессии
type SessionResponse struct {
	Session *Session        `json:"session"`
	Metrics *SessionMetrics `json:"metrics,omitempty"`
}

// SaveSessionRequest представляет запрос на сохранение сессии
type SaveSessionRequest struct {
	Notes string `json:"notes,omitempty"`
}

// ConvertFromFeatureResponse преобразует ответ от feature extractor
func ConvertFromFeatureResponse(response *featureextractorv1.ProcessBatchResponse) *SessionMetrics {
	return &SessionMetrics{
		SessionID:             response.SessionId,
		STV:                   response.Stv,
		LTV:                   response.Ltv,
		BaselineHeartRate:     response.BaselineHeartRate,
		TotalAccelerations:    response.TotalAccelerations,
		TotalDecelerations:    response.TotalDecelerations,
		LateDecelerations:     response.LateDecelerations,
		LateDecelerationRatio: response.LateDecelerationRatio,
		TotalContractions:     response.TotalContractions,
		AccelDecelRatio:       response.AccelDecelRatio,
		STVTrend:              response.StvTrend,
		BPMTrend:              response.BpmTrend,
		DataPoints:            response.DataPoints,
		TimeSpanSec:           response.TimeSpanSec,
		UpdatedAt:             time.Now(),
	}
}

// ConvertAccelerations преобразует акселерации из протобуфа
func ConvertAccelerations(sessionID string, accelerations []*featureextractorv1.Acceleration) []SessionEvent {
	events := make([]SessionEvent, 0, len(accelerations))
	for _, acc := range accelerations {
		events = append(events, SessionEvent{
			SessionID: sessionID,
			Type:      EventTypeAcceleration,
			StartTime: acc.Start,
			EndTime:   acc.End,
			Duration:  acc.Duration,
			Amplitude: acc.Amplitude,
			CreatedAt: time.Now(),
		})
	}
	return events
}

// ConvertDecelerations преобразует децелерации из протобуфа
func ConvertDecelerations(sessionID string, decelerations []*featureextractorv1.Deceleration) []SessionEvent {
	events := make([]SessionEvent, 0, len(decelerations))
	for _, dec := range decelerations {
		events = append(events, SessionEvent{
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
	return events
}

// ConvertContractions преобразует сокращения из протобуфа
func ConvertContractions(sessionID string, contractions []*featureextractorv1.Contraction) []SessionEvent {
	events := make([]SessionEvent, 0, len(contractions))
	for _, cont := range contractions {
		events = append(events, SessionEvent{
			SessionID: sessionID,
			Type:      EventTypeContraction,
			StartTime: cont.Start,
			EndTime:   cont.End,
			Duration:  cont.Duration,
			Amplitude: cont.Amplitude,
			CreatedAt: time.Now(),
		})
	}
	return events
}

// ConvertFilteredData преобразует отфильтрованные данные из протобуфа
func ConvertFilteredData(dataPoints []*featureextractorv1.DataPoint) []FilteredDataPoint {
	points := make([]FilteredDataPoint, 0, len(dataPoints))
	for _, dp := range dataPoints {
		points = append(points, FilteredDataPoint{
			TimeSec: dp.TimeSec,
			Value:   dp.Value,
		})
	}
	return points
}
