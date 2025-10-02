# Fetal Monitoring System

Система мониторинга состояния плода в режиме реального времени с использованием машинного обучения для предсказания патологий.

## 📋 Описание

Fetal Monitoring System - это распределенная система для анализа данных кардиотокографии (КТГ) плода в режиме реального времени. Система принимает телеметрию от медицинских устройств, извлекает медицинские признаки, выполняет ML-инференс и предоставляет результаты через WebSocket для визуализации на фронтенде.

### Основные возможности

- 📊 **Реал-тайм мониторинг** - Обработка данных ЧСС и маточных сокращений с частотой 4Hz
- 🧠 **ML предсказания** - Ансамбль CatBoost моделей для оценки риска патологий
- 📈 **Извлечение признаков** - Автоматический расчет STV, LTV, baseline, акселераций/децелераций
- 🔌 **WebSocket стрим** - Низколатентная передача данных на фронтенд
- 💾 **Управление сессиями** - Сохранение и восстановление сессий мониторинга
- 📁 **Offline анализ** - Загрузка и анализ CSV файлов
- 📝 **API документация** - Swagger UI для всех REST endpoints

## 🏗️ Архитектура

Система состоит из 6 микросервисов:

```
┌─────────────┐
│   Emulator  │ (Генерация тестовых данных)
└──────┬──────┘
       │ gRPC
       ▼
┌─────────────────────────────────────────────┐
│           Data Receiver (Go)                │
│  • Прием телеметрии (gRPC)                 │
│  • Батчинг данных                           │
│  • WebSocket Hub                            │
│  • Session Management                       │
└──┬────────────────────────────────────────┬─┘
   │ gRPC                                    │ WebSocket
   ▼                                         ▼
┌──────────────────┐                  ┌─────────────┐
│ Feature          │                  │  Frontend   │
│ Extractor (Py)   │                  │  (Browser)  │
│  • Фильтрация    │                  └─────────────┘
│  • STV/LTV       │
│  • События       │
└──────┬───────────┘
       │ Features
       ▼
┌──────────────────┐
│  ML Service (Py) │
│  • CatBoost      │
│  • Prediction    │
└──────────────────┘

┌──────────────────┐     ┌──────────────────┐
│   PostgreSQL     │     │      Redis       │
│  (Persistence)   │     │     (Cache)      │
└──────────────────┘     └──────────────────┘

┌──────────────────┐
│ Offline Service  │ (CSV анализ)
│      (Go)        │
└──────────────────┘
```

### Микросервисы

#### 1. **Data Receiver** (Go)
- **Порты**: 50051 (gRPC), 8080 (HTTP/WebSocket)
- **Функции**:
  - Прием телеметрии от устройств/эмулятора
  - Батчинг данных для эффективной обработки
  - Отправка батчей в Feature Extractor
  - Управление WebSocket соединениями
  - Управление сессиями (Redis + PostgreSQL)
  - REST API для управления сессиями
- **Технологии**: Go 1.24, gRPC, WebSocket, Redis, PostgreSQL

#### 2. **Feature Extractor** (Python)
- **Порт**: 50052 (gRPC)
- **Функции**:
  - Фильтрация физиологических сигналов
  - Расчет медицинских метрик (STV, LTV, baseline)
  - Детекция акселераций и децелераций
  - Детекция маточных сокращений
  - Расчет трендов
- **Технологии**: Python 3.11, gRPC, NumPy, Pandas, SciPy

#### 3. **ML Service** (Python)
- **Порт**: 50053 (gRPC)
- **Функции**:
  - Накопление признаков из Feature Extractor
  - Инференс на ансамбле CatBoost моделей
  - Возврат вероятности патологии (0.0-1.0)
  - Кэширование последних предсказаний
- **Технологии**: Python 3.11, gRPC, CatBoost, PyTorch, Pandas

#### 4. **Offline Service** (Go)
- **Порт**: 8081 (HTTP)
- **Функции**:
  - Загрузка CSV файлов
  - Обработка через Feature Extractor и ML Service
  - REST API с Swagger UI
- **Технологии**: Go 1.24, gRPC clients, Swagger

#### 5. **Emulator** (Go)
- **Функции**:
  - Чтение CSV файлов с историческими данными
  - Стриминг данных в Data Receiver
  - Симуляция реального устройства
- **Технологии**: Go 1.24, gRPC

#### 6. **Базы данных**
- **PostgreSQL** - Постоянное хранение сессий и данных
- **Redis** - Кэширование активных сессий и метрик

## 🔄 Поток данных

### Реал-тайм режим

```
1. Emulator → Data Receiver (gRPC)
   Отправка точек FHR и UC с временными метками

2. Data Receiver → Feature Extractor (gRPC)
   Батчи данных каждые 250мс (4Hz)

3. Feature Extractor → Data Receiver (gRPC Response)
   Медицинские метрики + отфильтрованные данные

4. Data Receiver → ML Service (gRPC, асинхронно)
   Признаки для предсказания

5. ML Service → Data Receiver (gRPC Response)
   Вероятность патологии

6. Data Receiver → Frontend (WebSocket, 4Hz)
   Все метрики + последний предикт
```

**Важно**: WebSocket пакеты отправляются с фиксированной частотой 4Hz с последним доступным значением ML предикта. Предикт обновляется асинхронно по мере готовности нейросети, не блокируя основной поток.

## 🚀 Деплой

### Требования

- **Docker** >= 20.10
- **Docker Compose** >= 2.0
- **Go** >= 1.24 (для локальной разработки)
- **Python** >= 3.11 (для локальной разработки)
- **protoc** >= 3.20 (для генерации proto файлов)

### Быстрый старт

1. **Клонируйте репозиторий**
```bash
git clone <repository-url>
cd fetal-monitory
```

2. **Запустите все сервисы через Docker Compose**
```bash
docker-compose up --build
```

Это запустит все микросервисы:
- PostgreSQL (порт 5432)
- Redis (порт 6379)
- Feature Extractor (порт 50052)
- ML Service (порт 50053)
- Data Receiver (порт 50051, 8080)
- Offline Service (порт 8081)
- Stream Emulator (автоматически начнет отправку данных)
- Frontend (порт 3000)

3. **Откройте в браузере**
- Интерфейс: `http://localhost:3000`
- WebSocket соединение: `ws://localhost:8080/ws?session_id=<your-session-id>`
- Swagger UI (Data Receiver): http://localhost:8080/swagger/
- Swagger UI (Offline Service): http://localhost:8081/swagger/

### Запуск отдельных сервисов

#### Data Receiver
```bash
cd receiver
go run cmd/receiver/main.go
```

#### Feature Extractor
```bash
cd feature_extractor
pip install -r requirements.txt
python grpc_server.py
```

#### ML Service
```bash
cd ml_service
pip install -r requirements.txt
python grpc_server.py
```

#### Offline Service
```bash
cd offline-service
go run cmd/server/main.go
```

#### Emulator
```bash
cd emulator
go run cmd/server/main.go
```

### Конфигурация

Все сервисы настраиваются через переменные окружения. См. `docker-compose.yml` для полного списка.

#### Основные переменные Data Receiver:

```bash
GRPC_PORT=50051                    # gRPC порт
HTTP_PORT=8080                     # HTTP/WebSocket порт
BATCH_MAX_SAMPLES=2                # Размер батча
BATCH_MAX_SPAN_MS=250             # Частота отправки батчей (4Hz)
FLUSH_INTERVAL_MS=250             # Интервал флаша
FEATURE_EXTRACTOR_ADDR=feature-extractor:50052
ML_SERVICE_ADDR=ml-service:50053
REDIS_ADDR=redis:6379
POSTGRES_DSN=postgres://fetal_user:fetal_pass@postgres:5432/fetal_monitor
SESSION_DATA_TTL_SECONDS=86400    # TTL для сессий в Redis
```

## 📡 API

### Data Receiver REST API

#### Создать сессию
```bash
POST /sessions
Content-Type: application/json

{
  "patient_id": "P001",
  "doctor_id": "D001",
  "facility_id": "F001",
  "notes": "Regular checkup"
}
```

#### Получить сессию
```bash
GET /sessions/{session_id}
```

#### Остановить сессию
```bash
POST /sessions/{session_id}/stop
```

#### Сохранить сессию в БД
```bash
POST /sessions/{session_id}/save
Content-Type: application/json

{
  "notes": "Additional notes"
}
```

#### Список сессий
```bash
GET /sessions?limit=10&offset=0
```

### WebSocket

```javascript
const ws = new WebSocket('ws://localhost:8080/ws?session_id=<session-id>');

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Prediction:', data.prediction);
  console.log('Records:', data.records);
};
```

**Формат WebSocket сообщения:**
```json
{
  "message": "Batch processed successfully",
  "prediction": 0.234,
  "session_id": "uuid",
  "status": "processed",
  "records": {
    "stv": 5.2,
    "ltv": 12.3,
    "baseline_heart_rate": 142.5,
    "accelerations": [...],
    "decelerations": [...],
    "contractions": [...],
    "stvs": [5.1, 5.2, 5.3],
    "ltvs": [12.1, 12.3],
    "filtered_bpm_batch": {
      "time_sec": [0, 0.25, 0.5],
      "value": [140, 142, 141]
    },
    "filtered_uterus_batch": {
      "time_sec": [0, 0.25, 0.5],
      "value": [10, 15, 12]
    }
  }
}
```

### Offline Service REST API

См. Swagger UI: http://localhost:8081/swagger/

## 🧪 Тестирование

### Запуск с тестовыми данными

1. Убедитесь, что CSV файлы находятся в `data/`:
   - `data/fhr.csv` - данные ЧСС
   - `data/uc.csv` - данные маточных сокращений

2. Запустите систему:
```bash
docker-compose up
```

3. Эмулятор автоматически начнет отправку данных

### Установка SESSION_ID для эмулятора

```bash
SESSION_ID=my-test-session docker-compose up stream-emulator
```

## 📁 Структура проекта

```
fetal-monitory/
├── data/                      # Тестовые CSV файлы
│   ├── fhr.csv
│   └── uc.csv
├── emulator/                  # Go эмулятор устройства
│   ├── cmd/
│   ├── internal/
│   └── Dockerfile
├── feature_extractor/         # Python сервис извлечения признаков
│   ├── collector.py
│   ├── preprocessor.py
│   ├── grpc_server.py
│   ├── requirements.txt
│   └── Dockerfile
├── ml_service/                # Python ML сервис
│   ├── collector.py
│   ├── inference.py
│   ├── grpc_server.py
│   ├── requirements.txt
│   ├── Dockerfile
│   └── weights/               # CatBoost модели
│       ├── catboost_kfold_0.cbm
│       ├── catboost_kfold_1.cbm
│       └── catboost_kfold_2.cbm
├── offline-service/           # Go сервис для CSV анализа
│   ├── cmd/
│   ├── internal/
│   ├── docs/                  # Swagger документация
│   └── Dockerfile
├── receiver/                  # Go сервис приема данных
│   ├── cmd/
│   ├── internal/
│   │   ├── batch/            # Батчинг
│   │   ├── config/           # Конфигурация
│   │   ├── health/           # Health checks
│   │   ├── server/           # gRPC сервер
│   │   ├── session/          # Управление сессиями
│   │   └── websocket/        # WebSocket hub
│   ├── docs/                 # Swagger документация
│   ├── Dockerfile
│   └── Makefile
├── proto/                     # Protocol Buffers определения
│   ├── feature_extractor/
│   │   └── feature_extractor.proto
│   ├── ml_service/
│   │   └── ml_service.proto
│   └── telemetry/
│       └── telemetry.proto
├── migrations/                # SQL миграции для PostgreSQL
│   └── 001_init.sql
├── docker-compose.yml         # Оркестрация сервисов
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## 🔧 Разработка

### Генерация proto файлов

#### Go
```bash
cd proto/feature_extractor
protoc --go_out=. --go-grpc_out=. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative feature_extractor.proto

cd ../ml_service
protoc --go_out=. --go-grpc_out=. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative ml_service.proto

cd ../telemetry
protoc --go_out=. --go-grpc_out=. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative telemetry.proto
```

#### Python (автоматически в Dockerfile)
```bash
cd feature_extractor
python -m grpc_tools.protoc --python_out=. --grpc_python_out=. --proto_path=../proto/feature_extractor ../proto/feature_extractor/feature_extractor.proto

cd ../ml_service
python -m grpc_tools.protoc --python_out=. --grpc_python_out=. --proto_path=../proto/ml_service ../proto/ml_service/ml_service.proto
```

### Swagger документация

Генерация Swagger для Go сервисов:

```bash
cd receiver
swag init -g cmd/receiver/main.go -o docs

cd ../offline-service
swag init -g cmd/server/main.go -o docs
```

## 📊 Мониторинг

### Health Checks

Все сервисы имеют health check endpoints:

```bash
# Data Receiver
curl http://localhost:8080/health

# Offline Service
curl http://localhost:8081/health

# gRPC health checks
grpcurl -plaintext localhost:50051 grpc.health.v1.Health/Check
grpcurl -plaintext localhost:50052 grpc.health.v1.Health/Check
grpcurl -plaintext localhost:50053 grpc.health.v1.Health/Check
```

### Логи

```bash
# Все сервисы
docker-compose logs -f

# Конкретный сервис
docker-compose logs -f data-receiver
docker-compose logs -f ml-service
docker-compose logs -f feature-extractor
```

## 🤝 Вклад в проект

1. Fork репозитория
2. Создайте feature branch (`git checkout -b feature/amazing-feature`)
3. Commit изменений (`git commit -m 'Add amazing feature'`)
4. Push в branch (`git push origin feature/amazing-feature`)
5. Откройте Pull Request


## 👥 Авторы

Тимур Пшиншев

Марина Савовичева

Денис Гридусов

Дмитрий Комягин

Дмитрий Черкашин

## 📞 Контакты

timurpshinshev@gmail.com
krimson.sun@yandex.ru

## 🙏 Благодарности

- CatBoost team за отличную ML библиотеку
- gRPC community
- Go и Python communities

