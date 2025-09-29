package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	telemetryv1 "github.com/Krimson/fetal-monitory/proto/telemetry"
	"github.com/Krimson/fetal-monitory/receiver/internal/batch"
	"github.com/Krimson/fetal-monitory/receiver/internal/config"
	"github.com/Krimson/fetal-monitory/receiver/internal/health"
	"github.com/Krimson/fetal-monitory/receiver/internal/server"
)

func main() {
	log.Printf("[INFO] Starting receiver server...")

	cfg := config.Load()
	log.Printf("[INFO] Configuration loaded: grpc_port=%s batch_max_samples=%d",
		cfg.GRPCPort, cfg.BatchMaxSamples)

	sink := &batch.LogSink{}

	batcher := batch.NewBatcher(cfg, sink)

	grpcServer := grpc.NewServer()

	dataServer := server.NewDataServer(cfg, batcher)
	telemetryv1.RegisterDataServiceServer(grpcServer, dataServer)

	healthServer := health.NewHealthServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

	reflection.Register(grpcServer)

	address := fmt.Sprintf(":%s", cfg.GRPCPort)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("[FATAL] Failed to listen on %s: %v", address, err)
	}

	log.Printf("[INFO] gRPC server listening on %s", address)

	healthServer.SetServingStatus("")
	healthServer.SetServingStatus("telemetry.v1.DataService")

	serverErrChan := make(chan error, 1)
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			serverErrChan <- fmt.Errorf("gRPC server error: %w", err)
		}
	}()

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrChan:
		log.Printf("[ERROR] Server error: %v", err)

	case sig := <-shutdownChan:
		log.Printf("[INFO] Received signal %v, starting graceful shutdown...", sig)

		healthServer.SetNotServingStatus("")
		healthServer.SetNotServingStatus("telemetry.v1.DataService")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		grpcServer.GracefulStop()

		batcher.Stop()

		log.Printf("[INFO] Graceful shutdown completed")

		select {
		case <-shutdownCtx.Done():
			log.Printf("[WARN] Graceful shutdown timeout, forcing stop")
			grpcServer.Stop()
		default:
		}
	}

	log.Printf("[INFO] Server stopped")
}
