package session

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore реализует CacheStore для Redis (Infrastructure Layer)
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore создает новый экземпляр RedisStore
func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{
		client: client,
	}
}

// ===== Ключи Redis =====

func sessionKey(sessionID string) string {
	return fmt.Sprintf("session:%s:metadata", sessionID)
}

func metricsKey(sessionID string) string {
	return fmt.Sprintf("session:%s:features:current", sessionID)
}

func eventsKey(sessionID string, eventType EventType) string {
	return fmt.Sprintf("session:%s:events:%s", sessionID, eventType)
}

func timeSeriesKey(sessionID string, seriesType TimeSeriesType) string {
	return fmt.Sprintf("session:%s:timeseries:%s", sessionID, seriesType)
}

func filteredDataKey(sessionID string, metricType MetricType) string {
	return fmt.Sprintf("session:%s:filtered:%s", sessionID, metricType)
}

// ===== Управление сессиями =====

func (r *RedisStore) SetSession(ctx context.Context, session *Session) error {
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	return r.client.Set(ctx, sessionKey(session.ID), data, 0).Err()
}

func (r *RedisStore) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	data, err := r.client.Get(ctx, sessionKey(sessionID)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("session not found: %s", sessionID)
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var session Session
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

func (r *RedisStore) DeleteSession(ctx context.Context, sessionID string) error {
	// Удаляем все ключи, связанные с сессией
	pattern := fmt.Sprintf("session:%s:*", sessionID)

	iter := r.client.Scan(ctx, 0, pattern, 0).Iterator()
	pipe := r.client.Pipeline()

	for iter.Next(ctx) {
		pipe.Del(ctx, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("failed to scan keys: %w", err)
	}

	_, err := pipe.Exec(ctx)
	return err
}

func (r *RedisStore) SessionExists(ctx context.Context, sessionID string) (bool, error) {
	count, err := r.client.Exists(ctx, sessionKey(sessionID)).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *RedisStore) SetSessionTTL(ctx context.Context, sessionID string, ttl int) error {
	pattern := fmt.Sprintf("session:%s:*", sessionID)
	duration := time.Duration(ttl) * time.Second

	iter := r.client.Scan(ctx, 0, pattern, 0).Iterator()
	pipe := r.client.Pipeline()

	for iter.Next(ctx) {
		pipe.Expire(ctx, iter.Val(), duration)
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("failed to scan keys: %w", err)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// ===== Метрики =====

func (r *RedisStore) SetMetrics(ctx context.Context, metrics *SessionMetrics) error {
	// Сохраняем как Hash для эффективного обновления отдельных полей
	fields := map[string]interface{}{
		"stv":                     metrics.STV,
		"ltv":                     metrics.LTV,
		"baseline_heart_rate":     metrics.BaselineHeartRate,
		"total_accelerations":     metrics.TotalAccelerations,
		"total_decelerations":     metrics.TotalDecelerations,
		"late_decelerations":      metrics.LateDecelerations,
		"late_deceleration_ratio": metrics.LateDecelerationRatio,
		"total_contractions":      metrics.TotalContractions,
		"accel_decel_ratio":       metrics.AccelDecelRatio,
		"stv_trend":               metrics.STVTrend,
		"bpm_trend":               metrics.BPMTrend,
		"data_points":             metrics.DataPoints,
		"time_span_sec":           metrics.TimeSpanSec,
		"updated_at":              metrics.UpdatedAt.Unix(),
	}

	return r.client.HSet(ctx, metricsKey(metrics.SessionID), fields).Err()
}

func (r *RedisStore) GetMetrics(ctx context.Context, sessionID string) (*SessionMetrics, error) {
	data, err := r.client.HGetAll(ctx, metricsKey(sessionID)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("metrics not found for session: %s", sessionID)
	}

	metrics := &SessionMetrics{SessionID: sessionID}

	// Парсим значения из Hash
	if val, ok := data["stv"]; ok {
		metrics.STV, _ = strconv.ParseFloat(val, 64)
	}
	if val, ok := data["ltv"]; ok {
		metrics.LTV, _ = strconv.ParseFloat(val, 64)
	}
	if val, ok := data["baseline_heart_rate"]; ok {
		metrics.BaselineHeartRate, _ = strconv.ParseFloat(val, 64)
	}
	if val, ok := data["total_accelerations"]; ok {
		i, _ := strconv.ParseInt(val, 10, 32)
		metrics.TotalAccelerations = int32(i)
	}
	if val, ok := data["total_decelerations"]; ok {
		i, _ := strconv.ParseInt(val, 10, 32)
		metrics.TotalDecelerations = int32(i)
	}
	if val, ok := data["late_decelerations"]; ok {
		i, _ := strconv.ParseInt(val, 10, 32)
		metrics.LateDecelerations = int32(i)
	}
	if val, ok := data["late_deceleration_ratio"]; ok {
		metrics.LateDecelerationRatio, _ = strconv.ParseFloat(val, 64)
	}
	if val, ok := data["total_contractions"]; ok {
		i, _ := strconv.ParseInt(val, 10, 32)
		metrics.TotalContractions = int32(i)
	}
	if val, ok := data["accel_decel_ratio"]; ok {
		metrics.AccelDecelRatio, _ = strconv.ParseFloat(val, 64)
	}
	if val, ok := data["stv_trend"]; ok {
		metrics.STVTrend, _ = strconv.ParseFloat(val, 64)
	}
	if val, ok := data["bpm_trend"]; ok {
		metrics.BPMTrend, _ = strconv.ParseFloat(val, 64)
	}
	if val, ok := data["data_points"]; ok {
		i, _ := strconv.ParseInt(val, 10, 32)
		metrics.DataPoints = int32(i)
	}
	if val, ok := data["time_span_sec"]; ok {
		metrics.TimeSpanSec, _ = strconv.ParseFloat(val, 64)
	}
	if val, ok := data["updated_at"]; ok {
		timestamp, _ := strconv.ParseInt(val, 10, 64)
		metrics.UpdatedAt = time.Unix(timestamp, 0)
	}

	return metrics, nil
}

// ===== События =====

func (r *RedisStore) AppendEvents(ctx context.Context, sessionID string, events []SessionEvent) error {
	if len(events) == 0 {
		return nil
	}

	pipe := r.client.Pipeline()

	for _, event := range events {
		data, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to marshal event: %w", err)
		}

		key := eventsKey(sessionID, event.Type)
		pipe.RPush(ctx, key, data)
	}

	_, err := pipe.Exec(ctx)
	return err
}

func (r *RedisStore) GetEvents(ctx context.Context, sessionID string, eventType EventType) ([]SessionEvent, error) {
	key := eventsKey(sessionID, eventType)
	data, err := r.client.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	events := make([]SessionEvent, 0, len(data))
	for _, item := range data {
		var event SessionEvent
		if err := json.Unmarshal([]byte(item), &event); err != nil {
			continue // Пропускаем поврежденные записи
		}
		events = append(events, event)
	}

	return events, nil
}

func (r *RedisStore) GetAllEvents(ctx context.Context, sessionID string) ([]SessionEvent, error) {
	var allEvents []SessionEvent

	eventTypes := []EventType{EventTypeAcceleration, EventTypeDeceleration, EventTypeContraction}

	for _, eventType := range eventTypes {
		events, err := r.GetEvents(ctx, sessionID, eventType)
		if err != nil {
			continue // Пропускаем ошибки для конкретных типов
		}
		allEvents = append(allEvents, events...)
	}

	return allEvents, nil
}

func (r *RedisStore) EventExists(ctx context.Context, sessionID string, eventType EventType, startTime float64) (bool, error) {
	events, err := r.GetEvents(ctx, sessionID, eventType)
	if err != nil {
		return false, err
	}

	for _, event := range events {
		if event.StartTime == startTime {
			return true, nil
		}
	}

	return false, nil
}

// ===== Временные ряды =====

func (r *RedisStore) AppendTimeSeries(ctx context.Context, sessionID string, seriesType TimeSeriesType, points []TimeSeriesPoint) error {
	if len(points) == 0 {
		return nil
	}

	key := timeSeriesKey(sessionID, seriesType)
	pipe := r.client.Pipeline()

	for _, point := range points {
		data, err := json.Marshal(point)
		if err != nil {
			return fmt.Errorf("failed to marshal time series point: %w", err)
		}
		pipe.RPush(ctx, key, data)
	}

	_, err := pipe.Exec(ctx)
	return err
}

func (r *RedisStore) GetTimeSeries(ctx context.Context, sessionID string, seriesType TimeSeriesType) ([]TimeSeriesPoint, error) {
	key := timeSeriesKey(sessionID, seriesType)
	data, err := r.client.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get time series: %w", err)
	}

	points := make([]TimeSeriesPoint, 0, len(data))
	for _, item := range data {
		var point TimeSeriesPoint
		if err := json.Unmarshal([]byte(item), &point); err != nil {
			continue
		}
		points = append(points, point)
	}

	return points, nil
}

func (r *RedisStore) GetTimeSeriesCount(ctx context.Context, sessionID string, seriesType TimeSeriesType) (int, error) {
	key := timeSeriesKey(sessionID, seriesType)
	count, err := r.client.LLen(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

// ===== Отфильтрованные данные =====

func (r *RedisStore) UpdateFilteredData(ctx context.Context, sessionID string, metricType MetricType, points []FilteredDataPoint) error {
	if len(points) == 0 {
		return nil
	}

	key := filteredDataKey(sessionID, metricType)
	pipe := r.client.Pipeline()

	// Используем Sorted Set где score = time_sec
	// Это автоматически обновляет существующие точки и добавляет новые
	for _, point := range points {
		data, err := json.Marshal(point)
		if err != nil {
			return fmt.Errorf("failed to marshal filtered data point: %w", err)
		}

		pipe.ZAdd(ctx, key, redis.Z{
			Score:  point.TimeSec,
			Member: data,
		})
	}

	_, err := pipe.Exec(ctx)
	return err
}

func (r *RedisStore) GetFilteredData(ctx context.Context, sessionID string, metricType MetricType) ([]FilteredDataPoint, error) {
	key := filteredDataKey(sessionID, metricType)

	// Получаем все элементы, отсортированные по score (time_sec)
	data, err := r.client.ZRange(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get filtered data: %w", err)
	}

	points := make([]FilteredDataPoint, 0, len(data))
	for _, item := range data {
		var point FilteredDataPoint
		if err := json.Unmarshal([]byte(item), &point); err != nil {
			continue
		}
		points = append(points, point)
	}

	return points, nil
}

func (r *RedisStore) GetFilteredDataCount(ctx context.Context, sessionID string, metricType MetricType) (int, error) {
	key := filteredDataKey(sessionID, metricType)
	count, err := r.client.ZCard(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

// ===== Получение всех данных сессии =====

func (r *RedisStore) GetSessionData(ctx context.Context, sessionID string) (*SessionData, error) {
	session, err := r.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	metrics, _ := r.GetMetrics(ctx, sessionID) // Может не быть метрик
	events, _ := r.GetAllEvents(ctx, sessionID)
	stvSeries, _ := r.GetTimeSeries(ctx, sessionID, TimeSeriesTypeSTV)
	ltvSeries, _ := r.GetTimeSeries(ctx, sessionID, TimeSeriesTypeLTV)
	bpmData, _ := r.GetFilteredData(ctx, sessionID, MetricTypeBPM)
	uterusData, _ := r.GetFilteredData(ctx, sessionID, MetricTypeUterus)

	return &SessionData{
		Session:            session,
		Metrics:            metrics,
		Events:             events,
		TimeSeriesSTV:      stvSeries,
		TimeSeriesLTV:      ltvSeries,
		FilteredBPMData:    bpmData,
		FilteredUterusData: uterusData,
	}, nil
}
