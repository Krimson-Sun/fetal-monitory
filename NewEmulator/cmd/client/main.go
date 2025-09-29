// cmd/client/main.go
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"new-fetal-emulator/internal/csvreader"
	"new-fetal-emulator/internal/grpcclient"

	telemetryv1 "new-fetal-emulator/proto/v1"
)

func main() {
	var (
		fhrFile    = flag.String("fhr", "fhr.csv", "Файл с данными пульса плода")
		ucFile     = flag.String("uc", "uc.csv", "Файл с данными сокращений матки")
		serverAddr = flag.String("server", "localhost:50051", "Адрес gRPC сервера")
		sessionID  = flag.String("session", "session-123", "ID сессии")
	)
	flag.Parse()

	// Чтение CSV файлов
	fhrData, err := csvreader.ReadCSVFile(*fhrFile)
	if err != nil {
		log.Fatalf("Failed to read FHR data: %v", err)
	}

	ucData, err := csvreader.ReadCSVFile(*ucFile)
	if err != nil {
		log.Fatalf("Failed to read UC data: %v", err)
	}

	log.Printf("Loaded %d FHR records and %d UC records", len(fhrData), len(ucData))

	// Создание gRPC клиента
	grpcClient, err := grpcclient.NewGRPCClient(*serverAddr, *sessionID)
	if err != nil {
		log.Fatalf("Failed to create gRPC client: %v", err)
	}
	defer grpcClient.Close()

	// Каналы для данных
	fhrSamples := make(chan telemetryv1.Sample, 100)
	ucSamples := make(chan telemetryv1.Sample, 100)
	mergedSamples := make(chan telemetryv1.Sample, 200)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Обработка сигналов для graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup

	// Запуск потоков чтения CSV данных в реальном времени
	startTime := time.Now()

	wg.Add(1)
	go func() {
		defer wg.Done()
		fhrDataChan := make(chan csvreader.DataPoint, len(fhrData))
		go csvreader.StreamData(fhrData, startTime, fhrDataChan)

		for point := range fhrDataChan {
			fhrSamples <- telemetryv1.Sample{
				SessionId: *sessionID,
				TsMs:      uint64(startTime.Add(time.Duration(point.TimeSec * float64(time.Second))).UnixMilli()),
				Metric:    telemetryv1.Metric_METRIC_FHR,
				Value:     float32(point.Value),
			}
		}
		close(fhrSamples)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		ucDataChan := make(chan csvreader.DataPoint, len(ucData))
		go csvreader.StreamData(ucData, startTime, ucDataChan)

		for point := range ucDataChan {
			ucSamples <- telemetryv1.Sample{
				SessionId: *sessionID,
				TsMs:      uint64(startTime.Add(time.Duration(point.TimeSec * float64(time.Second))).UnixMilli()),
				Metric:    telemetryv1.Metric_METRIC_UC,
				Value:     float32(point.Value),
			}
		}
		close(ucSamples)
	}()

	// Объединение потоков данных
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(mergedSamples)

		for {
			select {
			case sample, ok := <-fhrSamples:
				if !ok {
					fhrSamples = nil
				} else {
					mergedSamples <- sample
				}
			case sample, ok := <-ucSamples:
				if !ok {
					ucSamples = nil
				} else {
					mergedSamples <- sample
				}
			case <-ctx.Done():
				return
			}

			if fhrSamples == nil && ucSamples == nil {
				return
			}
		}
	}()

	// Отправка данных через gRPC
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := grpcClient.PushSamples(ctx, mergedSamples); err != nil {
			log.Printf("Error pushing samples: %v", err)
		}
	}()

	// Ожидание сигнала завершения
	<-sigCh
	log.Println("Received shutdown signal...")
	cancel()
	wg.Wait()
	log.Println("Application stopped gracefully")
}
