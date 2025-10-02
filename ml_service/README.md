# ML Service

Микросервис для ML инференса (предсказание состояния плода на основе медицинских признаков).

## Описание

ML Service принимает признаки из `feature_extractor`, накапливает их и выполняет предсказание с помощью ансамбля CatBoost моделей.

## Архитектура

- **collector.py** - Класс для накопления признаков в DataFrame
- **inference.py** - Класс для загрузки и использования CatBoost моделей
- **grpc_server.py** - gRPC сервер для приема запросов на предсказание
- **proto/ml_service.proto** - Определение gRPC сервиса

## Флоу работы

1. `receiver` отправляет данные телеметрии в `feature_extractor`
2. `feature_extractor` извлекает медицинские признаки и отправляет их обратно в `receiver`
3. `receiver` пересылает признаки в `ml_service` через gRPC
4. `ml_service` накапливает признаки и выполняет инференс (может быть долгим)
5. `ml_service` возвращает предсказание в `receiver`
6. `receiver` обновляет последнее предсказание для сессии
7. При отправке данных на фронтенд через WebSocket используется последнее доступное предсказание

**Важно**: Пакеты от WebSocket летят на фронтенд неизменно с частотой 4Hz, просто с последним значением предикта. Предикт обновляется асинхронно по мере поступления ответов от нейросети.

## gRPC API

### PredictFromFeatures
Выполняет предсказание на основе признаков.

**Request:**
```protobuf
message PredictRequest {
  string session_id = 1;
  uint64 batch_ts_ms = 2;
  double stv = 3;
  double ltv = 4;
  double baseline_heart_rate = 5;
  // ... другие признаки
}
```

**Response:**
```protobuf
message PredictResponse {
  string session_id = 1;
  uint64 batch_ts_ms = 2;
  double prediction = 3;      // Вероятность патологии (0.0 - 1.0)
  string status = 4;          // "success", "processing", "error"
  string message = 5;
  bool has_enough_data = 6;
}
```

### ResetCollector
Сбрасывает коллектор для указанной сессии.

## Запуск

### Docker (рекомендуется)
```bash
docker-compose up ml-service
```

### Локально
```bash
cd ml_service
pip install -r requirements.txt
python grpc_server.py
```

## Конфигурация

- **GRPC_PORT** - Порт gRPC сервера (по умолчанию 50053)
- Веса моделей должны находиться в папке `weights/` (.cbm файлы)

## Модель

Используется ансамбль из 3 CatBoost моделей (k-fold):
- `catboost_kfold_0.cbm`
- `catboost_kfold_1.cbm`
- `catboost_kfold_2.cbm`

Предсказание - это среднее вероятностей от всех моделей.

