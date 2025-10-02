package batch

import (
	"context"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	featureextractorv1 "github.com/Krimson/fetal-monitory/proto/feature_extractor"
	mlservicev1 "github.com/Krimson/fetal-monitory/proto/ml_service"
)

// MLServiceSink отправляет признаки в ML сервис для предсказания
type MLServiceSink struct {
	client mlservicev1.MLServiceClient
	conn   *grpc.ClientConn

	// Канал для передачи предсказаний дальше (например, для WebSocket)
	predictionChan chan *mlservicev1.PredictResponse
}

// NewMLServiceSink создает новый экземпляр MLServiceSink
func NewMLServiceSink(mlServiceAddr string) (*MLServiceSink, error) {
	// Подключаемся к ML gRPC сервису
	conn, err := grpc.Dial(mlServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	client := mlservicev1.NewMLServiceClient(conn)

	return &MLServiceSink{
		client:         client,
		conn:           conn,
		predictionChan: make(chan *mlservicev1.PredictResponse, 100),
	}, nil
}

// ConsumeFeatures принимает признаки от feature extractor и отправляет в ML сервис
func (ms *MLServiceSink) ConsumeFeatures(ctx context.Context, features *featureextractorv1.ProcessBatchResponse) error {
	log.Printf("[ML_SERVICE] Processing features: session=%s stv=%.2f ltv=%.2f",
		features.SessionId, features.Stv, features.Ltv)

	// Конвертируем признаки в ML request
	request := &mlservicev1.PredictRequest{
		SessionId:             features.SessionId,
		BatchTsMs:             features.BatchTsMs,
		Stv:                   features.Stv,
		Ltv:                   features.Ltv,
		BaselineHeartRate:     features.BaselineHeartRate,
		TotalDecelerations:    features.TotalDecelerations,
		LateDecelerations:     features.LateDecelerations,
		LateDecelerationRatio: features.LateDecelerationRatio,
		TotalAccelerations:    features.TotalAccelerations,
		AccelDecelRatio:       features.AccelDecelRatio,
		TotalContractions:     features.TotalContractions,
		StvTrend:              features.StvTrend,
		BpmTrend:              features.BpmTrend,
		DataPoints:            features.DataPoints,
		TimeSpanSec:           features.TimeSpanSec,
	}

	// Отправляем в ML сервис (асинхронно, не блокируем основной поток)
	go func() {
		response, err := ms.client.PredictFromFeatures(ctx, request)
		if err != nil {
			log.Printf("[ERROR] Failed to get prediction from ML service: %v", err)
			// При ошибке отправляем response со статусом error и последним известным предиктом (0.0)
			response = &mlservicev1.PredictResponse{
				SessionId:     request.SessionId,
				BatchTsMs:     request.BatchTsMs,
				Prediction:    0.0,
				Status:        "error",
				Message:       err.Error(),
				HasEnoughData: false,
			}
		}

		// Отправляем предсказание в канал
		select {
		case ms.predictionChan <- response:
			log.Printf("[ML_SERVICE] Prediction for session %s: %.4f (status: %s)",
				response.SessionId, response.Prediction, response.Status)
		default:
			log.Printf("[WARN] Prediction channel full, dropping prediction")
		}
	}()

	return nil
}

// GetPredictionChannel возвращает канал с предсказаниями
func (ms *MLServiceSink) GetPredictionChannel() <-chan *mlservicev1.PredictResponse {
	return ms.predictionChan
}

// Close закрывает соединение
func (ms *MLServiceSink) Close() error {
	close(ms.predictionChan)
	return ms.conn.Close()
}
