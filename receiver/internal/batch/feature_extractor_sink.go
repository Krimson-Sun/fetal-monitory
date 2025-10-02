package batch

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	featureextractorv1 "github.com/Krimson/fetal-monitory/proto/feature_extractor"
	telemetryv1 "github.com/Krimson/fetal-monitory/proto/telemetry"
)

// SessionManager интерфейс для управления сессиями
type SessionManager interface {
	ProcessFeatureBatch(ctx context.Context, response *featureextractorv1.ProcessBatchResponse) error
}

// FeatureExtractorSink отправляет батчи в Python сервис для обработки
type FeatureExtractorSink struct {
	client         featureextractorv1.FeatureExtractorServiceClient
	conn           *grpc.ClientConn
	sessionManager SessionManager

	// Канал для передачи обработанных данных дальше (например, для WebSocket)
	processedBatchChan chan *featureextractorv1.ProcessBatchResponse
}

// NewFeatureExtractorSink создает новый экземпляр FeatureExtractorSink (без session manager)
func NewFeatureExtractorSink(featureExtractorAddr string) (*FeatureExtractorSink, error) {
	return NewFeatureExtractorSinkWithSession(featureExtractorAddr, nil)
}

// NewFeatureExtractorSinkWithSession создает новый экземпляр с session manager
func NewFeatureExtractorSinkWithSession(featureExtractorAddr string, sessionManager SessionManager) (*FeatureExtractorSink, error) {
	// Подключаемся к Python gRPC сервису
	conn, err := grpc.Dial(featureExtractorAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	client := featureextractorv1.NewFeatureExtractorServiceClient(conn)

	return &FeatureExtractorSink{
		client:             client,
		conn:               conn,
		sessionManager:     sessionManager,
		processedBatchChan: make(chan *featureextractorv1.ProcessBatchResponse, 100),
	}, nil
}

// Consume реализует интерфейс Sink
func (fs *FeatureExtractorSink) Consume(ctx context.Context, b Batch) error {
	log.Printf("[FEATURE_EXTRACTOR] Processing batch: session=%s metric=%s points=%d",
		b.Key.SessionID, b.Key.Metric.String(), len(b.Points))

	// Конвертируем batch в gRPC request
	request, err := fs.convertBatchToRequest(b)
	if err != nil {
		log.Printf("[ERROR] Failed to convert batch to request: %v", err)
		return err
	}

	// Отправляем в Python сервис
	response, err := fs.client.ProcessBatch(ctx, request)
	if err != nil {
		log.Printf("[ERROR] Failed to process batch in feature extractor: %v", err)
		return err
	}

	// Сохраняем в Session Manager (если доступен)
	if fs.sessionManager != nil {
		if err := fs.sessionManager.ProcessFeatureBatch(ctx, response); err != nil {
			log.Printf("[ERROR] Failed to process feature batch in session manager: %v", err)
			// Не возвращаем ошибку, продолжаем обработку
		}
	}

	// Отправляем обработанные данные в канал для дальнейшей обработки (WebSocket)
	select {
	case fs.processedBatchChan <- response:
		log.Printf("[FEATURE_EXTRACTOR] Processed batch successfully: session=%s stv=%.2f ltv=%.2f baseline=%.1f",
			response.SessionId, response.Stv, response.Ltv, response.BaselineHeartRate)
	default:
		log.Printf("[WARN] Processed batch channel full, dropping batch")
	}

	return nil
}

// convertBatchToRequest конвертирует внутренний Batch в gRPC request
func (fs *FeatureExtractorSink) convertBatchToRequest(b Batch) (*featureextractorv1.ProcessBatchRequest, error) {
	request := &featureextractorv1.ProcessBatchRequest{
		SessionId:  b.Key.SessionID,
		BatchTsMs:  uint64(time.Now().UnixMilli()),
		BpmData:    []*featureextractorv1.DataPoint{},
		UterusData: []*featureextractorv1.DataPoint{},
	}

	// Разделяем точки по типу метрики
	for _, point := range b.Points {
		dataPoint := &featureextractorv1.DataPoint{
			TimeSec: float64(point.TsMS) / 1000.0, // Конвертируем из мс в секунды
			Value:   float64(point.Value),
		}

		switch b.Key.Metric {
		case telemetryv1.Metric_METRIC_FHR:
			request.BpmData = append(request.BpmData, dataPoint)
		case telemetryv1.Metric_METRIC_UC:
			request.UterusData = append(request.UterusData, dataPoint)
		}
	}

	return request, nil
}

// GetProcessedBatchChannel возвращает канал с обработанными данными
func (fs *FeatureExtractorSink) GetProcessedBatchChannel() <-chan *featureextractorv1.ProcessBatchResponse {
	return fs.processedBatchChan
}

// Close закрывает соединение
func (fs *FeatureExtractorSink) Close() error {
	close(fs.processedBatchChan)
	return fs.conn.Close()
}

// CompositeSink объединяет несколько Sink для параллельной обработки батчей
type CompositeSink struct {
	sinks []Sink
}

// NewCompositeSink создает новый композитный Sink
func NewCompositeSink(sinks ...Sink) *CompositeSink {
	return &CompositeSink{
		sinks: sinks,
	}
}

// Consume отправляет batch во все подключенные sink'и
func (cs *CompositeSink) Consume(ctx context.Context, b Batch) error {
	for _, sink := range cs.sinks {
		if err := sink.Consume(ctx, b); err != nil {
			log.Printf("[ERROR] Sink failed to consume batch: %v", err)
			// Не возвращаем ошибку, продолжаем обработку в других sink'ах
		}
	}
	return nil
}
