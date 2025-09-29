.PHONY: build run test clean lint

# Сборка бинарного файла
build:
	go build -o bin/receiver ./cmd/receiver

# Запуск сервера
run: build
	./bin/receiver

# Запуск тестов
test:
	go test -v ./...

# Запуск тестов с покрытием
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Линтинг
lint:
	go vet ./...
	go fmt ./...

# Очистка артефактов
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Запуск сервера с переменными окружения для разработки
dev:
	GRPC_PORT=50051 \
	BATCH_MAX_SAMPLES=10 \
	BATCH_MAX_SPAN_MS=5000 \
	FLUSH_INTERVAL_MS=200 \
	ACK_EVERY_N=5 \
	OUT_OF_ORDER_TOLERANCE_MS=500 \
	DROP_TOO_OLD_MS=10000 \
	go run ./cmd/receiver

# Быстрая сборка и запуск
quick: lint test build
