# 🗄️ База данных - Документация

Простое руководство по работе с базой данных системы мониторинга плода.

---

## 🔌 Подключение

```
Host: localhost
Port: 5432
Database: fetal_monitor
User: fetal_user
Password: fetal_pass
```

**Строка подключения:**
```
postgres://fetal_user:fetal_pass@localhost:5432/fetal_monitor?sslmode=disable
```

---

## 📊 Таблицы

### 1. `sessions` - Сессии мониторинга

Основная таблица с информацией о сессиях.

```sql
CREATE TABLE sessions (
    id VARCHAR(64) PRIMARY KEY,
    status VARCHAR(20) NOT NULL,
    started_at TIMESTAMP NOT NULL,
    stopped_at TIMESTAMP,
    saved_at TIMESTAMP,
    total_duration_ms BIGINT DEFAULT 0,
    total_data_points BIGINT DEFAULT 0,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT NOW()
);
```

**Поля:**
- `id` - ID сессии (UUID)
- `status` - Статус: `ACTIVE`, `STOPPED`, `SAVED`
- `started_at` - Время начала
- `stopped_at` - Время остановки
- `saved_at` - Время сохранения в БД
- `total_duration_ms` - Общая длительность (миллисекунды)
- `total_data_points` - Количество точек данных
- `metadata` - JSON с доп. данными (patient_id, doctor_id, notes и т.д.)

**Примеры запросов:**

```sql
-- Получить все активные сессии
SELECT * FROM sessions WHERE status = 'ACTIVE';

-- Получить сессии пациента
SELECT * FROM sessions 
WHERE metadata->>'patient_id' = 'patient-001'
ORDER BY started_at DESC;

-- Средняя длительность сессий
SELECT AVG(total_duration_ms / 1000.0 / 60.0) as avg_minutes
FROM sessions WHERE status = 'SAVED';
```

---

### 2. `session_metrics` - Метрики сессии

Агрегированные медицинские показатели.

```sql
CREATE TABLE session_metrics (
    session_id VARCHAR(64) PRIMARY KEY REFERENCES sessions(id) ON DELETE CASCADE,
    stv DOUBLE PRECISION DEFAULT 0,
    ltv DOUBLE PRECISION DEFAULT 0,
    baseline_heart_rate DOUBLE PRECISION DEFAULT 0,
    total_accelerations INTEGER DEFAULT 0,
    total_decelerations INTEGER DEFAULT 0,
    late_decelerations INTEGER DEFAULT 0,
    late_deceleration_ratio DOUBLE PRECISION DEFAULT 0,
    total_contractions INTEGER DEFAULT 0,
    accel_decel_ratio DOUBLE PRECISION DEFAULT 0,
    stv_trend DOUBLE PRECISION DEFAULT 0,
    bpm_trend DOUBLE PRECISION DEFAULT 0,
    data_points INTEGER DEFAULT 0,
    time_span_sec DOUBLE PRECISION DEFAULT 0,
    updated_at TIMESTAMP DEFAULT NOW()
);
```

**Основные метрики:**
- `stv` - Краткосрочная вариабельность (норма > 3.0)
- `ltv` - Долгосрочная вариабельность (норма 5-25)
- `baseline_heart_rate` - Базовый пульс плода (норма 110-160 уд/мин)
- `total_accelerations` - Количество ускорений ЧСС
- `total_decelerations` - Количество замедлений ЧСС
- `late_decelerations` - Количество поздних замедлений (тревожно)
- `total_contractions` - Количество сокращений матки

**Примеры запросов:**

```sql
-- Получить метрики сессии
SELECT * FROM session_metrics WHERE session_id = 'abc-123';

-- Найти сессии с тревожными показателями
SELECT s.id, s.metadata->>'patient_id' as patient_id,
       m.stv, m.late_decelerations
FROM sessions s
JOIN session_metrics m ON s.id = m.session_id
WHERE m.stv < 3.0 OR m.late_decelerations > 0;
```

---

### 3. `session_events` - События

Обнаруженные клинические события.

```sql
CREATE TABLE session_events (
    id BIGSERIAL PRIMARY KEY,
    session_id VARCHAR(64) NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    event_type VARCHAR(20) NOT NULL,
    start_time DOUBLE PRECISION NOT NULL,
    end_time DOUBLE PRECISION NOT NULL,
    duration DOUBLE PRECISION NOT NULL,
    amplitude DOUBLE PRECISION NOT NULL,
    is_late BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW()
);
```

**Типы событий:**
- `acceleration` - Ускорение ЧСС
- `deceleration` - Замедление ЧСС
- `contraction` - Сокращение матки

**Поля:**
- `start_time` - Время начала (секунды от начала сессии)
- `end_time` - Время окончания
- `duration` - Длительность события
- `amplitude` - Амплитуда (уд/мин для ЧСС)
- `is_late` - Позднее замедление (только для deceleration)

**Примеры запросов:**

```sql
-- Получить все события сессии
SELECT event_type, start_time, duration, amplitude
FROM session_events
WHERE session_id = 'abc-123'
ORDER BY start_time;

-- Подсчитать события по типам
SELECT event_type, COUNT(*) as count
FROM session_events
WHERE session_id = 'abc-123'
GROUP BY event_type;

-- Найти поздние замедления
SELECT * FROM session_events
WHERE event_type = 'deceleration' AND is_late = TRUE;
```

---

### 4. `session_timeseries` - Временные ряды

Временные ряды метрик (STV, LTV).

```sql
CREATE TABLE session_timeseries (
    id BIGSERIAL PRIMARY KEY,
    session_id VARCHAR(64) NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    metric_type VARCHAR(20) NOT NULL,
    time_index INTEGER NOT NULL,
    value DOUBLE PRECISION NOT NULL,
    window_duration DOUBLE PRECISION NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
```

**Типы метрик:**
- `stv` - Временной ряд STV
- `ltv` - Временной ряд LTV

**Примеры запросов:**

```sql
-- Получить временной ряд STV
SELECT time_index, value FROM session_timeseries
WHERE session_id = 'abc-123' AND metric_type = 'stv'
ORDER BY time_index;

-- Рассчитать тренд STV
SELECT 
    session_id,
    REGR_SLOPE(value, time_index) as trend,
    AVG(value) as avg_stv
FROM session_timeseries
WHERE session_id = 'abc-123' AND metric_type = 'stv'
GROUP BY session_id;
```

---

### 5. `session_raw_data` - Сырые данные

Отфильтрованные данные для воспроизведения и анализа.

```sql
CREATE TABLE session_raw_data (
    id BIGSERIAL PRIMARY KEY,
    session_id VARCHAR(64) NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    batch_ts_ms BIGINT NOT NULL,
    metric_type VARCHAR(10) NOT NULL,
    data JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
```

**Типы данных:**
- `FHR` - Частота сердцебиения плода
- `UC` - Сокращения матки

**Формат данных (JSONB):**
```json
[
  {"time_sec": 0.0, "value": 138.5},
  {"time_sec": 0.25, "value": 139.2},
  {"time_sec": 0.5, "value": 138.8}
]
```

**Примеры запросов:**

```sql
-- Получить сырые данные FHR
SELECT batch_ts_ms, data FROM session_raw_data
WHERE session_id = 'abc-123' AND metric_type = 'FHR'
ORDER BY batch_ts_ms;

-- Извлечь первое значение из JSONB
SELECT 
    batch_ts_ms,
    data->0->>'value' as first_value
FROM session_raw_data
WHERE session_id = 'abc-123' AND metric_type = 'FHR';
```

---

## 🔗 Связи между таблицами

```
sessions (1)
    ├── (1) session_metrics
    ├── (*) session_events
    ├── (*) session_timeseries
    └── (*) session_raw_data
```

При удалении сессии автоматически удаляются все связанные данные (`ON DELETE CASCADE`).

---

## 📊 Полезные запросы

### Полная информация о сессии

```sql
SELECT 
    s.id,
    s.status,
    s.started_at,
    s.total_duration_ms / 1000.0 / 60.0 as duration_minutes,
    s.metadata->>'patient_id' as patient_id,
    s.metadata->>'doctor_id' as doctor_id,
    
    m.stv,
    m.ltv,
    m.baseline_heart_rate,
    m.total_accelerations,
    m.total_decelerations,
    m.total_contractions
    
FROM sessions s
LEFT JOIN session_metrics m ON s.id = m.session_id
WHERE s.id = 'abc-123';
```

### Статистика по всем сессиям

```sql
SELECT 
    COUNT(*) as total,
    COUNT(*) FILTER (WHERE status = 'ACTIVE') as active,
    COUNT(*) FILTER (WHERE status = 'SAVED') as saved,
    AVG(total_duration_ms / 1000.0 / 60.0) as avg_duration_minutes
FROM sessions;
```

### Сессии за последние 7 дней

```sql
SELECT 
    DATE(started_at) as date,
    COUNT(*) as count
FROM sessions
WHERE started_at >= NOW() - INTERVAL '7 days'
GROUP BY DATE(started_at)
ORDER BY date DESC;
```

---

## 🔧 Примеры интеграции

### Python (psycopg2)

```python
import psycopg2

# Подключение
conn = psycopg2.connect(
    host="localhost",
    port=5432,
    database="fetal_monitor",
    user="fetal_user",
    password="fetal_pass"
)

# Получить сессию
cur = conn.cursor()
cur.execute("""
    SELECT s.id, s.metadata, m.stv, m.ltv, m.baseline_heart_rate
    FROM sessions s
    LEFT JOIN session_metrics m ON s.id = m.session_id
    WHERE s.id = %s
""", ('abc-123',))

row = cur.fetchone()
print(f"Session: {row[0]}, STV: {row[2]}, LTV: {row[3]}")

conn.close()
```

### Node.js (pg)

```javascript
const { Pool } = require('pg');

const pool = new Pool({
  host: 'localhost',
  port: 5432,
  database: 'fetal_monitor',
  user: 'fetal_user',
  password: 'fetal_pass'
});

// Получить события сессии
async function getEvents(sessionId) {
  const result = await pool.query(
    'SELECT * FROM session_events WHERE session_id = $1 ORDER BY start_time',
    [sessionId]
  );
  return result.rows;
}

getEvents('abc-123').then(events => {
  console.log('Events:', events);
});
```

### Go (database/sql)

```go
package main

import (
    "database/sql"
    _ "github.com/lib/pq"
)

func main() {
    db, err := sql.Open("postgres", 
        "postgres://fetal_user:fetal_pass@localhost:5432/fetal_monitor?sslmode=disable")
    if err != nil {
        panic(err)
    }
    defer db.Close()

    var stv, ltv float64
    err = db.QueryRow(
        "SELECT stv, ltv FROM session_metrics WHERE session_id = $1",
        "abc-123").Scan(&stv, &ltv)
    
    if err != nil {
        panic(err)
    }
    
    println("STV:", stv, "LTV:", ltv)
}
```

---

## 🔒 Пользователь только для чтения

Для аналитики можно создать пользователя с правами только на чтение:

```sql
-- Создать пользователя
CREATE USER analyst WITH PASSWORD 'secure_password';

-- Дать права
GRANT CONNECT ON DATABASE fetal_monitor TO analyst;
GRANT USAGE ON SCHEMA public TO analyst;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO analyst;
```

---

## 💾 Backup и Restore

### Backup
```bash
# Полный backup
docker exec -it fetal-monitory-postgres-1 pg_dump -U fetal_user fetal_monitor > backup.sql

# Только схема
docker exec -it fetal-monitory-postgres-1 pg_dump -U fetal_user --schema-only fetal_monitor > schema.sql
```

### Restore
```bash
# Восстановить из backup
docker exec -i fetal-monitory-postgres-1 psql -U fetal_user fetal_monitor < backup.sql
```

---

## 🔍 Проверка данных

### Подключиться к БД
```bash
docker exec -it fetal-monitory-postgres-1 psql -U fetal_user -d fetal_monitor
```

### Полезные команды psql
```sql
\dt              -- Список таблиц
\d sessions      -- Структура таблицы sessions
\q               -- Выход
```

---

## 📚 Дополнительно

- **API для фронтенда:** см. `FRONTEND_README.md`
- **Запуск проекта:** `docker-compose up`
- **Логи БД:** `docker-compose logs postgres`

