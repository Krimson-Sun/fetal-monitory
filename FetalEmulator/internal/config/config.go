package config

import (
	"flag"
	"time"
)

type Config struct {
	Emulator EmulatorConfig
	FHR      FHRConfig
	TOCO     TOCOConfig
	Output   OutputConfig
}

type EmulatorConfig struct {
	Duration   time.Duration
	SampleRate time.Duration
}

type FHRConfig struct {
	MinValue    int
	MaxValue    int
	BaseValue   int
	Variability int
}

type TOCOConfig struct {
	MinContractionInterval time.Duration
	MaxContractionInterval time.Duration
	ContractionDuration    time.Duration
	PeakIntensity          int
}

type OutputConfig struct {
	FilePath string
	Format   string
}

func Load() (*Config, error) {
	var cfg Config

	// Параметры командной строки
	outputFile := flag.String("output", "data/test_data.jsonl", "Выходной файл")
	duration := flag.String("duration", "30s", "Длительность работы")
	sampleRate := flag.String("rate", "0.5s", "Частота дискретизации")

	flag.Parse()

	// Парсим длительность
	dur, err := time.ParseDuration(*duration)
	if err != nil {
		return nil, err
	}

	rate, err := time.ParseDuration(*sampleRate)
	if err != nil {
		return nil, err
	}

	cfg.Emulator = EmulatorConfig{
		Duration:   dur,
		SampleRate: rate,
	}

	cfg.FHR = FHRConfig{
		MinValue:    110,
		MaxValue:    160,
		BaseValue:   140,
		Variability: 10,
	}

	cfg.TOCO = TOCOConfig{
		MinContractionInterval: 3 * time.Minute,
		MaxContractionInterval: 5 * time.Minute,
		ContractionDuration:    90 * time.Second,
		PeakIntensity:          60,
	}

	cfg.Output = OutputConfig{
		FilePath: *outputFile,
		Format:   "jsonl",
	}

	return &cfg, nil
}
