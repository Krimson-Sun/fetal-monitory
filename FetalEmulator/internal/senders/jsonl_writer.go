package senders

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"fetal-emulator/internal/models"
)

// JSONLWriter отвечает за запись данных в JSONL формат
type JSONLWriter struct {
	writer    *bufio.Writer
	file      *os.File
	mu        sync.Mutex
	filePath  string
	autoFlush bool
	stats     *WriteStats
}

// WriteStats содержит статистику записи
type WriteStats struct {
	TotalLines       int64         `json:"total_lines"`
	TotalBytes       int64         `json:"total_bytes"`
	LastWriteTime    time.Time     `json:"last_write_time"`
	AverageWriteTime time.Duration `json:"average_write_time"`
	ErrorsCount      int64         `json:"errors_count"`
	mu               sync.RWMutex
}

// JSONLConfig конфигурация JSONL писателя
type JSONLConfig struct {
	FilePath   string      `json:"file_path"`
	AutoFlush  bool        `json:"auto_flush"`
	BufferSize int         `json:"buffer_size"`
	CreateDir  bool        `json:"create_dir"`
	FilePerm   os.FileMode `json:"file_perm"`
}

// NewJSONLWriter создает новый JSONL писатель
func NewJSONLWriter(config JSONLConfig) (*JSONLWriter, error) {
	// Создаем директорию если нужно
	if config.CreateDir {
		dir := filepath.Dir(config.FilePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Открываем файл
	file, err := os.OpenFile(config.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, config.FilePerm)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	// Создаем буферизированного писателя
	var writer *bufio.Writer
	if config.BufferSize > 0 {
		writer = bufio.NewWriterSize(file, config.BufferSize)
	} else {
		writer = bufio.NewWriter(file)
	}

	return &JSONLWriter{
		writer:    writer,
		file:      file,
		filePath:  config.FilePath,
		autoFlush: config.AutoFlush,
		stats: &WriteStats{
			LastWriteTime: time.Now(),
		},
	}, nil
}

// WriteDataPoint записывает одну точку данных в формате JSONL
func (j *JSONLWriter) WriteDataPoint(data models.DataPoint) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	startTime := time.Now()

	// Валидируем данные перед записью
	if err := data.Validate(); err != nil {
		j.recordError()
		return fmt.Errorf("data validation failed: %w", err)
	}

	// Конвертируем в JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		j.recordError()
		return fmt.Errorf("JSON marshaling failed: %w", err)
	}

	// Записываем в файл с новой строкой
	if _, err := j.writer.Write(jsonData); err != nil {
		j.recordError()
		return fmt.Errorf("write failed: %w", err)
	}

	if _, err := j.writer.WriteString("\n"); err != nil {
		j.recordError()
		return fmt.Errorf("newline write failed: %w", err)
	}

	// Флашим если нужно
	if j.autoFlush {
		if err := j.writer.Flush(); err != nil {
			j.recordError()
			return fmt.Errorf("flush failed: %w", err)
		}
	}

	j.recordWrite(startTime, len(jsonData)+1) // +1 для символа новой строки

	return nil
}

// WriteBatch записывает несколько точек данных пачкой
func (j *JSONLWriter) WriteBatch(data []models.DataPoint) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	if len(data) == 0 {
		return nil
	}

	startTime := time.Now()
	totalBytes := 0

	for _, point := range data {
		// Валидация каждой точки
		if err := point.Validate(); err != nil {
			j.recordError()
			return fmt.Errorf("batch contains invalid data: %w", err)
		}

		jsonData, err := json.Marshal(point)
		if err != nil {
			j.recordError()
			return fmt.Errorf("JSON marshaling failed: %w", err)
		}

		if _, err := j.writer.Write(jsonData); err != nil {
			j.recordError()
			return fmt.Errorf("write failed: %w", err)
		}

		if _, err := j.writer.WriteString("\n"); err != nil {
			j.recordError()
			return fmt.Errorf("newline write failed: %w", err)
		}

		totalBytes += len(jsonData) + 1
	}

	if j.autoFlush {
		if err := j.writer.Flush(); err != nil {
			j.recordError()
			return fmt.Errorf("flush failed: %w", err)
		}
	}

	j.recordWrite(startTime, totalBytes)
	j.stats.mu.Lock()
	j.stats.TotalLines += int64(len(data))
	j.stats.mu.Unlock()

	return nil
}

// Flush принудительно сбрасывает буфер в файл
func (j *JSONLWriter) Flush() error {
	j.mu.Lock()
	defer j.mu.Unlock()

	if j.writer == nil {
		return io.ErrClosedPipe
	}

	return j.writer.Flush()
}

// Close закрывает файл и освобождает ресурсы
func (j *JSONLWriter) Close() error {
	j.mu.Lock()
	defer j.mu.Unlock()

	if j.writer != nil {
		if err := j.writer.Flush(); err != nil {
			return fmt.Errorf("final flush failed: %w", err)
		}
	}

	if j.file != nil {
		if err := j.file.Close(); err != nil {
			return fmt.Errorf("file close failed: %w", err)
		}
	}

	j.writer = nil
	j.file = nil

	return nil
}

// GetStats возвращает текущую статистику записи
func (j *JSONLWriter) GetStats() WriteStats {
	j.stats.mu.RLock()
	defer j.stats.mu.RUnlock()

	return *j.stats
}

// ResetStats сбрасывает статистику
func (j *JSONLWriter) ResetStats() {
	j.stats.mu.Lock()
	defer j.stats.mu.Unlock()

	j.stats = &WriteStats{
		LastWriteTime: time.Now(),
	}
}

// recordWrite обновляет статистику после успешной записи
func (j *JSONLWriter) recordWrite(startTime time.Time, bytes int) {
	j.stats.mu.Lock()
	defer j.stats.mu.Unlock()

	writeTime := time.Since(startTime)
	j.stats.TotalLines++
	j.stats.TotalBytes += int64(bytes)
	j.stats.LastWriteTime = time.Now()

	// Обновляем среднее время записи
	if j.stats.TotalLines > 1 {
		totalTime := j.stats.AverageWriteTime * time.Duration(j.stats.TotalLines-1)
		j.stats.AverageWriteTime = (totalTime + writeTime) / time.Duration(j.stats.TotalLines)
	} else {
		j.stats.AverageWriteTime = writeTime
	}
}

// recordError увеличивает счетчик ошибок
func (j *JSONLWriter) recordError() {
	j.stats.mu.Lock()
	defer j.stats.mu.Unlock()

	j.stats.ErrorsCount++
}

// RotateFile закрывает текущий файл и открывает новый
func (j *JSONLWriter) RotateFile(newFilePath string) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	// Сохраняем старую статистику
	oldStats := j.GetStats()

	// Закрываем текущий файл
	if j.writer != nil {
		if err := j.writer.Flush(); err != nil {
			return fmt.Errorf("flush before rotate failed: %w", err)
		}
	}

	if j.file != nil {
		if err := j.file.Close(); err != nil {
			return fmt.Errorf("close before rotate failed: %w", err)
		}
	}

	// Открываем новый файл
	file, err := os.OpenFile(newFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open new file: %w", err)
	}

	writer := bufio.NewWriter(file)

	j.writer = writer
	j.file = file
	j.filePath = newFilePath

	// Восстанавливаем статистику
	j.stats = &oldStats

	return nil
}
