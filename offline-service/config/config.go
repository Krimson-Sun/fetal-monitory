package config

import (
	"time"
)

type Config struct {
	HTTPPort string `default:"8080"`

	RedisAddr     string        `default:"localhost:6379"`
	RedisPassword string        `default:""`
	RedisDB       int           `default:"0"`
	RedisTTL      time.Duration `default:"24h"`

	PostgreSQLConnStr string `default:"host=localhost port=5432 user=postgres password=postgres dbname=medical sslmode=disable"`

	FilterServiceAddr string `default:"localhost:50051"`
	MLServiceAddr     string `default:"localhost:50052"`
}

func LoadConfig() *Config {
	return &Config{
		HTTPPort:          "8080",
		RedisAddr:         "localhost:6379",
		RedisPassword:     "",
		RedisDB:           0,
		RedisTTL:          24 * time.Hour,
		PostgreSQLConnStr: "host=localhost port=5432 user=postgres password=postgres dbname=medical sslmode=disable",
		FilterServiceAddr: "localhost:50051",
		MLServiceAddr:     "localhost:50052",
	}
}
