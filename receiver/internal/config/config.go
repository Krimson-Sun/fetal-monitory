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

	// Batch settings
	BatchMaxSamples     int
	BatchMaxSpanMS      int64
	FlushIntervalMS     int64
	AckEveryN           int
	OutOfOrderTolerance time.Duration
	DropTooOldMS        int64
}

// Load загружает конфигурацию из переменных окружения с дефолтными значениями
func Load() *Config {
	return &Config{
		GRPCPort:            getEnvString("GRPC_PORT", "50051"),
		BatchMaxSamples:     getEnvInt("BATCH_MAX_SAMPLES", 250),
		BatchMaxSpanMS:      getEnvInt64("BATCH_MAX_SPAN_MS", 30000),
		FlushIntervalMS:     getEnvInt64("FLUSH_INTERVAL_MS", 500),
		AckEveryN:           getEnvInt("ACK_EVERY_N", 50),
		OutOfOrderTolerance: time.Duration(getEnvInt64("OUT_OF_ORDER_TOLERANCE_MS", 1000)) * time.Millisecond,
		DropTooOldMS:        getEnvInt64("DROP_TOO_OLD_MS", 30000),
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
