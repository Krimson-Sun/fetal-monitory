package batch

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	telemetryv1 "github.com/Krimson/fetal-monitory/proto/telemetry"
	"github.com/Krimson/fetal-monitory/receiver/internal/config"
)

type Batcher struct {
	cfg     *config.Config
	sink    Sink
	mu      sync.RWMutex
	batches map[BatchKey]*currentBatch

	flushChan chan Batch
	stopChan  chan struct{}

	stats struct {
		mu         sync.RWMutex
		received   int64
		dropped    int64
		flushed    int64
		outOfOrder int64
	}
}

type LogSink struct{}

func (ls *LogSink) Consume(ctx context.Context, b Batch) error {
	spanMS := b.T1MS - b.T0MS
	log.Printf("[BATCH] session=%s metric=%s points=%d span_ms=%d t0=%d t1=%d",
		b.Key.SessionID,
		b.Key.Metric.String(),
		len(b.Points),
		spanMS,
		b.T0MS,
		b.T1MS)
	return nil
}

func NewBatcher(cfg *config.Config, sink Sink) *Batcher {
	b := &Batcher{
		cfg:       cfg,
		sink:      sink,
		batches:   make(map[BatchKey]*currentBatch),
		flushChan: make(chan Batch, 100),
		stopChan:  make(chan struct{}),
	}

	go b.flushWorker()
	go b.timerFlusher()

	return b
}

func (b *Batcher) Add(sample *telemetryv1.Sample) error {
	if err := b.validateSample(sample); err != nil {
		b.incrementDropped()
		log.Printf("[WARN] Invalid sample dropped: %v", err)
		return nil
	}

	key := BatchKey{
		SessionID: sample.SessionId,
		Metric:    sample.Metric,
	}

	point := Point{
		TsMS:  int64(sample.TsMs),
		Value: sample.Value,
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	batch, exists := b.batches[key]
	if !exists {
		batch = newCurrentBatch(key)
		b.batches[key] = batch
	}

	if exists && len(batch.Points) > 0 {
		timeDiff := batch.T1MS - point.TsMS

		if timeDiff > b.cfg.DropTooOldMS {
			b.incrementDropped()
			log.Printf("[WARN] Sample too old, dropped: session=%s metric=%s ts_diff=%d",
				key.SessionID, key.Metric.String(), timeDiff)
			return nil
		}

		if timeDiff > int64(b.cfg.OutOfOrderTolerance.Milliseconds()) {
			b.incrementOutOfOrder()
			log.Printf("[WARN] Out of order sample: session=%s metric=%s ts_diff=%d",
				key.SessionID, key.Metric.String(), timeDiff)
		}
	}

	if len(batch.Points) > 0 {
		tempSpan := point.TsMS - batch.T0MS
		if point.TsMS < batch.T0MS {
			tempSpan = batch.T1MS - point.TsMS
		}
		if tempSpan > b.cfg.BatchMaxSpanMS {
			b.flushBatch(key, batch)
			batch = newCurrentBatch(key)
			b.batches[key] = batch
		}
	}

	batch.addPoint(point)
	b.incrementReceived()

	if batch.shouldFlushBySize(b.cfg.BatchMaxSamples) {
		b.flushBatch(key, batch)
	}

	return nil
}

func (b *Batcher) validateSample(sample *telemetryv1.Sample) error {
	if sample.SessionId == "" {
		return fmt.Errorf("empty session_id")
	}

	if sample.Metric != telemetryv1.Metric_METRIC_FHR &&
		sample.Metric != telemetryv1.Metric_METRIC_UC {
		return fmt.Errorf("invalid metric: %v", sample.Metric)
	}

	if sample.TsMs == 0 {
		return fmt.Errorf("invalid timestamp: %d", sample.TsMs)
	}

	if math.IsNaN(float64(sample.Value)) || math.IsInf(float64(sample.Value), 0) {
		return fmt.Errorf("invalid value: %f", sample.Value)
	}

	return nil
}

func (b *Batcher) flushBatch(key BatchKey, batch *currentBatch) {
	if len(batch.Points) == 0 {
		return
	}

	batchCopy := batch.clone()

	batch.reset()

	select {
	case b.flushChan <- batchCopy:
		b.incrementFlushed()
	default:
		log.Printf("[WARN] Flush channel full, batch dropped")
		b.incrementDropped()
	}
}

func (b *Batcher) flushWorker() {
	for {
		select {
		case batch := <-b.flushChan:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := b.sink.Consume(ctx, batch); err != nil {
				log.Printf("[ERROR] Failed to consume batch: %v", err)
			}
			cancel()

		case <-b.stopChan:
			return
		}
	}
}

func (b *Batcher) timerFlusher() {
	ticker := time.NewTicker(time.Duration(b.cfg.FlushIntervalMS) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			b.flushOldBatches()

		case <-b.stopChan:
			return
		}
	}
}

func (b *Batcher) flushOldBatches() {
	now := time.Now().UnixMilli()
	flushIntervalMS := b.cfg.FlushIntervalMS

	b.mu.Lock()
	defer b.mu.Unlock()

	for key, batch := range b.batches {
		if len(batch.Points) > 0 &&
			(now-batch.lastAddedMS) > flushIntervalMS {
			b.flushBatch(key, batch)
		}
	}
}

func (b *Batcher) Stop() {
	log.Printf("[INFO] Stopping batcher...")

	b.flushAllBatches()

	for len(b.flushChan) > 0 {
		time.Sleep(10 * time.Millisecond)
	}

	time.Sleep(100 * time.Millisecond)

	select {
	case <-b.stopChan:
	default:
		close(b.stopChan)
	}

	b.logStats()
}

func (b *Batcher) flushAllBatches() {
	b.mu.Lock()
	defer b.mu.Unlock()

	for key, batch := range b.batches {
		if len(batch.Points) > 0 {
			b.flushBatch(key, batch)
		}
	}
}

// Методы для работы со статистикой
func (b *Batcher) incrementReceived() {
	b.stats.mu.Lock()
	b.stats.received++
	b.stats.mu.Unlock()
}

func (b *Batcher) incrementDropped() {
	b.stats.mu.Lock()
	b.stats.dropped++
	b.stats.mu.Unlock()
}

func (b *Batcher) incrementFlushed() {
	b.stats.mu.Lock()
	b.stats.flushed++
	b.stats.mu.Unlock()
}

func (b *Batcher) incrementOutOfOrder() {
	b.stats.mu.Lock()
	b.stats.outOfOrder++
	b.stats.mu.Unlock()
}

func (b *Batcher) logStats() {
	b.stats.mu.RLock()
	defer b.stats.mu.RUnlock()

	log.Printf("[STATS] received=%d dropped=%d flushed=%d out_of_order=%d",
		b.stats.received,
		b.stats.dropped,
		b.stats.flushed,
		b.stats.outOfOrder)
}

func (b *Batcher) GetStats() (received, dropped, flushed, outOfOrder int64) {
	b.stats.mu.RLock()
	defer b.stats.mu.RUnlock()

	return b.stats.received, b.stats.dropped, b.stats.flushed, b.stats.outOfOrder
}
