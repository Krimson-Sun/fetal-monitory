package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"offline-service/internal/service"
	"offline-service/pkg/models"

	"github.com/google/uuid"
)

type HTTPHandler struct {
	medicalService *service.MedicalService
}

func NewHTTPHandler(medicalService *service.MedicalService) *HTTPHandler {
	return &HTTPHandler{
		medicalService: medicalService,
	}
}

// UploadDualCSV загружает и обрабатывает CSV файлы с FHR и UC данными
// @Summary Загрузить CSV файлы для анализа
// @Description Загружает два CSV файла (BPM и UC), выполняет фильтрацию, извлечение признаков и ML предсказание
// @Tags Offline Analysis
// @Accept multipart/form-data
// @Produce json
// @Param bpm_file formData file true "CSV файл с данными FHR (сердцебиение плода)"
// @Param uc_file formData file true "CSV файл с данными UC (маточные сокращения)"
// @Param session_id formData string false "ID сессии (генерируется автоматически если не указан)"
// @Success 200 {object} models.UploadResponse "Результат анализа"
// @Failure 400 {object} map[string]string "Неверный запрос"
// @Failure 500 {object} map[string]string "Ошибка обработки"
// @Router /upload [post]
func (h *HTTPHandler) UploadDualCSV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Парсим multipart form
	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB max memory
		http.Error(w, `{"error": "Failed to parse form: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	// Получаем BPM файл
	bpmFile, bpmHeader, err := r.FormFile("bpm_file")
	if err != nil {
		http.Error(w, `{"error": "Failed to get BPM file: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}
	defer bpmFile.Close()

	// Получаем UC файл
	ucFile, ucHeader, err := r.FormFile("uc_file")
	if err != nil {
		http.Error(w, `{"error": "Failed to get UC file: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}
	defer ucFile.Close()

	sessionID := r.FormValue("session_id")
	if sessionID == "" {
		sessionID = generateSessionID()
	}

	fmt.Printf("Received BPM file: %s, UC file: %s, Session: %s\n",
		bpmHeader.Filename, ucHeader.Filename, sessionID)

	// Обрабатываем оба файла
	response, err := h.medicalService.ProcessDualCSV(r.Context(), bpmFile, ucFile, sessionID)
	if err != nil {
		errorResponse := map[string]string{
			"error":   "Processing failed",
			"details": err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	// Возвращаем полный ответ с данными
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// HandleDecision обрабатывает решение о сохранении данных сессии
// @Summary Принять решение о сохранении
// @Description Сохраняет или отклоняет сохранение данных сессии в базу данных
// @Tags Offline Analysis
// @Accept json
// @Produce json
// @Param request body models.SaveDecision true "Решение о сохранении"
// @Success 200 {object} models.DecisionResponse "Результат операции"
// @Failure 400 {object} map[string]string "Неверный запрос"
// @Failure 500 {object} map[string]string "Ошибка обработки"
// @Router /decision [post]
func (h *HTTPHandler) HandleDecision(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var decision models.SaveDecision
	if err := json.NewDecoder(r.Body).Decode(&decision); err != nil {
		errorResponse := map[string]string{
			"error":   "Invalid request body",
			"details": err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	if decision.SessionID == "" {
		errorResponse := map[string]string{
			"error": "session_id is required",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	// Обрабатываем решение
	response, err := h.medicalService.HandleDecision(r.Context(), &decision)
	if err != nil {
		errorResponse := map[string]string{
			"error":   "Decision processing failed",
			"details": err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	// Возвращаем ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetSessionData получает данные сессии
// @Summary Получить данные сессии
// @Description Возвращает информацию о сессии (пока заглушка, нужно использовать /upload для получения данных)
// @Tags Offline Analysis
// @Produce json
// @Param session_id query string true "ID сессии"
// @Success 200 {object} map[string]interface{} "Информация о сессии"
// @Failure 400 {object} map[string]string "Неверный запрос"
// @Router /session [get]
func (h *HTTPHandler) GetSessionData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		errorResponse := map[string]string{
			"error": "session_id parameter is required",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	// Здесь можно добавить метод в сервисе для получения данных сессии
	// Пока просто возвращаем сообщение
	response := map[string]interface{}{
		"status":     "success",
		"session_id": sessionID,
		"message":    "Use /upload to get session data",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func generateSessionID() string {
	return uuid.New().String()
}
