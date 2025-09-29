package server

import (
	"io"
	"log"
	"sync"

	telemetryv1 "github.com/Krimson/fetal-monitory/proto/telemetry"
	"github.com/Krimson/fetal-monitory/receiver/internal/batch"
	"github.com/Krimson/fetal-monitory/receiver/internal/config"
)

// DataServer реализует telemetryv1.DataServiceServer
type DataServer struct {
	telemetryv1.UnimplementedDataServiceServer
	cfg     *config.Config
	batcher *batch.Batcher
}

// NewDataServer создает новый экземпляр DataServer
func NewDataServer(cfg *config.Config, batcher *batch.Batcher) *DataServer {
	return &DataServer{
		cfg:     cfg,
		batcher: batcher,
	}
}

// PushSamples обрабатывает стрим сэмплов от клиента
func (s *DataServer) PushSamples(stream telemetryv1.DataService_PushSamplesServer) error {
	log.Printf("[INFO] New PushSamples stream started")

	// Счетчики для Ack
	var (
		mu              sync.Mutex
		totalReceived   uint64 = 0
		sessionCounters        = make(map[string]uint64)
	)

	// Горутина для отправки Ack
	ackChan := make(chan *telemetryv1.Sample, 100)
	defer close(ackChan)

	go s.ackSender(stream, ackChan, &mu, &totalReceived, sessionCounters)

	// Основной цикл чтения сэмплов
	for {
		select {
		case <-stream.Context().Done():
			log.Printf("[INFO] PushSamples stream context cancelled")
			return stream.Context().Err()

		default:
			sample, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					log.Printf("[INFO] PushSamples stream finished normally")
					return nil
				}
				log.Printf("[ERROR] Failed to receive sample: %v", err)
				return err
			}

			// Обрабатываем сэмпл
			if err := s.processSample(sample); err != nil {
				log.Printf("[WARN] Failed to process sample: %v", err)
				// Не возвращаем ошибку, продолжаем обработку
				continue
			}

			// Отправляем в канал для Ack
			select {
			case ackChan <- sample:
			default:
				log.Printf("[WARN] Ack channel full, skipping ack for sample")
			}
		}
	}
}

// processSample обрабатывает один сэмпл
func (s *DataServer) processSample(sample *telemetryv1.Sample) error {
	return s.batcher.Add(sample)
}

// ackSender отправляет Ack сообщения клиенту
func (s *DataServer) ackSender(
	stream telemetryv1.DataService_PushSamplesServer,
	ackChan <-chan *telemetryv1.Sample,
	mu *sync.Mutex,
	totalReceived *uint64,
	sessionCounters map[string]uint64,
) {
	for {
		select {
		case <-stream.Context().Done():
			return

		case sample, ok := <-ackChan:
			if !ok {
				return
			}

			mu.Lock()
			*totalReceived++
			sessionCounters[sample.SessionId]++

			// Отправляем Ack каждые ACK_EVERY_N сэмплов
			if *totalReceived%uint64(s.cfg.AckEveryN) == 0 {
				ack := &telemetryv1.Ack{
					SessionId:   sample.SessionId,
					ReceivedCnt: sessionCounters[sample.SessionId],
				}

				if err := stream.Send(ack); err != nil {
					log.Printf("[ERROR] Failed to send ack: %v", err)
					mu.Unlock()
					return
				}

				log.Printf("[DEBUG] Sent ack: session=%s count=%d",
					ack.SessionId, ack.ReceivedCnt)
			}
			mu.Unlock()
		}
	}
}
