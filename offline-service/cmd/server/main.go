package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"offline-service/config"
	"offline-service/internal/handler"
	"offline-service/internal/pb"
	"offline-service/internal/repository"
	"offline-service/internal/service"
)

func main() {
	// Загрузка конфигурации
	cfg := config.LoadConfig()

	// Инициализация заглушек вместо реальных репозиториев
	redisRepo := repository.NewRedisStub(cfg.RedisTTL)
	defer redisRepo.Close()

	postgresRepo := repository.NewPostgresStub()
	defer postgresRepo.Close()

	log.Println("🚀 Using STUB repositories (Redis & PostgreSQL)")

	// Инициализация gRPC клиентов (заглушки уже запущены отдельно)
	filterConn, err := grpc.NewClient(cfg.FilterServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Printf("⚠️  Failed to connect to filter service: %v", err)
		log.Println("⚠️  Make sure filter stub is running on", cfg.FilterServiceAddr)
	} else {
		defer filterConn.Close()
	}

	mlConn, err := grpc.NewClient(cfg.MLServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Printf("⚠️  Failed to connect to ML service: %v", err)
		log.Println("⚠️  Make sure ML stub is running on", cfg.MLServiceAddr)
	} else {
		defer mlConn.Close()
	}

	// Инициализация сервиса
	var medicalService *service.MedicalService

	if filterConn != nil && mlConn != nil {
		medicalService = service.NewMedicalService(
			pb.NewFilterServiceClient(filterConn),
			pb.NewMLServiceClient(mlConn),
			redisRepo,
			postgresRepo,
		)
		log.Printf("✅ Connected to gRPC services: %s, %s", cfg.FilterServiceAddr, cfg.MLServiceAddr)
	} else {
		log.Println("❌ Running without gRPC services - some functionality will be limited")
		// Можно создать сервис с nil клиентами, если обработать это в коде
	}

	// Инициализация HTTP хендлера
	httpHandler := handler.NewHTTPHandler(medicalService)

	// Настройка маршрутов
	mux := http.NewServeMux()
	mux.HandleFunc("/upload", httpHandler.UploadDualCSV)
	mux.HandleFunc("/upload-dual", httpHandler.UploadDualCSV)
	mux.HandleFunc("/decision", httpHandler.HandleDecision)
	mux.HandleFunc("/session", httpHandler.GetSessionData) // Новый endpoint

	// Добавляем endpoint для отладки заглушек
	mux.HandleFunc("/debug/stats", func(w http.ResponseWriter, r *http.Request) {
		stats := map[string]interface{}{
			"redis":     redisRepo.GetStats(),
			"postgres":  postgresRepo.GetStats(),
			"timestamp": time.Now().Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	})

	handlerWithCORS := enableCORS(mux)
	// Настройка HTTP сервера
	server := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      handlerWithCORS,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		log.Printf("✅ Medical service starting on port %s", cfg.HTTPPort)
		log.Printf("✅ Using STUB repositories")
		log.Printf("✅ Debug stats available at http://localhost:%s/debug/stats", cfg.HTTPPort)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Server failed: %v", err)
		}
	}()

	// Ожидание сигнала завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("🛑 Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("❌ Server forced to shutdown: %v", err)
	}

	log.Println("✅ Server exited gracefully")
}

func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			return
		}

		next.ServeHTTP(w, r)
	})
}
