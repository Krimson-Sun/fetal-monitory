package utils

import (
	"errors"
	"fetal-emulator/internal/models"
	"time"
)

// Ошибки валидации
var (
	ErrInvalidFHRValue  = errors.New("invalid FHR value")
	ErrInvalidTOCOValue = errors.New("invalid TOCO value")
	ErrInvalidTimestamp = errors.New("invalid timestamp")
	ErrFutureTimestamp  = errors.New("timestamp is in the future")
)

// ValidateDataPoint проверяет корректность точки данных
func ValidateDataPoint(dp models.DataPoint) error {
	if !ValidateRange(dp.FHR, 50, 240) { // Расширенные пределы для безопасности
		return ErrInvalidFHRValue
	}
	if !ValidateRange(dp.TOCO, 0, 100) {
		return ErrInvalidTOCOValue
	}
	if dp.Timestamp.IsZero() {
		return ErrInvalidTimestamp
	}
	if dp.Timestamp.After(time.Now().Add(1 * time.Minute)) {
		return ErrFutureTimestamp
	}
	return nil
}

// ValidateFHRValue проверяет значение пульса плода
func ValidateFHRValue(fhr int) error {
	if fhr < 50 || fhr > 240 {
		return ErrInvalidFHRValue
	}
	return nil
}

// ValidateTOCOValue проверяет значение маточных сокращений
func ValidateTOCOValue(toco int) error {
	if toco < 0 || toco > 100 {
		return ErrInvalidTOCOValue
	}
	return nil
}

// ValidateRange проверяет что значение в диапазоне [min, max]
func ValidateRange(value, min, max int) bool {
	return value >= min && value <= max
}

// ValidateTimestamp проверяет корректность метки времени
func ValidateTimestamp(timestamp time.Time) error {
	if timestamp.IsZero() {
		return ErrInvalidTimestamp
	}
	if timestamp.After(time.Now().Add(1 * time.Minute)) {
		return ErrFutureTimestamp
	}
	if timestamp.Before(time.Now().Add(-24 * time.Hour)) {
		return errors.New("timestamp is too far in the past")
	}
	return nil
}

// ValidateSignalQuality проверяет качество сигнала (0-100)
func ValidateSignalQuality(quality int) error {
	if quality < 0 || quality > 100 {
		return errors.New("signal quality must be between 0 and 100")
	}
	return nil
}
