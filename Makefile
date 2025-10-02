.PHONY: help build up down restart logs clean proto swagger test

help: ## Показать эту помощь
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Собрать все Docker образы
	docker-compose build

up: ## Запустить все сервисы
	docker-compose up -d

down: ## Остановить все сервисы
	docker-compose down

restart: ## Перезапустить все сервисы
	docker-compose restart

logs: ## Показать логи всех сервисов
	docker-compose logs -f

clean: ## Удалить все контейнеры, образы и volumes
	docker-compose down -v --rmi all

# Proto generation
proto-feature: ## Генерировать proto для feature_extractor
	cd proto/feature_extractor && \
	protoc --go_out=. --go-grpc_out=. \
		--go_opt=paths=source_relative --go-grpc_opt=paths=source_relative \
		feature_extractor.proto

proto-ml: ## Генерировать proto для ml_service
	cd proto/ml_service && \
	protoc --go_out=. --go-grpc_out=. \
		--go_opt=paths=source_relative --go-grpc_opt=paths=source_relative \
		ml_service.proto

proto-telemetry: ## Генерировать proto для telemetry
	cd proto/telemetry && \
	protoc --go_out=. --go-grpc_out=. \
		--go_opt=paths=source_relative --go-grpc_opt=paths=source_relative \
		telemetry.proto

proto: proto-feature proto-ml proto-telemetry ## Генерировать все proto файлы

# Swagger generation
swagger-receiver: ## Генерировать Swagger для receiver
	cd receiver && swag init -g cmd/receiver/main.go -o docs

swagger-offline: ## Генерировать Swagger для offline-service
	cd offline-service && swag init -g cmd/server/main.go -o docs

swagger: swagger-receiver swagger-offline ## Генерировать всю Swagger документацию

# Development
dev-receiver: ## Запустить receiver локально
	cd receiver && go run cmd/receiver/main.go

dev-feature: ## Запустить feature_extractor локально
	cd feature_extractor && python grpc_server.py

dev-ml: ## Запустить ml_service локально
	cd ml_service && python grpc_server.py

dev-offline: ## Запустить offline-service локально
	cd offline-service && go run cmd/server/main.go

# Testing
test-go: ## Запустить Go тесты
	go test -v ./...

test-receiver: ## Запустить тесты receiver
	cd receiver && go test -v ./...

# Database
db-migrate: ## Применить миграции
	docker-compose exec postgres psql -U fetal_user -d fetal_monitor -f /docker-entrypoint-initdb.d/001_init.sql

db-backup: ## Создать backup PostgreSQL
	docker exec -t $$(docker-compose ps -q postgres) pg_dumpall -c -U fetal_user > backup_$$(date +%Y%m%d_%H%M%S).sql

# Monitoring
health: ## Проверить health всех сервисов
	@echo "Data Receiver:"
	@curl -s http://localhost:8080/health || echo "Failed"
	@echo "\nOffline Service:"
	@curl -s http://localhost:8081/health || echo "Failed"
	@echo "\nFeature Extractor (gRPC):"
	@grpcurl -plaintext localhost:50052 grpc.health.v1.Health/Check || echo "Failed"
	@echo "\nML Service (gRPC):"
	@grpcurl -plaintext localhost:50053 grpc.health.v1.Health/Check || echo "Failed"

logs-receiver: ## Логи receiver
	docker-compose logs -f data-receiver

logs-feature: ## Логи feature_extractor
	docker-compose logs -f feature-extractor

logs-ml: ## Логи ml_service
	docker-compose logs -f ml-service

logs-offline: ## Логи offline-service
	docker-compose logs -f offline-service

# Cleanup
clean-logs: ## Очистить логи
	docker-compose logs --no-log-prefix > /dev/null

clean-data: ## Очистить volumes с данными (ОСТОРОЖНО!)
	docker-compose down -v