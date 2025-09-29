package models

import (
	"encoding/json"
	"errors"
	"time"
)

// Ошибки валидации
var (
	ErrInvalidFHR       = errors.New("invalid FHR value")
	ErrInvalidTOCO      = errors.New("invalid TOCO value")
	ErrInvalidTimestamp = errors.New("invalid timestamp")
)

// DataPoint представляет одну точку данных КТГ
type DataPoint struct {
	Timestamp     time.Time `json:"timestamp"`
	FHR           int       `json:"fhr"`                      // Пульс плода
	TOCO          int       `json:"toco"`                     // Сокращения матки (исправлено T0C0 → TOCO)
	SignalQuality int       `json:"signal_quality,omitempty"` // Качество сигнала
}

// ToJSON преобразует DataPoint в JSON строку
func (dp DataPoint) ToJSON() string {
	data, _ := json.Marshal(dp) // Исправлено Marshall → Marshal
	return string(data)
}

// Validate проверяет корректность данных
func (dp DataPoint) Validate() error {
	if dp.FHR < 50 || dp.FHR > 240 { // Расширенные пределы для безопасности
		return ErrInvalidFHR
	}
	if dp.TOCO < 0 || dp.TOCO > 100 { // Исправлено T0C0 → TOCO
		return ErrInvalidTOCO
	}
	if dp.Timestamp.IsZero() {
		return ErrInvalidTimestamp
	}
	if dp.Timestamp.After(time.Now().Add(1 * time.Minute)) {
		return ErrInvalidTimestamp // Нельзя иметь метку времени из будущего
	}
	return nil
}

// NewDataPoint создает новую точку данных с текущим временем
func NewDataPoint(fhr, toco int) DataPoint {
	return DataPoint{
		Timestamp:     time.Now(),
		FHR:           fhr,
		TOCO:          toco,
		SignalQuality: 100, // По умолчанию отличное качество сигнала
	}
}

// WithSignalQuality устанавливает качество сигнала
func (dp DataPoint) WithSignalQuality(quality int) DataPoint {
	if quality < 0 {
		quality = 0
	}
	if quality > 100 {
		quality = 100
	}
	dp.SignalQuality = quality
	return dp
}

// IsValidFHR проверяет корректность значения FHR
func IsValidFHR(fhr int) bool {
	return fhr >= 50 && fhr <= 240
}

// IsValidTOCO проверяет корректность значения TOCO
func IsValidTOCO(toco int) bool {
	return toco >= 0 && toco <= 100
}

// ParseJSON парсит JSON строку в DataPoint
func ParseJSON(jsonStr string) (DataPoint, error) {
	var dp DataPoint
	err := json.Unmarshal([]byte(jsonStr), &dp)
	if err != nil {
		return DataPoint{}, err
	}
	return dp, dp.Validate()
}
