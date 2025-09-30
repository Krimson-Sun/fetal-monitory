package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"offline-service/internal/service"
	"offline-service/pkg/models"
)

type HTTPHandler struct {
	medicalService *service.MedicalService
}

func NewHTTPHandler(medicalService *service.MedicalService) *HTTPHandler {
	return &HTTPHandler{
		medicalService: medicalService,
	}
}

//func (h *HTTPHandler) UploadCSV(w http.ResponseWriter, r *http.Request) {
//	if r.Method != http.MethodPost {
//		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
//		return
//	}
//
//	// Парсим multipart form
//	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB max memory
//		http.Error(w, `{"error": "Failed to parse form: `+err.Error()+`"}`, http.StatusBadRequest)
//		return
//	}
//
//	file, header, err := r.FormFile("csv_file")
//	if err != nil {
//		http.Error(w, `{"error": "Failed to get file: `+err.Error()+`"}`, http.StatusBadRequest)
//		return
//	}
//	defer file.Close()
//
//	fmt.Printf("Received file: %s, Size: %d\n", header.Filename, header.Size)
//
//	sessionID := r.FormValue("session_id")
//	if sessionID == "" {
//		sessionID = generateSessionID()
//	}
//
//	// Обрабатываем CSV
//	response, err := h.medicalService.ProcessCSV(r.Context(), file, sessionID)
//	if err != nil {
//		errorResponse := map[string]string{
//			"error":   "Processing failed",
//			"details": err.Error(),
//		}
//		w.Header().Set("Content-Type", "application/json")
//		w.WriteHeader(http.StatusInternalServerError)
//		json.NewEncoder(w).Encode(errorResponse)
//		return
//	}
//
//	// Возвращаем полный ответ с данными
//	w.Header().Set("Content-Type", "application/json")
//	w.WriteHeader(http.StatusOK)
//	json.NewEncoder(w).Encode(response)
//}

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

// Новый endpoint для получения данных сессии
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
	return fmt.Sprintf("session_%d", time.Now().UnixNano())
}
