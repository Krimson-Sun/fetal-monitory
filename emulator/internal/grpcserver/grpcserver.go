// internal/grpcserver/grpcserver.go
package grpcserver

import (
	"log"
	"sync"
	"time"

	telemetryv1 "github.com/Krimson/fetal-monitory/proto/telemetry"
)

type Server struct {
	telemetryv1.UnimplementedDataServiceServer
	mu       sync.RWMutex
	sessions map[string]*SessionStats
}

type SessionStats struct {
	ReceivedCount uint64
	LastUpdate    time.Time
	MetricCounts  map[telemetryv1.Metric]uint64
}

func NewServer() *Server {
	return &Server{
		sessions: make(map[string]*SessionStats),
	}
}

func (s *Server) PushSamples(stream telemetryv1.DataService_PushSamplesServer) error {
	var sessionID string
	var receivedCount uint64

	for {
		sample, err := stream.Recv()
		if err != nil {
			log.Printf("Stream closed for session %s: %v", sessionID, err)
			return err
		}

		if sessionID == "" {
			sessionID = sample.SessionId
			log.Printf("New connection for session: %s", sessionID)
		}

		// Обновляем статистику
		s.mu.Lock()
		if _, exists := s.sessions[sessionID]; !exists {
			s.sessions[sessionID] = &SessionStats{
				MetricCounts: make(map[telemetryv1.Metric]uint64),
			}
		}
		stats := s.sessions[sessionID]
		stats.ReceivedCount++
		stats.LastUpdate = time.Now()
		stats.MetricCounts[sample.Metric]++
		receivedCount = stats.ReceivedCount
		s.mu.Unlock()

		// Логируем полученные данные
		metricName := "UNKNOWN"
		switch sample.Metric {
		case telemetryv1.Metric_METRIC_FHR:
			metricName = "FHR"
		case telemetryv1.Metric_METRIC_UC:
			metricName = "UC"
		}

		log.Printf("Received sample: session=%s, metric=%s, value=%.2f, ts_ms=%d",
			sessionID, metricName, sample.Value, sample.TsMs)

		// Отправляем подтверждение каждые 10 samples
		if receivedCount%10 == 0 {
			ack := &telemetryv1.Ack{
				SessionId:   sessionID,
				ReceivedCnt: receivedCount,
			}
			if err := stream.Send(ack); err != nil {
				log.Printf("Failed to send ack: %v", err)
				return err
			}
			log.Printf("Sent ack for session %s: count=%d", sessionID, receivedCount)
		}
	}
}

func (s *Server) GetSessionStats(sessionID string) *SessionStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if stats, exists := s.sessions[sessionID]; exists {
		return stats
	}
	return nil
}

func (s *Server) GetAllSessions() map[string]*SessionStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessions := make(map[string]*SessionStats)
	for k, v := range s.sessions {
		sessions[k] = v
	}
	return sessions
}
