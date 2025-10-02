package session

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// HTTPHandler обрабатывает HTTP запросы для управления сессиями (Presentation Layer)
type HTTPHandler struct {
	manager *Manager
}

// NewHTTPHandler создает новый HTTP обработчик
func NewHTTPHandler(manager *Manager) *HTTPHandler {
	return &HTTPHandler{
		manager: manager,
	}
}

// RegisterRoutes регистрирует маршруты в роутере
func (h *HTTPHandler) RegisterRoutes(router *mux.Router) {
	api := router.PathPrefix("/api/sessions").Subrouter()

	api.HandleFunc("", h.CreateSession).Methods("POST", "OPTIONS")
	api.HandleFunc("", h.ListSessions).Methods("GET", "OPTIONS")
	api.HandleFunc("/{id}", h.GetSession).Methods("GET", "OPTIONS")
	api.HandleFunc("/{id}/stop", h.StopSession).Methods("POST", "OPTIONS")
	api.HandleFunc("/{id}/save", h.SaveSession).Methods("POST", "OPTIONS")
	api.HandleFunc("/{id}", h.DeleteSession).Methods("DELETE", "OPTIONS")
	api.HandleFunc("/{id}/metrics", h.GetSessionMetrics).Methods("GET", "OPTIONS")
	api.HandleFunc("/{id}/data", h.GetSessionData).Methods("GET", "OPTIONS")
}

// CreateSession создает новую сессию мониторинга
// @Summary Создать новую сессию
// @Description Создает новую сессию мониторинга плода с указанными параметрами
// @Tags Sessions
// @Accept json
// @Produce json
// @Param request body CreateSessionRequest true "Параметры сессии"
// @Success 201 {object} SessionResponse "Сессия успешно создана"
// @Failure 400 {object} map[string]interface{} "Неверный запрос"
// @Failure 500 {object} map[string]interface{} "Ошибка сервера"
// @Router /api/sessions [post]
func (h *HTTPHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	var req CreateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	session, err := h.manager.CreateSession(r.Context(), &req)
	if err != nil {
		log.Printf("[ERROR] Failed to create session: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	respondJSON(w, http.StatusCreated, SessionResponse{Session: session})
}

// ListSessions возвращает список сессий
// @Summary Получить список сессий
// @Description Возвращает список сессий с пагинацией
// @Tags Sessions
// @Produce json
// @Param limit query int false "Количество сессий на странице" default(50)
// @Param offset query int false "Смещение от начала списка" default(0)
// @Success 200 {object} map[string]interface{} "Список сессий"
// @Failure 500 {object} map[string]interface{} "Ошибка сервера"
// @Router /api/sessions [get]
func (h *HTTPHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	limit := getQueryInt(r, "limit", 50)
	offset := getQueryInt(r, "offset", 0)

	sessions, err := h.manager.ListSessions(r.Context(), limit, offset)
	if err != nil {
		log.Printf("[ERROR] Failed to list sessions: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to list sessions")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"sessions": sessions,
		"limit":    limit,
		"offset":   offset,
		"count":    len(sessions),
	})
}

// GetSession получает информацию о сессии
// @Summary Получить информацию о сессии
// @Description Возвращает детальную информацию о сессии и ее метрики
// @Tags Sessions
// @Produce json
// @Param id path string true "ID сессии"
// @Success 200 {object} SessionResponse "Информация о сессии"
// @Failure 404 {object} map[string]interface{} "Сессия не найдена"
// @Router /api/sessions/{id} [get]
func (h *HTTPHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	sessionID := mux.Vars(r)["id"]

	session, err := h.manager.GetSession(r.Context(), sessionID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Session not found")
		return
	}

	// Пытаемся получить метрики (может не быть для новой сессии)
	metrics, _ := h.manager.GetSessionMetrics(r.Context(), sessionID)

	respondJSON(w, http.StatusOK, SessionResponse{
		Session: session,
		Metrics: metrics,
	})
}

// StopSession останавливает сессию
// @Summary Остановить сессию
// @Description Останавливает активную сессию мониторинга
// @Tags Sessions
// @Produce json
// @Param id path string true "ID сессии"
// @Success 200 {object} map[string]interface{} "Сессия остановлена"
// @Failure 500 {object} map[string]interface{} "Ошибка остановки сессии"
// @Router /api/sessions/{id}/stop [post]
func (h *HTTPHandler) StopSession(w http.ResponseWriter, r *http.Request) {
	sessionID := mux.Vars(r)["id"]

	if err := h.manager.StopSession(r.Context(), sessionID); err != nil {
		log.Printf("[ERROR] Failed to stop session %s: %v", sessionID, err)
		respondError(w, http.StatusInternalServerError, "Failed to stop session")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":    "Session stopped successfully",
		"session_id": sessionID,
	})
}

// SaveSession сохраняет сессию в базу данных
// @Summary Сохранить сессию
// @Description Сохраняет данные сессии в PostgreSQL для долгосрочного хранения
// @Tags Sessions
// @Accept json
// @Produce json
// @Param id path string true "ID сессии"
// @Param request body SaveSessionRequest false "Дополнительные заметки"
// @Success 200 {object} map[string]interface{} "Сессия сохранена"
// @Failure 500 {object} map[string]interface{} "Ошибка сохранения сессии"
// @Router /api/sessions/{id}/save [post]
func (h *HTTPHandler) SaveSession(w http.ResponseWriter, r *http.Request) {
	sessionID := mux.Vars(r)["id"]

	var req SaveSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Не критично, если нет body
		req = SaveSessionRequest{}
	}

	if err := h.manager.SaveSession(r.Context(), sessionID, req.Notes); err != nil {
		log.Printf("[ERROR] Failed to save session %s: %v", sessionID, err)
		respondError(w, http.StatusInternalServerError, "Failed to save session")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":    "Session saved successfully",
		"session_id": sessionID,
	})
}

// DeleteSession удаляет сессию
// @Summary Удалить сессию
// @Description Удаляет сессию из Redis (не удаляет из PostgreSQL)
// @Tags Sessions
// @Produce json
// @Param id path string true "ID сессии"
// @Success 200 {object} map[string]interface{} "Сессия удалена"
// @Failure 500 {object} map[string]interface{} "Ошибка удаления сессии"
// @Router /api/sessions/{id} [delete]
func (h *HTTPHandler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	sessionID := mux.Vars(r)["id"]

	if err := h.manager.DeleteSession(r.Context(), sessionID); err != nil {
		log.Printf("[ERROR] Failed to delete session %s: %v", sessionID, err)
		respondError(w, http.StatusInternalServerError, "Failed to delete session")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":    "Session deleted successfully",
		"session_id": sessionID,
	})
}

// GetSessionMetrics получает метрики сессии
// @Summary Получить метрики сессии
// @Description Возвращает агрегированные метрики сессии (STV, LTV, ЧСС, и т.д.)
// @Tags Sessions
// @Produce json
// @Param id path string true "ID сессии"
// @Success 200 {object} SessionMetrics "Метрики сессии"
// @Failure 404 {object} map[string]interface{} "Метрики не найдены"
// @Router /api/sessions/{id}/metrics [get]
func (h *HTTPHandler) GetSessionMetrics(w http.ResponseWriter, r *http.Request) {
	sessionID := mux.Vars(r)["id"]

	metrics, err := h.manager.GetSessionMetrics(r.Context(), sessionID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Metrics not found")
		return
	}

	respondJSON(w, http.StatusOK, metrics)
}

// GetSessionData получает все данные сессии
// @Summary Получить все данные сессии
// @Description Возвращает полный набор данных сессии включая метрики, события и временные ряды
// @Tags Sessions
// @Produce json
// @Param id path string true "ID сессии"
// @Success 200 {object} SessionData "Полные данные сессии"
// @Failure 404 {object} map[string]interface{} "Данные сессии не найдены"
// @Router /api/sessions/{id}/data [get]
func (h *HTTPHandler) GetSessionData(w http.ResponseWriter, r *http.Request) {
	sessionID := mux.Vars(r)["id"]

	data, err := h.manager.GetSessionData(r.Context(), sessionID)
	if err != nil {
		log.Printf("[ERROR] Failed to get session data %s: %v", sessionID, err)
		respondError(w, http.StatusNotFound, "Session data not found")
		return
	}

	respondJSON(w, http.StatusOK, data)
}

// ===== Утилиты =====

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("[ERROR] Failed to encode JSON response: %v", err)
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]interface{}{
		"error":  message,
		"status": status,
	})
}

func getQueryInt(r *http.Request, key string, defaultValue int) int {
	valueStr := r.URL.Query().Get(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}
