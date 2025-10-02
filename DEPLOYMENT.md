# Инструкция по деплою

## Развертывание в Production

### Требования

- Docker >= 20.10
- Docker Compose >= 2.0
- Минимум 4GB RAM
- 10GB свободного места на диске

### Шаги деплоя

#### 1. Подготовка окружения

```bash
# Клонируйте репозиторий
git clone <repository-url>
cd fetal-monitory

# Создайте .env файл для production переменных
cp .env.example .env
```

#### 2. Настройка .env файла

Создайте `.env` файл в корне проекта:

```bash
# PostgreSQL
POSTGRES_DB=fetal_monitor
POSTGRES_USER=fetal_user
POSTGRES_PASSWORD=<secure-password>

# Redis
REDIS_PASSWORD=<redis-password>

# Data Receiver
GRPC_PORT=50051
HTTP_PORT=8080
BATCH_MAX_SAMPLES=2
BATCH_MAX_SPAN_MS=250
FLUSH_INTERVAL_MS=250
SESSION_DATA_TTL_SECONDS=86400

# External URLs (для production)
FEATURE_EXTRACTOR_ADDR=feature-extractor:50052
ML_SERVICE_ADDR=ml-service:50053
```

#### 3. Обновите docker-compose.yml для production

Создайте `docker-compose.prod.yml`:

```yaml
version: "3.9"

services:
  redis:
    image: redis:7-alpine
    restart: always
    command: redis-server --appendonly yes --requirepass ${REDIS_PASSWORD}
    volumes:
      - redis-data:/data
    networks:
      - fetal-network

  postgres:
    image: postgres:15-alpine
    restart: always
    environment:
      POSTGRES_DB: ${POSTGRES_DB}
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    volumes:
      - postgres-data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d:ro
    networks:
      - fetal-network

  feature-extractor:
    build:
      context: .
      dockerfile: ./feature_extractor/Dockerfile
    restart: always
    environment:
      - GRPC_PORT=50052
    networks:
      - fetal-network

  ml-service:
    build:
      context: .
      dockerfile: ./ml_service/Dockerfile
    restart: always
    environment:
      - GRPC_PORT=50053
    networks:
      - fetal-network

  data-receiver:
    build:
      context: .
      dockerfile: ./receiver/Dockerfile
    restart: always
    ports:
      - "50051:50051"
      - "8080:8080"
    environment:
      - GRPC_PORT=${GRPC_PORT}
      - HTTP_PORT=${HTTP_PORT}
      - BATCH_MAX_SAMPLES=${BATCH_MAX_SAMPLES}
      - BATCH_MAX_SPAN_MS=${BATCH_MAX_SPAN_MS}
      - FLUSH_INTERVAL_MS=${FLUSH_INTERVAL_MS}
      - FEATURE_EXTRACTOR_ADDR=${FEATURE_EXTRACTOR_ADDR}
      - ML_SERVICE_ADDR=${ML_SERVICE_ADDR}
      - REDIS_ADDR=redis:6379
      - REDIS_PASSWORD=${REDIS_PASSWORD}
      - REDIS_DB=0
      - POSTGRES_DSN=postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable
      - SESSION_DATA_TTL_SECONDS=${SESSION_DATA_TTL_SECONDS}
    depends_on:
      - redis
      - postgres
      - feature-extractor
      - ml-service
    networks:
      - fetal-network

  offline-service:
    build:
      context: ./offline-service
      dockerfile: Dockerfile
    restart: always
    ports:
      - "8081:8081"
    environment:
      - HTTP_PORT=8081
      - REDIS_ADDR=redis:6379
      - REDIS_PASSWORD=${REDIS_PASSWORD}
      - REDIS_DB=0
      - REDIS_TTL=24h
      - POSTGRES_CONN_STR=host=postgres port=5432 user=${POSTGRES_USER} password=${POSTGRES_PASSWORD} dbname=${POSTGRES_DB} sslmode=disable
      - FILTER_SERVICE_ADDR=feature-extractor:50052
      - ML_SERVICE_ADDR=ml-service:50053
    depends_on:
      - redis
      - postgres
      - feature-extractor
      - ml-service
    networks:
      - fetal-network

volumes:
  redis-data:
    driver: local
  postgres-data:
    driver: local

networks:
  fetal-network:
    driver: bridge
```

#### 4. Запуск в production

```bash
# Соберите образы
docker-compose -f docker-compose.prod.yml build

# Запустите сервисы
docker-compose -f docker-compose.prod.yml up -d

# Проверьте статус
docker-compose -f docker-compose.prod.yml ps

# Проверьте логи
docker-compose -f docker-compose.prod.yml logs -f
```

#### 5. Проверка работоспособности

```bash
# Health checks
curl http://localhost:8080/health
curl http://localhost:8081/health

# gRPC health checks
grpcurl -plaintext localhost:50051 grpc.health.v1.Health/Check
```

### Остановка и обновление

```bash
# Остановка без удаления данных
docker-compose -f docker-compose.prod.yml stop

# Полная остановка с удалением контейнеров
docker-compose -f docker-compose.prod.yml down

# Обновление с пересборкой
docker-compose -f docker-compose.prod.yml down
git pull
docker-compose -f docker-compose.prod.yml build
docker-compose -f docker-compose.prod.yml up -d
```

## Деплой на Kubernetes

### 1. Создайте namespace

```yaml
# namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: fetal-monitoring
```

```bash
kubectl apply -f namespace.yaml
```

### 2. ConfigMaps и Secrets

```yaml
# configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: fetal-config
  namespace: fetal-monitoring
data:
  GRPC_PORT: "50051"
  HTTP_PORT: "8080"
  BATCH_MAX_SAMPLES: "2"
  BATCH_MAX_SPAN_MS: "250"
  FLUSH_INTERVAL_MS: "250"
  FEATURE_EXTRACTOR_ADDR: "feature-extractor:50052"
  ML_SERVICE_ADDR: "ml-service:50053"
```

```yaml
# secrets.yaml
apiVersion: v1
kind: Secret
metadata:
  name: fetal-secrets
  namespace: fetal-monitoring
type: Opaque
stringData:
  POSTGRES_PASSWORD: "<your-password>"
  REDIS_PASSWORD: "<your-password>"
```

### 3. Deployments

См. примеры в `k8s/` директории.

## Мониторинг в Production

### Prometheus

Добавьте метрики endpoints в каждый сервис и настройте Prometheus для сбора метрик.

### Grafana

Создайте дашборды для мониторинга:
- Throughput (точек/сек)
- Latency (время обработки)
- Error rate
- ML prediction distribution
- WebSocket connections

### Логирование

Настройте централизованное логирование (ELK, Loki):

```bash
# Пример с fluentd
docker-compose -f docker-compose.prod.yml -f docker-compose.logging.yml up -d
```

## Backup

### PostgreSQL

```bash
# Создание backup
docker exec -t fetal-monitoring-postgres-1 pg_dumpall -c -U fetal_user > dump_$(date +%Y%m%d_%H%M%S).sql

# Восстановление
cat dump_20250102_120000.sql | docker exec -i fetal-monitoring-postgres-1 psql -U fetal_user
```

### Redis

```bash
# Backup RDB
docker exec fetal-monitoring-redis-1 redis-cli save
docker cp fetal-monitoring-redis-1:/data/dump.rdb ./backup/

# Restore
docker cp ./backup/dump.rdb fetal-monitoring-redis-1:/data/
docker restart fetal-monitoring-redis-1
```

## Безопасность

### SSL/TLS

Используйте Nginx или Traefik как reverse proxy с SSL:

```nginx
server {
    listen 443 ssl http2;
    server_name fetal-monitor.example.com;

    ssl_certificate /etc/nginx/certs/cert.pem;
    ssl_certificate_key /etc/nginx/certs/key.pem;

    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

### Firewall

Откройте только необходимые порты:
- 443 (HTTPS)
- 8080 (WebSocket, за reverse proxy)

Закройте прямой доступ к:
- 5432 (PostgreSQL)
- 6379 (Redis)
- 50051-50053 (gRPC, только internal)

## Масштабирование

### Горизонтальное

```bash
# Увеличьте количество реплик
docker-compose -f docker-compose.prod.yml up -d --scale data-receiver=3
```

### Вертикальное

Увеличьте ресурсы в `docker-compose.prod.yml`:

```yaml
services:
  ml-service:
    deploy:
      resources:
        limits:
          cpus: '4'
          memory: 8G
        reservations:
          cpus: '2'
          memory: 4G
```

