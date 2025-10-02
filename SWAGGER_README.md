# 📚 Swagger API Документация

Swagger UI теперь доступен для всех HTTP API в системе мониторинга плода!

---

## 🚀 Быстрый старт

### Receiver Service (Real-time мониторинг)

**Swagger UI:** http://localhost:8080/swagger/index.html

Этот сервис предоставляет:
- ✅ Управление сессиями мониторинга (`/api/sessions`)
- ✅ Real-time данные через WebSocket (`/ws`)
- ✅ Метрики и аналитику

**Основные endpoints:**
- `POST /api/sessions` - Создать новую сессию
- `GET /api/sessions` - Список всех сессий
- `GET /api/sessions/{id}` - Получить информацию о сессии
- `GET /api/sessions/{id}/metrics` - Получить метрики сессии
- `GET /api/sessions/{id}/data` - Получить все данные сессии
- `POST /api/sessions/{id}/stop` - Остановить сессию
- `POST /api/sessions/{id}/save` - Сохранить сессию в БД
- `DELETE /api/sessions/{id}` - Удалить сессию

---

### Offline Service (Анализ CSV файлов)

**Swagger UI:** http://localhost:8081/swagger/index.html

Этот сервис предоставляет:
- ✅ Загрузку CSV файлов с FHR и UC данными
- ✅ Фильтрацию и извлечение признаков
- ✅ ML предсказание

**Основные endpoints:**
- `POST /upload` - Загрузить CSV файлы для анализа
- `POST /decision` - Сохранить или отклонить результаты
- `GET /session` - Получить данные сессии

---

## 📖 Как использовать Swagger UI

### 1. Запустите сервисы

```bash
# Запустить все сервисы через Docker
docker-compose up

# Или запустить локально:

# Receiver
cd receiver
go run cmd/receiver/main.go

# Offline Service
cd offline-service
go run cmd/server/main.go
```

### 2. Откройте Swagger UI в браузере

- **Receiver:** http://localhost:8080/swagger/index.html
- **Offline Service:** http://localhost:8081/swagger/index.html

### 3. Тестируйте API прямо в браузере

Swagger UI позволяет:
- ✅ Просматривать все доступные endpoints
- ✅ Видеть структуру запросов и ответов
- ✅ Тестировать API без Postman
- ✅ Генерировать примеры кода (curl, JavaScript, Python и др.)

---

## 🎯 Примеры использования

### Пример 1: Создать новую сессию (Receiver)

1. Откройте http://localhost:8080/swagger/index.html
2. Найдите `POST /api/sessions`
3. Нажмите "Try it out"
4. Вставьте JSON:

```json
{
  "patient_id": "patient-001",
  "doctor_id": "dr-ivanov",
  "notes": "Плановое обследование"
}
```

5. Нажмите "Execute"
6. Получите `session_id` для дальнейшего использования

### Пример 2: Загрузить CSV файлы (Offline Service)

1. Откройте http://localhost:8081/swagger/index.html
2. Найдите `POST /upload`
3. Нажмите "Try it out"
4. Выберите файлы:
   - `bpm_file`: CSV с данными FHR
   - `uc_file`: CSV с данными UC
5. Нажмите "Execute"
6. Получите результаты анализа

---

## 🔧 Настройка и обновление документации

### Обновить Swagger документацию после изменений

Если вы изменили API endpoints или их аннотации:

```bash
# Для Receiver
cd receiver
swag init -g cmd/receiver/main.go -o docs

# Для Offline Service
cd offline-service
swag init -g cmd/server/main.go -o docs
```

### Добавить новые endpoints

1. Добавьте Swagger аннотации к вашему handler:

```go
// GetExample пример endpoint
// @Summary Пример endpoint
// @Description Описание того, что делает endpoint
// @Tags Examples
// @Produce json
// @Param id path string true "ID объекта"
// @Success 200 {object} YourResponseType
// @Failure 404 {object} map[string]interface{}
// @Router /example/{id} [get]
func (h *Handler) GetExample(w http.ResponseWriter, r *http.Request) {
    // ...
}
```

2. Перегенерируйте документацию:

```bash
swag init -g cmd/receiver/main.go -o docs
```

3. Перезапустите сервис

---

## 📋 Swagger Аннотации - Шпаргалка

### Основные теги:

- `@Summary` - Краткое описание endpoint
- `@Description` - Детальное описание
- `@Tags` - Группировка endpoints
- `@Accept` - Тип контента запроса (json, multipart/form-data)
- `@Produce` - Тип контента ответа (json)
- `@Param` - Параметры запроса
- `@Success` - Успешный ответ
- `@Failure` - Ответ с ошибкой
- `@Router` - Путь и HTTP метод

### Типы параметров:

- `path` - Параметр в URL (`/sessions/{id}`)
- `query` - Query параметр (`?limit=50`)
- `body` - JSON в теле запроса
- `formData` - Данные формы (multipart)
- `header` - HTTP заголовок

---

## 🌐 Интеграция с фронтендом

### Генерация клиента из Swagger

Вы можете сгенерировать TypeScript/JavaScript клиент для фронтенда:

```bash
# Установить swagger-codegen
npm install -g @openapitools/openapi-generator-cli

# Сгенерировать TypeScript клиент
openapi-generator-cli generate \
  -i http://localhost:8080/swagger/doc.json \
  -g typescript-axios \
  -o ./frontend/src/api
```

### Прямое использование OpenAPI спецификации

OpenAPI spec доступны по адресам:
- **Receiver:** http://localhost:8080/swagger/doc.json
- **Offline Service:** http://localhost:8081/swagger/doc.json

---

## 🔍 Troubleshooting

### Swagger UI не открывается

1. Проверьте, что сервис запущен:
   ```bash
   curl http://localhost:8080/health
   ```

2. Проверьте порты в конфигурации

3. Убедитесь, что `docs/` папка создана:
   ```bash
   ls receiver/docs/
   ls offline-service/docs/
   ```

### Документация не обновляется

1. Перегенерируйте документацию:
   ```bash
   swag init -g cmd/receiver/main.go -o docs
   ```

2. Перезапустите сервис

3. Очистите кеш браузера (Ctrl+Shift+R)

---

## 📚 Дополнительные ресурсы

- [Swagger Official Docs](https://swagger.io/docs/)
- [Swaggo GitHub](https://github.com/swaggo/swag)
- [OpenAPI Specification](https://swagger.io/specification/)
- [Frontend Integration Guide](./FRONTEND_README.md)

---

## 🎉 Готово!

Теперь ваш API полностью документирован и готов к использованию! 

Откройте Swagger UI и начните тестировать:
- **Receiver:** http://localhost:8080/swagger/index.html
- **Offline Service:** http://localhost:8081/swagger/index.html

Удачи! 🚀

