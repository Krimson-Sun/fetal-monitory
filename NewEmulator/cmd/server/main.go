// cmd/server/main.go
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"

	"new-fetal-emulator/internal/grpcserver"
	telemetryv1 "new-fetal-emulator/proto/v1"
)

func main() {
	port := flag.Int("port", 50051, "Port for gRPC server")
	flag.Parse()

	server := grpcserver.NewServer()
	grpcServer := grpc.NewServer()
	telemetryv1.RegisterDataServiceServer(grpcServer, server)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	log.Printf("Starting gRPC server on port %d", *port)

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// Горутина для мониторинга статистики
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				sessions := server.GetAllSessions()
				log.Printf("=== Server Statistics ===")
				log.Printf("Active sessions: %d", len(sessions))
				for sessionID, stats := range sessions {
					log.Printf("Session %s: total=%d, FHR=%d, UC=%d, last_update=%v",
						sessionID, stats.ReceivedCount,
						stats.MetricCounts[telemetryv1.Metric_METRIC_FHR],
						stats.MetricCounts[telemetryv1.Metric_METRIC_UC],
						time.Since(stats.LastUpdate))
				}
				log.Printf("=========================")
			}
		}
	}()

	<-stop
	log.Println("Shutting down server...")
	grpcServer.GracefulStop()
	log.Println("Server stopped")
}
