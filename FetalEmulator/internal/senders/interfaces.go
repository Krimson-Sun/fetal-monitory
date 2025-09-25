package senders

import (
	"errors"
	"fetal-emulator/internal/models"
	"time"
)

// Ошибки отправителей
var (
	ErrSendFailed       = errors.New("failed to send data")
	ErrConnectionFailed = errors.New("connection failed")
	ErrInvalidData      = errors.New("invalid data format")
)

// DataSender интерфейс для отправки данных
type DataSender interface {
	// Send отправляет одну точку данных
	Send(data models.DataPoint) error

	// Validate проверяет готовность отправителя
	Validate() error

	// Close освобождает ресурсы
	Close() error
}

// BatchSender интерфейс для пакетной отправки
type BatchSender interface {
	DataSender

	// SendBatch отправляет несколько точек данных
	SendBatch(data []models.DataPoint) error

	// GetBatchSize возвращает оптимальный размер пакета
	GetBatchSize() int
}

// BufferedSender интерфейс для буферизированной отправки
type BufferedSender interface {
	DataSender

	// Flush принудительно отправляет данные из буфера
	Flush() error

	// GetBufferSize возвращает текущий размер буфера
	GetBufferSize() int
}

// StatusSender интерфейс для отправителей со статусом
type StatusSender interface {
	DataSender

	// GetStatus возвращает статус соединения
	GetStatus() SenderStatus

	// GetMetrics возвращает метрики отправки
	GetMetrics() SenderMetrics
}

// SenderStatus представляет статус отправителя
type SenderStatus struct {
	Connected    bool
	LastError    error
	LastActivity time.Time
	TotalSent    int
}

// SenderMetrics содержит метрики отправки
type SenderMetrics struct {
	TotalSent        int
	TotalFailed      int
	AverageSendTime  time.Duration
	LastSendTime     time.Duration
	BytesTransferred int64
}
