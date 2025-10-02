# 🎨 API для Фронтенда

Простое руководство для подключения фронтенда к системе мониторинга плода.

---

## 🚀 Быстрый старт

### Базовый URL
```
HTTP API: http://localhost:8080
WebSocket: ws://localhost:8080/ws
```

---

## 📋 Основные эндпоинты

### 1. Создать сессию
```http
POST /api/sessions
Content-Type: application/json

{
  "patient_id": "patient-001",
  "doctor_id": "dr-ivanov",
  "facility_id": "hospital-1",
  "notes": "Плановое обследование"
}
```

**Ответ:**
```json
{
  "session": {
    "id": "abc-123-def-456",
    "status": "ACTIVE",
    "started_at": "2025-10-02T10:00:00Z"
  }
}
```

### 2. Получить информацию о сессии
```http
GET /api/sessions/{session_id}
```

**Ответ:**
```json
{
  "session": {
    "id": "abc-123",
    "status": "ACTIVE",
    "started_at": "2025-10-02T10:00:00Z",
    "total_duration_ms": 120000,
    "total_data_points": 2400
  },
  "metrics": {
    "stv": 5.2,
    "ltv": 11.8,
    "baseline_heart_rate": 138.5,
    "total_accelerations": 5,
    "total_decelerations": 2
  }
}
```

### 3. Список всех сессий
```http
GET /api/sessions?limit=50&offset=0
```

### 4. Остановить сессию
```http
POST /api/sessions/{session_id}/stop
```

### 5. Сохранить в БД
```http
POST /api/sessions/{session_id}/save
Content-Type: application/json

{
  "notes": "Обследование завершено. Пациент в норме."
}
```

### 6. Удалить сессию
```http
DELETE /api/sessions/{session_id}
```

---

## 🔌 WebSocket для real-time данных

### Подключение
```javascript
const sessionId = 'abc-123'; // ID из POST /api/sessions
const ws = new WebSocket(`ws://localhost:8080/ws?session_id=${sessionId}`);

ws.onopen = () => {
  console.log('Подключено к WebSocket');
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Получены данные:', data);
  
  // Обновить графики
  updateCharts(data.records);
};

ws.onerror = (error) => {
  console.error('Ошибка WebSocket:', error);
};

ws.onclose = () => {
  console.log('WebSocket закрыт');
};
```

### Формат данных от WebSocket

```json
{
  "session_id": "abc-123",
  "status": "processed",
  "records": {
    "stv": 5.2,
    "ltv": 11.8,
    "baseline_heart_rate": 138.5,
    
    "total_accelerations": 5,
    "total_decelerations": 2,
    "total_contractions": 3,
    
    "accelerations": [
      {
        "start": 125.5,
        "end": 140.2,
        "duration": 14.7,
        "amplitude": 18.5
      }
    ],
    
    "decelerations": [
      {
        "start": 200.1,
        "end": 215.3,
        "duration": 15.2,
        "amplitude": -12.3,
        "is_late": false
      }
    ],
    
    "contractions": [
      {
        "start": 180.0,
        "end": 210.0,
        "duration": 30.0,
        "amplitude": 45.2
      }
    ],
    
    "filtered_bpm_batch": {
      "time_sec": [0.0, 0.25, 0.5, 0.75],
      "value": [138.5, 139.2, 138.8, 139.5]
    },
    
    "filtered_uterus_batch": {
      "time_sec": [0.0, 0.25, 0.5, 0.75],
      "value": [10.2, 15.3, 18.7, 22.1]
    }
  }
}
```

---

## 📊 Полный пример (React)

```jsx
import React, { useState, useEffect, useRef } from 'react';

function FetalMonitor() {
  const [sessionId, setSessionId] = useState(null);
  const [metrics, setMetrics] = useState({});
  const [isActive, setIsActive] = useState(false);
  const wsRef = useRef(null);

  // Начать мониторинг
  const startMonitoring = async () => {
    try {
      const response = await fetch('http://localhost:8080/api/sessions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          patient_id: 'patient-001',
          doctor_id: 'dr-ivanov',
          notes: 'Начало мониторинга'
        })
      });

      const { session } = await response.json();
      setSessionId(session.id);
      setIsActive(true);

      // Подключить WebSocket
      const ws = new WebSocket(`ws://localhost:8080/ws?session_id=${session.id}`);
      
      ws.onmessage = (event) => {
        const data = JSON.parse(event.data);
        setMetrics(data.records);
      };

      wsRef.current = ws;
    } catch (error) {
      console.error('Ошибка:', error);
    }
  };

  // Остановить мониторинг
  const stopMonitoring = async () => {
    try {
      // Остановить сессию
      await fetch(`http://localhost:8080/api/sessions/${sessionId}/stop`, {
        method: 'POST'
      });

      // Сохранить в БД
      await fetch(`http://localhost:8080/api/sessions/${sessionId}/save`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          notes: 'Мониторинг завершен'
        })
      });

      // Закрыть WebSocket
      if (wsRef.current) {
        wsRef.current.close();
      }

      setIsActive(false);
    } catch (error) {
      console.error('Ошибка:', error);
    }
  };

  return (
    <div>
      <h1>Мониторинг плода</h1>
      
      {!isActive ? (
        <button onClick={startMonitoring}>Начать мониторинг</button>
      ) : (
        <>
          <div className="metrics">
            <div>STV: {metrics.stv?.toFixed(2) || '-'}</div>
            <div>LTV: {metrics.ltv?.toFixed(2) || '-'}</div>
            <div>ЧСС: {metrics.baseline_heart_rate?.toFixed(1) || '-'} уд/мин</div>
            <div>Акселерации: {metrics.total_accelerations || 0}</div>
            <div>Децелерации: {metrics.total_decelerations || 0}</div>
          </div>

          <button onClick={stopMonitoring}>Остановить мониторинг</button>
        </>
      )}
    </div>
  );
}

export default FetalMonitor;
```

---

## 📈 Метрики

| Метрика | Описание | Нормальные значения |
|---------|----------|---------------------|
| **STV** | Краткосрочная вариабельность (удар к удару) | > 3.0 |
| **LTV** | Долгосрочная вариабельность (минута к минуте) | 5-25 |
| **Baseline HR** | Базовый пульс плода | 110-160 уд/мин |
| **Accelerations** | Ускорения ЧСС | Хорошо, если есть |
| **Decelerations** | Замедления ЧСС | Зависит от типа |
| **Late Decelerations** | Поздние замедления | Тревожный признак |

---

## 🔧 Настройка CORS

Если фронтенд на другом домене, бэкенд уже настроен на CORS:

```go
// Разрешены любые источники (Origin)
Access-Control-Allow-Origin: *
Access-Control-Allow-Methods: GET, POST, DELETE, OPTIONS
Access-Control-Allow-Headers: Content-Type
```

---

## ⚠️ Обработка ошибок

```javascript
async function createSession(data) {
  try {
    const response = await fetch('http://localhost:8080/api/sessions', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data)
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Ошибка создания сессии');
    }

    return await response.json();
  } catch (error) {
    console.error('Ошибка:', error.message);
    alert(`Не удалось создать сессию: ${error.message}`);
    return null;
  }
}
```

**Коды ошибок:**
- `200` - OK
- `201` - Создано
- `400` - Неверный запрос
- `404` - Сессия не найдена
- `500` - Ошибка сервера

---

## 🔄 Переподключение WebSocket

```javascript
class WebSocketManager {
  constructor(sessionId) {
    this.sessionId = sessionId;
    this.reconnectAttempts = 0;
    this.maxAttempts = 5;
  }

  connect() {
    this.ws = new WebSocket(`ws://localhost:8080/ws?session_id=${this.sessionId}`);
    
    this.ws.onopen = () => {
      console.log('Подключено');
      this.reconnectAttempts = 0;
    };

    this.ws.onclose = () => {
      console.log('Отключено');
      this.reconnect();
    };

    this.ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      this.onData(data);
    };
  }

  reconnect() {
    if (this.reconnectAttempts >= this.maxAttempts) {
      console.error('Превышено количество попыток переподключения');
      return;
    }

    this.reconnectAttempts++;
    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000);
    
    console.log(`Переподключение через ${delay}мс...`);
    setTimeout(() => this.connect(), delay);
  }

  onData(data) {
    // Обработка данных
  }
}
```

---

## 📚 Дополнительно

- **База данных:** см. `DATABASE_README.md`
- **Запуск проекта:** `docker-compose up`
- **Логи сервера:** `docker-compose logs data-receiver`

