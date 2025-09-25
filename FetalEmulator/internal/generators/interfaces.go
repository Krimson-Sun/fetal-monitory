package generators

import (
	"errors"
	"sync"
)

// Ошибки генераторов
var (
	ErrInvalidFHRValue   = errors.New("invalid FHR value")
	ErrInvalidTOCOValue  = errors.New("invalid TOCO value")
	ErrGeneratorNotReady = errors.New("generator not ready")
	ErrInvalidConfig     = errors.New("invalid generator configuration")
)

// DataGenerator базовый интерфейс для всех генераторов данных
type DataGenerator interface {
	// NextValue возвращает следующее сгенерированное значение
	NextValue() int

	// Validate проверяет корректность работы генератора
	Validate() error

	// Reset сбрасывает состояние генератора
	Reset()

	// Seed устанавливает seed для случайного генератора
	Seed(seed int64)
}

// FHRGenerator интерфейс для генератора пульса плода
type FHRGenerator interface {
	DataGenerator

	// SetBaseValue устанавливает базовое значение пульса
	SetBaseValue(value int)

	// GetVariability возвращает текущую вариабельность
	GetVariability() int

	// SetVariability устанавливает вариабельность
	SetVariability(variability int)

	// GetStats возвращает статистику работы генератора
	GetStats() GeneratorStats
}

// TOCOGenerator интерфейс для генератора маточных сокращений
type TOCOGenerator interface {
	DataGenerator

	// GetContractionStatus возвращает true если идет схватка
	GetContractionStatus() bool

	// GetContractionProgress возвращает прогресс текущей схватки (0-100)
	GetContractionProgress() int

	// ForceContraction принудительно запускает схватку (для тестирования)
	ForceContraction()
}

// GeneratorStats содержит статистику генератора
type GeneratorStats struct {
	TotalValuesGenerated int
	MinValueGenerated    int
	MaxValueGenerated    int
	AverageValue         float64
	LastValue            int
	ErrorsCount          int
	mu                   sync.RWMutex
}
