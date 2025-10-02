package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPPort string `default:"8081"`

	RedisAddr     string        `default:"localhost:6379"`
	RedisPassword string        `default:""`
	RedisDB       int           `default:"0"`
	RedisTTL      time.Duration `default:"24h"`

	PostgreSQLConnStr string `default:"host=localhost port=5432 user=fetal_user password=fetal_pass dbname=fetal_monitor sslmode=disable"`

	FilterServiceAddr string `default:"localhost:50051"`
	MLServiceAddr     string `default:"localhost:50052"`
}

func LoadConfig() *Config {
	cfg := &Config{
		HTTPPort:          getEnv("HTTP_PORT", "8081"),
		RedisAddr:         getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:     getEnv("REDIS_PASSWORD", ""),
		RedisDB:           getEnvInt("REDIS_DB", 0),
		RedisTTL:          getEnvDuration("REDIS_TTL", 24*time.Hour),
		PostgreSQLConnStr: getEnv("POSTGRES_CONN_STR", "host=localhost port=5432 user=fetal_user password=fetal_pass dbname=fetal_monitor sslmode=disable"),
		FilterServiceAddr: getEnv("FILTER_SERVICE_ADDR", "localhost:50051"),
		MLServiceAddr:     getEnv("ML_SERVICE_ADDR", "localhost:50052"),
	}
	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
