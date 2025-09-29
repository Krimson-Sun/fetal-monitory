package batch

import (
	"context"
	"sync"
	"testing"
	"time"

	telemetryv1 "github.com/Krimson/fetal-monitory/proto/telemetry"
	"github.com/Krimson/fetal-monitory/receiver/internal/config"
)

// TestSink для тестирования - собирает все батчи
type TestSink struct {
	mu      sync.Mutex
	batches []Batch
}

func (ts *TestSink) Consume(ctx context.Context, b Batch) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.batches = append(ts.batches, b)
	return nil
}

func (ts *TestSink) GetBatches() []Batch {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	result := make([]Batch, len(ts.batches))
	copy(result, ts.batches)
	return result
}

func (ts *TestSink) Reset() {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.batches = nil
}

func TestBatcher_FlushBySize(t *testing.T) {
	cfg := &config.Config{
		BatchMaxSamples: 3,
		BatchMaxSpanMS:  30000,
		FlushIntervalMS: 500,
		AckEveryN:       50,
		DropTooOldMS:    30000,
	}

	sink := &TestSink{}
	batcher := NewBatcher(cfg, sink)
	defer batcher.Stop()

	// Добавляем 5 сэмплов подряд - должно быть 2 батча (3+2)
	samples := []*telemetryv1.Sample{
		{SessionId: "session1", TsMs: 1000, Metric: telemetryv1.Metric_METRIC_FHR, Value: 120.0},
		{SessionId: "session1", TsMs: 1100, Metric: telemetryv1.Metric_METRIC_FHR, Value: 121.0},
		{SessionId: "session1", TsMs: 1200, Metric: telemetryv1.Metric_METRIC_FHR, Value: 122.0}, // Флаш здесь
		{SessionId: "session1", TsMs: 1300, Metric: telemetryv1.Metric_METRIC_FHR, Value: 123.0},
		{SessionId: "session1", TsMs: 1400, Metric: telemetryv1.Metric_METRIC_FHR, Value: 124.0},
	}

	for _, sample := range samples {
		if err := batcher.Add(sample); err != nil {
			t.Fatalf("Failed to add sample: %v", err)
		}
	}

	// Даем время для обработки
	time.Sleep(100 * time.Millisecond)

	batches := sink.GetBatches()
	if len(batches) != 1 {
		t.Errorf("Expected 1 flushed batch, got %d", len(batches))
	}

	if len(batches) > 0 && len(batches[0].Points) != 3 {
		t.Errorf("Expected 3 points in first batch, got %d", len(batches[0].Points))
	}
}

func TestBatcher_FlushBySpan(t *testing.T) {
	cfg := &config.Config{
		BatchMaxSamples: 100,
		BatchMaxSpanMS:  1000, // 1 секунда
		FlushIntervalMS: 500,
		AckEveryN:       50,
		DropTooOldMS:    30000,
	}

	sink := &TestSink{}
	batcher := NewBatcher(cfg, sink)
	defer batcher.Stop()

	// Добавляем сэмплы с большим временным промежутком
	samples := []*telemetryv1.Sample{
		{SessionId: "session1", TsMs: 1000, Metric: telemetryv1.Metric_METRIC_FHR, Value: 120.0},
		{SessionId: "session1", TsMs: 1500, Metric: telemetryv1.Metric_METRIC_FHR, Value: 121.0},
		{SessionId: "session1", TsMs: 2100, Metric: telemetryv1.Metric_METRIC_FHR, Value: 122.0}, // Флаш здесь (span > 1000ms)
	}

	for _, sample := range samples {
		if err := batcher.Add(sample); err != nil {
			t.Fatalf("Failed to add sample: %v", err)
		}
	}

	// Даем время для обработки
	time.Sleep(100 * time.Millisecond)

	batches := sink.GetBatches()
	if len(batches) != 1 {
		t.Errorf("Expected 1 flushed batch, got %d", len(batches))
		return
	}

	batch := batches[0]
	// После добавления третьего сэмпла (ts=2100) span стал 1100ms (2100-1000), что > 1000ms
	// Поэтому флашится батч с первыми двумя точками, третья остается в новом батче
	if len(batch.Points) != 2 {
		t.Errorf("Expected 2 points in flushed batch, got %d", len(batch.Points))
	}
	span := batch.T1MS - batch.T0MS
	if span != 500 {
		t.Errorf("Expected span of 500ms in flushed batch, got %dms", span)
	}
}

func TestBatcher_OutOfOrderTolerance(t *testing.T) {
	cfg := &config.Config{
		BatchMaxSamples:     100,
		BatchMaxSpanMS:      30000,
		FlushIntervalMS:     500,
		AckEveryN:           50,
		OutOfOrderTolerance: 500 * time.Millisecond,
		DropTooOldMS:        30000,
	}

	sink := &TestSink{}
	batcher := NewBatcher(cfg, sink)

	// Добавляем сэмплы не по порядку, но в пределах tolerance
	samples := []*telemetryv1.Sample{
		{SessionId: "session1", TsMs: 1000, Metric: telemetryv1.Metric_METRIC_FHR, Value: 120.0},
		{SessionId: "session1", TsMs: 1500, Metric: telemetryv1.Metric_METRIC_FHR, Value: 121.0},
		{SessionId: "session1", TsMs: 1200, Metric: telemetryv1.Metric_METRIC_FHR, Value: 122.0}, // out of order, но в пределах tolerance
	}

	for _, sample := range samples {
		if err := batcher.Add(sample); err != nil {
			t.Fatalf("Failed to add sample: %v", err)
		}
	}

	// Даем время для обработки
	time.Sleep(100 * time.Millisecond)

	// Останавливаем batcher, чтобы все батчи были сброшены
	batcher.Stop()

	batches := sink.GetBatches()
	if len(batches) != 1 {
		t.Errorf("Expected 1 batch, got %d", len(batches))
	}

	if len(batches) > 0 && len(batches[0].Points) != 3 {
		t.Errorf("Expected 3 points in batch, got %d", len(batches[0].Points))
	}
}

func TestBatcher_DropTooOld(t *testing.T) {
	cfg := &config.Config{
		BatchMaxSamples:     100,
		BatchMaxSpanMS:      30000,
		FlushIntervalMS:     500,
		AckEveryN:           50,
		OutOfOrderTolerance: 500 * time.Millisecond,
		DropTooOldMS:        2000, // 2 секунды
	}

	sink := &TestSink{}
	batcher := NewBatcher(cfg, sink)
	defer batcher.Stop()

	// Добавляем сэмплы
	samples := []*telemetryv1.Sample{
		{SessionId: "session1", TsMs: 5000, Metric: telemetryv1.Metric_METRIC_FHR, Value: 120.0},
		{SessionId: "session1", TsMs: 6000, Metric: telemetryv1.Metric_METRIC_FHR, Value: 121.0},
		{SessionId: "session1", TsMs: 1000, Metric: telemetryv1.Metric_METRIC_FHR, Value: 122.0}, // Слишком старый (diff > 2000ms)
	}

	for _, sample := range samples {
		if err := batcher.Add(sample); err != nil {
			t.Fatalf("Failed to add sample: %v", err)
		}
	}

	// Проверяем статистику
	received, dropped, _, _ := batcher.GetStats()
	if received != 2 {
		t.Errorf("Expected 2 received samples, got %d", received)
	}
	if dropped != 1 {
		t.Errorf("Expected 1 dropped sample, got %d", dropped)
	}
}

func TestBatcher_TimerFlush(t *testing.T) {
	cfg := &config.Config{
		BatchMaxSamples: 100,
		BatchMaxSpanMS:  30000,
		FlushIntervalMS: 100, // Очень частая проверка
		AckEveryN:       50,
		DropTooOldMS:    30000,
	}

	sink := &TestSink{}
	batcher := NewBatcher(cfg, sink)
	defer batcher.Stop()

	// Добавляем один сэмпл
	sample := &telemetryv1.Sample{
		SessionId: "session1",
		TsMs:      1000,
		Metric:    telemetryv1.Metric_METRIC_FHR,
		Value:     120.0,
	}

	if err := batcher.Add(sample); err != nil {
		t.Fatalf("Failed to add sample: %v", err)
	}

	// Ждем, пока таймер сработает
	time.Sleep(200 * time.Millisecond)

	batches := sink.GetBatches()
	if len(batches) != 1 {
		t.Errorf("Expected 1 batch flushed by timer, got %d", len(batches))
	}

	if len(batches) > 0 && len(batches[0].Points) != 1 {
		t.Errorf("Expected 1 point in batch, got %d", len(batches[0].Points))
	}
}

func TestBatcher_MultipleMetrics(t *testing.T) {
	cfg := &config.Config{
		BatchMaxSamples: 2,
		BatchMaxSpanMS:  30000,
		FlushIntervalMS: 500,
		AckEveryN:       50,
		DropTooOldMS:    30000,
	}

	sink := &TestSink{}
	batcher := NewBatcher(cfg, sink)
	defer batcher.Stop()

	// Добавляем сэмплы разных метрик - должны быть в разных батчах
	samples := []*telemetryv1.Sample{
		{SessionId: "session1", TsMs: 1000, Metric: telemetryv1.Metric_METRIC_FHR, Value: 120.0},
		{SessionId: "session1", TsMs: 1100, Metric: telemetryv1.Metric_METRIC_UC, Value: 50.0},
		{SessionId: "session1", TsMs: 1200, Metric: telemetryv1.Metric_METRIC_FHR, Value: 121.0}, // Флаш FHR
		{SessionId: "session1", TsMs: 1300, Metric: telemetryv1.Metric_METRIC_UC, Value: 51.0},   // Флаш UC
	}

	for _, sample := range samples {
		if err := batcher.Add(sample); err != nil {
			t.Fatalf("Failed to add sample: %v", err)
		}
	}

	// Даем время для обработки
	time.Sleep(100 * time.Millisecond)

	batches := sink.GetBatches()
	if len(batches) != 2 {
		t.Errorf("Expected 2 batches (one per metric), got %d", len(batches))
	}
}
