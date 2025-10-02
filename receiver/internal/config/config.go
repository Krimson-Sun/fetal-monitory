package config

import (
	"os"
	"strconv"
	"time"
)

// Config содержит все настройки приложения
type Config struct {
	// gRPC server settings
	GRPCPort string
	HTTPPort string

	// Batch settings
	BatchMaxSamples     int
	BatchMaxSpanMS      int64
	FlushIntervalMS     int64
	AckEveryN           int
	OutOfOrderTolerance time.Duration
	DropTooOldMS        int64

	// Redis settings
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// PostgreSQL settings
	PostgresDSN string

	// Session settings
	SessionDataTTLSeconds int

	// External services
	FeatureExtractorAddr string
	MLServiceAddr        string
}

// Load загружает конфигурацию из переменных окружения с дефолтными значениями
func Load() *Config {
	return &Config{
		GRPCPort:            getEnvString("GRPC_PORT", "50051"),
		HTTPPort:            getEnvString("HTTP_PORT", "8080"),
		BatchMaxSamples:     getEnvInt("BATCH_MAX_SAMPLES", 2),     // 1 точка каждой метрики при 4Hz = 2 точки за 0.25сек
		BatchMaxSpanMS:      getEnvInt64("BATCH_MAX_SPAN_MS", 250), // 250мс для 4Hz на фронтенде
		FlushIntervalMS:     getEnvInt64("FLUSH_INTERVAL_MS", 250), // Отправляем каждые 250мс (4Hz)
		AckEveryN:           getEnvInt("ACK_EVERY_N", 10),          // Меньше acks
		OutOfOrderTolerance: time.Duration(getEnvInt64("OUT_OF_ORDER_TOLERANCE_MS", 250)) * time.Millisecond,
		DropTooOldMS:        getEnvInt64("DROP_TOO_OLD_MS", 5000), // Более короткий timeout

		// Redis
		RedisAddr:     getEnvString("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnvString("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),

		// PostgreSQL
		PostgresDSN: getEnvString("POSTGRES_DSN", "postgres://fetal_user:fetal_pass@localhost:5432/fetal_monitor?sslmode=disable"),

		// Session
		SessionDataTTLSeconds: getEnvInt("SESSION_DATA_TTL_SECONDS", 86400), // 24 часа по умолчанию

		// External services
		FeatureExtractorAddr: getEnvString("FEATURE_EXTRACTOR_ADDR", "feature-extractor:50052"),
		MLServiceAddr:        getEnvString("ML_SERVICE_ADDR", "ml-service:50053"),
	}
}

func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}
