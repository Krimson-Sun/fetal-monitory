package websocket

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	featureextractorv1 "github.com/Krimson/fetal-monitory/proto/feature_extractor"
	"github.com/gorilla/websocket"
)

// Hub управляет WebSocket соединениями
type Hub struct {
	// Зарегистрированные клиенты
	clients map[*Client]bool

	// Канал для регистрации клиентов
	register chan *Client

	// Канал для отмены регистрации клиентов
	unregister chan *Client

	// Канал для входящих сообщений от клиентов
	broadcast chan []byte

	// Мютекс для безопасной работы с картой клиентов
	mu sync.RWMutex

	// Последние предикты для каждой сессии (session_id -> prediction)
	lastPredictions map[string]float64
	predMu          sync.RWMutex
}

// Client представляет WebSocket клиента
type Client struct {
	hub *Hub

	// WebSocket соединение
	conn *websocket.Conn

	// Буферизованный канал исходящих сообщений
	send chan []byte

	// ID сессии для фильтрации данных
	sessionID string
}

// ProcessedData представляет данные для отправки на фронтенд в новом формате
type ProcessedData struct {
	Message    string      `json:"message"`
	Prediction float64     `json:"prediction"` // Заглушка для ML модели
	Records    RecordsData `json:"records"`
	SessionID  string      `json:"session_id"`
	Status     string      `json:"status"`
}

// RecordsData содержит все медицинские метрики
type RecordsData struct {
	STV                   float64           `json:"stv"`
	LTV                   float64           `json:"ltv"`
	BaselineHeartRate     float64           `json:"baseline_heart_rate"`
	Accelerations         []Acceleration    `json:"accelerations"`
	Decelerations         []Deceleration    `json:"decelerations"`
	Contractions          []Contraction     `json:"contractions"`
	STVs                  []float64         `json:"stvs"`
	STVsWindowDuration    float64           `json:"stvs_window_duration"`
	LTVs                  []float64         `json:"ltvs"`
	LTVsWindowDuration    float64           `json:"ltvs_window_duration"`
	TotalDecelerations    int32             `json:"total_decelerations"`
	LateDecelerations     int32             `json:"late_decelerations"`
	LateDecelerationRatio float64           `json:"late_deceleration_ratio"`
	TotalAccelerations    int32             `json:"total_accelerations"`
	AccelDecelRatio       float64           `json:"accel_decel_ratio"`
	TotalContractions     int32             `json:"total_contractions"`
	STVTrend              float64           `json:"stv_trend"`
	BPMTrend              float64           `json:"bpm_trend"`
	DataPoints            int32             `json:"data_points"`
	TimeSpanSec           float64           `json:"time_span_sec"`
	FilteredBPMBatch      FilteredBatchData `json:"filtered_bpm_batch"`
	FilteredUterusBatch   FilteredBatchData `json:"filtered_uterus_batch"`
}

// FilteredBatchData представляет скользящее окно отфильтрованных данных
type FilteredBatchData struct {
	TimeSec []float64 `json:"time_sec"`
	Value   []float64 `json:"value"`
}

type DataPoint struct {
	TimeSec float64 `json:"time_sec"`
	Value   float64 `json:"value"`
}

type Acceleration struct {
	Start     float64 `json:"start"`
	End       float64 `json:"end"`
	Duration  float64 `json:"duration"`
	Amplitude float64 `json:"amplitude"`
}

type Deceleration struct {
	Start     float64 `json:"start"`
	End       float64 `json:"end"`
	Duration  float64 `json:"duration"`
	Amplitude float64 `json:"amplitude"`
	IsLate    bool    `json:"is_late"`
}

type Contraction struct {
	Start     float64 `json:"start"`
	End       float64 `json:"end"`
	Duration  float64 `json:"duration"`
	Amplitude float64 `json:"amplitude"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// В продакшене следует проверять домен
		return true
	},
}

// NewHub создает новый Hub
func NewHub() *Hub {
	return &Hub{
		clients:         make(map[*Client]bool),
		register:        make(chan *Client),
		unregister:      make(chan *Client),
		broadcast:       make(chan []byte),
		lastPredictions: make(map[string]float64),
	}
}

// Run запускает Hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("[WEBSOCKET] Client registered: %p, session: %s", client, client.sessionID)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("[WEBSOCKET] Client unregistered: %p", client)

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					delete(h.clients, client)
					close(client.send)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastProcessedData отправляет обработанные данные всем клиентам
func (h *Hub) BroadcastProcessedData(response *featureextractorv1.ProcessBatchResponse) {
	data := h.convertResponseToProcessedData(response)

	message, err := json.Marshal(data)
	if err != nil {
		log.Printf("[ERROR] Failed to marshal processed data: %v", err)
		return
	}

	select {
	case h.broadcast <- message:
	default:
		log.Printf("[WARN] Broadcast channel full, dropping message")
	}
}

// GetLastPrediction возвращает последнее предсказание для сессии
func (h *Hub) GetLastPrediction(sessionID string) float64 {
	h.predMu.RLock()
	defer h.predMu.RUnlock()
	return h.lastPredictions[sessionID]
}

// UpdatePrediction обновляет последнее предсказание для сессии
func (h *Hub) UpdatePrediction(sessionID string, prediction float64) {
	h.predMu.Lock()
	defer h.predMu.Unlock()
	h.lastPredictions[sessionID] = prediction
	log.Printf("[WEBSOCKET] Updated prediction for session %s: %.4f", sessionID, prediction)
}

// convertResponseToProcessedData конвертирует gRPC ответ в JSON структуру нового формата
func (h *Hub) convertResponseToProcessedData(response *featureextractorv1.ProcessBatchResponse) *ProcessedData {
	// Конвертируем отфильтрованные BPM данные в формат {time_sec: [], value: []}
	bpmTimeSec := make([]float64, 0, len(response.FilteredBpmBatch))
	bpmValue := make([]float64, 0, len(response.FilteredBpmBatch))
	for _, dp := range response.FilteredBpmBatch {
		bpmTimeSec = append(bpmTimeSec, dp.TimeSec)
		bpmValue = append(bpmValue, dp.Value)
	}

	// Конвертируем отфильтрованные Uterus данные
	uterusTimeSec := make([]float64, 0, len(response.FilteredUterusBatch))
	uterusValue := make([]float64, 0, len(response.FilteredUterusBatch))
	for _, dp := range response.FilteredUterusBatch {
		uterusTimeSec = append(uterusTimeSec, dp.TimeSec)
		uterusValue = append(uterusValue, dp.Value)
	}

	// Конвертируем события
	accelerations := make([]Acceleration, 0, len(response.Accelerations))
	for _, acc := range response.Accelerations {
		accelerations = append(accelerations, Acceleration{
			Start:     acc.Start,
			End:       acc.End,
			Duration:  acc.Duration,
			Amplitude: acc.Amplitude,
		})
	}

	decelerations := make([]Deceleration, 0, len(response.Decelerations))
	for _, dec := range response.Decelerations {
		decelerations = append(decelerations, Deceleration{
			Start:     dec.Start,
			End:       dec.End,
			Duration:  dec.Duration,
			Amplitude: dec.Amplitude,
			IsLate:    dec.IsLate,
		})
	}

	contractions := make([]Contraction, 0, len(response.Contractions))
	for _, cont := range response.Contractions {
		contractions = append(contractions, Contraction{
			Start:     cont.Start,
			End:       cont.End,
			Duration:  cont.Duration,
			Amplitude: cont.Amplitude,
		})
	}

	// Создаем структуру данных в новом формате
	// Получаем последнее предсказание для этой сессии
	prediction := h.GetLastPrediction(response.SessionId)

	data := &ProcessedData{
		Message:    "Done",
		Prediction: prediction, // Реальный предикт из ML сервиса
		SessionID:  response.SessionId,
		Status:     "processed",
		Records: RecordsData{
			STV:                   response.Stv,
			LTV:                   response.Ltv,
			BaselineHeartRate:     response.BaselineHeartRate,
			Accelerations:         accelerations,
			Decelerations:         decelerations,
			Contractions:          contractions,
			STVs:                  response.Stvs,
			STVsWindowDuration:    response.StvsWindowDuration,
			LTVs:                  response.Ltvs,
			LTVsWindowDuration:    response.LtvsWindowDuration,
			TotalDecelerations:    response.TotalDecelerations,
			LateDecelerations:     response.LateDecelerations,
			LateDecelerationRatio: response.LateDecelerationRatio,
			TotalAccelerations:    response.TotalAccelerations,
			AccelDecelRatio:       response.AccelDecelRatio,
			TotalContractions:     response.TotalContractions,
			STVTrend:              response.StvTrend,
			BPMTrend:              response.BpmTrend,
			DataPoints:            response.DataPoints,
			TimeSpanSec:           response.TimeSpanSec,
			FilteredBPMBatch: FilteredBatchData{
				TimeSec: bpmTimeSec,
				Value:   bpmValue,
			},
			FilteredUterusBatch: FilteredBatchData{
				TimeSec: uterusTimeSec,
				Value:   uterusValue,
			},
		},
	}

	return data
}

// HandleWebSocket обрабатывает WebSocket соединения
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[ERROR] Failed to upgrade connection: %v", err)
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		sessionID = "default"
	}

	client := &Client{
		hub:       h,
		conn:      conn,
		send:      make(chan []byte, 256),
		sessionID: sessionID,
	}

	client.hub.register <- client

	// Запускаем горутины для клиента
	go client.writePump()
	go client.readPump()
}

// readPump обрабатывает входящие сообщения от клиента
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[ERROR] WebSocket error: %v", err)
			}
			break
		}
	}
}

// writePump отправляет сообщения клиенту
func (c *Client) writePump() {
	defer c.conn.Close()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("[ERROR] Failed to write message: %v", err)
				return
			}
		}
	}
}

// ProcessedDataConsumer обрабатывает обработанные данные из feature extractor
func (h *Hub) ProcessedDataConsumer(ctx context.Context, processedBatchChan <-chan *featureextractorv1.ProcessBatchResponse) {
	for {
		select {
		case <-ctx.Done():
			return
		case response, ok := <-processedBatchChan:
			if !ok {
				return
			}
			h.BroadcastProcessedData(response)
		}
	}
}

// PredictionConsumer обрабатывает предсказания из ML сервиса
func (h *Hub) PredictionConsumer(ctx context.Context, predictionChan <-chan interface{}) {
	for {
		select {
		case <-ctx.Done():
			return
		case prediction, ok := <-predictionChan:
			if !ok {
				return
			}
			// Обновляем последнее предсказание для сессии
			// prediction должен иметь поля SessionId и Prediction
			// Используем type assertion для доступа к полям
			if pred, ok := prediction.(interface {
				GetSessionId() string
				GetPrediction() float64
			}); ok {
				h.UpdatePrediction(pred.GetSessionId(), pred.GetPrediction())
			}
		}
	}
}
