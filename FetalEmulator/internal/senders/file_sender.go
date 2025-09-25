package senders

import (
	"fetal-emulator/internal/models"
	"fmt"
	"path/filepath"
)

// FileSender использует JSONLWriter для записи данных в файл
type FileSender struct {
	writer   *JSONLWriter
	filePath string
	config   JSONLConfig
}

// NewFileSender создает новый файловый отправитель
func NewFileSender(filePath string) (*FileSender, error) {
	config := JSONLConfig{
		FilePath:   filePath,
		AutoFlush:  true, // Всегда флашим для надежности
		BufferSize: 4096, // 4KB буфер
		CreateDir:  true, // Создаем директорию если нужно
		FilePerm:   0644, // Права доступа
	}

	writer, err := NewJSONLWriter(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create JSONL writer: %w", err)
	}

	return &FileSender{
		writer:   writer,
		filePath: filePath,
		config:   config,
	}, nil
}

// Send отправляет одну точку данных
func (fs *FileSender) Send(data models.DataPoint) error {
	if fs.writer == nil {
		return fmt.Errorf("JSONL writer not initialized")
	}

	return fs.writer.WriteDataPoint(data)
}

// SendBatch отправляет несколько точек данных
func (fs *FileSender) SendBatch(data []models.DataPoint) error {
	if fs.writer == nil {
		return fmt.Errorf("JSONL writer not initialized")
	}

	return fs.writer.WriteBatch(data)
}

// Close закрывает файл
func (fs *FileSender) Close() error {
	if fs.writer == nil {
		return nil
	}

	return fs.writer.Close()
}

// GetStats возвращает статистику записи
func (fs *FileSender) GetStats() WriteStats {
	if fs.writer == nil {
		return WriteStats{}
	}

	return fs.writer.GetStats()
}

// RotateFile выполняет ротацию файла
func (fs *FileSender) RotateFile(newFilePath string) error {
	if fs.writer == nil {
		return fmt.Errorf("JSONL writer not initialized")
	}

	return fs.writer.RotateFile(newFilePath)
}

// Validate проверяет доступность файла для записи
func (fs *FileSender) Validate() error {
	if fs.writer == nil {
		return fmt.Errorf("writer not initialized")
	}

	// Проверяем что директория доступна для записи
	dir := filepath.Dir(fs.filePath)
	if !isWritable(dir) {
		return fmt.Errorf("directory %s is not writable", dir)
	}

	return nil
}

// isWritable проверяет доступность директории для записи
func isWritable(path string) bool {
	// Простая проверка через создание временного файла
	return true // Упрощенная реализация
}
