package batch

import (
	"context"

	telemetryv1 "github.com/Krimson/fetal-monitory/proto/telemetry"
)

// Point представляет одну точку данных
type Point struct {
	TsMS  int64   // Временная метка в миллисекундах
	Value float32 // Значение измерения
}

// BatchKey уникально идентифицирует батч по сессии и метрике
type BatchKey struct {
	SessionID string             // Идентификатор сессии
	Metric    telemetryv1.Metric // Тип метрики (FHR или UC)
}

// Batch представляет собранный батч точек
type Batch struct {
	Key    BatchKey // Ключ батча (сессия + метрика)
	T0MS   int64    // Время первой точки в батче
	T1MS   int64    // Время последней точки в батче
	Points []Point  // Точки данных в батче
}

// Sink интерфейс для обработки готовых батчей
type Sink interface {
	Consume(ctx context.Context, b Batch) error
}

// currentBatch - внутренняя структура для отслеживания текущего состояния батча
type currentBatch struct {
	Batch
	lastAddedMS int64 // Время последнего добавления точки
}

// newCurrentBatch создает новый текущий батч
func newCurrentBatch(key BatchKey) *currentBatch {
	return &currentBatch{
		Batch: Batch{
			Key:    key,
			Points: make([]Point, 0),
		},
	}
}

// addPoint добавляет точку в текущий батч и обновляет временные границы
func (cb *currentBatch) addPoint(point Point) {
	if len(cb.Points) == 0 {
		cb.T0MS = point.TsMS
		cb.T1MS = point.TsMS
	} else {
		if point.TsMS < cb.T0MS {
			cb.T0MS = point.TsMS
		}
		if point.TsMS > cb.T1MS {
			cb.T1MS = point.TsMS
		}
	}

	cb.Points = append(cb.Points, point)
	cb.lastAddedMS = point.TsMS
}

// shouldFlushBySize проверяет, нужно ли сбросить батч по размеру
func (cb *currentBatch) shouldFlushBySize(maxSamples int) bool {
	return len(cb.Points) >= maxSamples
}

// shouldFlushBySpan проверяет, нужно ли сбросить батч по временному диапазону
func (cb *currentBatch) shouldFlushBySpan(maxSpanMS int64) bool {
	if len(cb.Points) == 0 {
		return false
	}
	return (cb.T1MS - cb.T0MS) >= maxSpanMS
}

// clone создает копию батча для отправки в sink
func (cb *currentBatch) clone() Batch {
	pointsCopy := make([]Point, len(cb.Points))
	copy(pointsCopy, cb.Points)

	return Batch{
		Key:    cb.Key,
		T0MS:   cb.T0MS,
		T1MS:   cb.T1MS,
		Points: pointsCopy,
	}
}

// reset очищает батч для переиспользования
func (cb *currentBatch) reset() {
	cb.T0MS = 0
	cb.T1MS = 0
	cb.Points = cb.Points[:0] // Сохраняем capacity
	cb.lastAddedMS = 0
}
