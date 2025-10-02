package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	telemetryv1 "github.com/Krimson/fetal-monitory/proto/telemetry"
	"github.com/Krimson/fetal-monitory/receiver/internal/batch"
	"github.com/Krimson/fetal-monitory/receiver/internal/config"
	"github.com/Krimson/fetal-monitory/receiver/internal/health"
	"github.com/Krimson/fetal-monitory/receiver/internal/server"
	"github.com/Krimson/fetal-monitory/receiver/internal/session"
	"github.com/Krimson/fetal-monitory/receiver/internal/websocket"
)

func main() {
	log.Printf("[INFO] Starting receiver server...")

	cfg := config.Load()
	log.Printf("[INFO] Configuration loaded: grpc_port=%s http_port=%s redis=%s",
		cfg.GRPCPort, cfg.HTTPPort, cfg.RedisAddr)

	// Инициализируем Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	defer redisClient.Close()

	// Проверяем подключение к Redis
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("[FATAL] Failed to connect to Redis: %v", err)
	}
	log.Printf("[INFO] Connected to Redis: %s", cfg.RedisAddr)

	// Инициализируем PostgreSQL
	postgresRepo, err := session.NewPostgresRepositoryFromDSN(cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("[FATAL] Failed to connect to PostgreSQL: %v", err)
	}
	defer postgresRepo.Close()
	log.Printf("[INFO] Connected to PostgreSQL")

	// Создаем Session Manager
	redisStore := session.NewRedisStore(redisClient)
	sessionManager := session.NewManager(redisStore, postgresRepo)
	log.Printf("[INFO] Session manager initialized")

	// Создаем WebSocket hub
	wsHub := websocket.NewHub()
	go wsHub.Run()

	// Создаем Feature Extractor Sink с интеграцией Session Manager
	featureExtractorAddr := "feature-extractor:50052" // Docker service name
	featureSink, err := batch.NewFeatureExtractorSinkWithSession(featureExtractorAddr, sessionManager)
	if err != nil {
		log.Fatalf("[FATAL] Failed to create feature extractor sink: %v", err)
	}
	defer featureSink.Close()

	// Создаем композитный sink (логирование + feature extraction)
	logSink := &batch.LogSink{}
	compositeSink := batch.NewCompositeSink(logSink, featureSink)

	batcher := batch.NewBatcher(cfg, compositeSink)

	// Запускаем обработчик обработанных данных из feature extractor
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go wsHub.ProcessedDataConsumer(ctx, featureSink.GetProcessedBatchChannel())

	// Настраиваем gRPC сервер
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

	// Настраиваем HTTP сервер с роутером
	router := mux.NewRouter()

	// WebSocket endpoint
	router.HandleFunc("/ws", wsHub.HandleWebSocket)

	// Health check
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Session management API
	sessionHandler := session.NewHTTPHandler(sessionManager)
	sessionHandler.RegisterRoutes(router)

	// CORS middleware (для разработки)
	router.Use(corsMiddleware)

	httpPort := cfg.HTTPPort
	log.Printf("[INFO] HTTP server (WebSocket + API) listening on :%s", httpPort)

	healthServer.SetServingStatus("")
	healthServer.SetServingStatus("telemetry.v1.DataService")

	// Запускаем серверы
	serverErrChan := make(chan error, 2)

	// gRPC сервер
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			serverErrChan <- fmt.Errorf("gRPC server error: %w", err)
		}
	}()

	// HTTP сервер (WebSocket + API)
	go func() {
		if err := http.ListenAndServe(":"+httpPort, router); err != nil {
			serverErrChan <- fmt.Errorf("HTTP server error: %w", err)
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

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		cancel() // Останавливаем consumer

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

// corsMiddleware добавляет CORS заголовки для разработки
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Разрешаем запросы с любого источника
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Credentials", "true")

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")

		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization, X-CSRF-Token, X-Requested-With")

		w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Type")

		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
