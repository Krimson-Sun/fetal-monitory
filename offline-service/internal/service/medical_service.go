package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"offline-service/internal/pb"
	"offline-service/pkg/models"
)

type MedicalService struct {
	filterClient pb.FilterServiceClient
	mlClient     pb.MLServiceClient
	cacheRepo    CacheRepository
	dbRepo       DBRepository
}

type CacheRepository interface {
	SaveSession(ctx context.Context, sessionID string, data *models.MedicalSession) error
	GetSession(ctx context.Context, sessionID string) (*models.MedicalSession, error)
	DeleteSession(ctx context.Context, sessionID string) error
}

type DBRepository interface {
	SaveMedicalData(ctx context.Context, data *models.MedicalSession) error
}

func NewMedicalService(
	filterClient pb.FilterServiceClient,
	mlClient pb.MLServiceClient,
	cacheRepo CacheRepository,
	dbRepo DBRepository,
) *MedicalService {
	return &MedicalService{
		filterClient: filterClient,
		mlClient:     mlClient,
		cacheRepo:    cacheRepo,
		dbRepo:       dbRepo,
	}
}

//func (s *MedicalService) ProcessCSV(ctx context.Context, file io.Reader, sessionID string) (*models.UploadResponse, error) {
//	fmt.Printf("Starting CSV processing for session: %s\n", sessionID)
//
//	// Парсинг CSV
//	medicalRecord, err := s.parseCSV(file)
//	if err != nil {
//		fmt.Printf("CSV parsing failed for session %s: %v\n", sessionID, err)
//		return nil, fmt.Errorf("failed to parse CSV: %w", err)
//	}
//
//	fmt.Printf("Parsed %d FHR points and %d UC points for session %s\n",
//		len(medicalRecord.FetalHeartRate.Time), len(medicalRecord.UterineContractions.Time), sessionID)
//
//	// Если gRPC клиенты не доступны, возвращаем данные без обработки
//	if s.filterClient == nil || s.mlClient == nil {
//		fmt.Printf("⚠️ gRPC services not available, returning raw data for session %s\n", sessionID)
//
//		medicalSession := &models.MedicalSession{
//			SessionID:  sessionID,
//			Records:    *medicalRecord,
//			Prediction: 0.5, // Значение по умолчанию
//			CreatedAt:  time.Now(),
//			Status:     "pending",
//		}
//
//		// Сохраняем в Redis
//		if err := s.cacheRepo.SaveSession(ctx, sessionID, medicalSession); err != nil {
//			fmt.Printf("Failed to save session %s to Redis: %v\n", sessionID, err)
//			return nil, fmt.Errorf("failed to save session: %w", err)
//		}
//
//		return &models.UploadResponse{
//			SessionID:  sessionID,
//			Status:     "processed",
//			Records:    *medicalRecord,
//			Prediction: medicalSession.Prediction,
//			Message:    "Data processed without gRPC services",
//		}, nil
//	}
//
//	// Конвертация в gRPC формат
//	grpcData := s.convertToGRPCData(medicalRecord)
//
//	// Отправка в сервис фильтрации
//	fmt.Printf("Sending data to filter service for session %s\n", sessionID)
//	filterResp, err := s.filterClient.FilterData(ctx, &pb.FilterRequest{
//		MedicalData: grpcData,
//		SessionId:   sessionID,
//	})
//	if err != nil {
//		fmt.Printf("Filter service error for session %s: %v\n", sessionID, err)
//		return nil, fmt.Errorf("filter service error: %w", err)
//	}
//
//	// Отправка в ML сервис
//	fmt.Printf("Sending filtered data to ML service for session %s\n", sessionID)
//	predictResp, err := s.mlClient.Predict(ctx, &pb.PredictRequest{
//		MedicalData: filterResp.FilteredData,
//		SessionId:   sessionID,
//	})
//	if err != nil {
//		fmt.Printf("ML service error for session %s: %v\n", sessionID, err)
//		return nil, fmt.Errorf("ML service error: %w", err)
//	}
//
//	fmt.Printf("Received prediction for session %s: %f\n", sessionID, predictResp.Prediction)
//
//	// Конвертируем обратно из gRPC формата
//	filteredRecord := s.convertFromGRPCData(filterResp.FilteredData)
//
//	// Сохранение в Redis
//	medicalSession := &models.MedicalSession{
//		SessionID:  sessionID,
//		Records:    filteredRecord,
//		Prediction: predictResp.Prediction,
//		CreatedAt:  time.Now(),
//		Status:     "pending",
//	}
//
//	if err := s.cacheRepo.SaveSession(ctx, sessionID, medicalSession); err != nil {
//		fmt.Printf("Failed to save session %s to Redis: %v\n", sessionID, err)
//		return nil, fmt.Errorf("failed to save session: %w", err)
//	}
//
//	fmt.Printf("Successfully processed and saved session %s\n", sessionID)
//
//	// Возвращаем полный ответ с данными
//	return &models.UploadResponse{
//		SessionID:  sessionID,
//		Status:     "processed",
//		Records:    filteredRecord,
//		Prediction: medicalSession.Prediction,
//		Message:    "Data successfully processed and ready for decision",
//	}, nil
//}

func (s *MedicalService) HandleDecision(ctx context.Context, decision *models.SaveDecision) (*models.DecisionResponse, error) {
	fmt.Printf("Processing decision for session %s: save=%t\n", decision.SessionID, decision.Save)

	if !decision.Save {
		// Получаем данные перед удалением для ответа
		session, err := s.cacheRepo.GetSession(ctx, decision.SessionID)
		if err != nil {
			fmt.Printf("Failed to get session %s from cache: %v\n", decision.SessionID, err)
			return nil, fmt.Errorf("failed to get session data: %w", err)
		}

		// Удаляем сессию из кеша
		if err := s.cacheRepo.DeleteSession(ctx, decision.SessionID); err != nil {
			fmt.Printf("Failed to delete session %s: %v\n", decision.SessionID, err)
			return nil, fmt.Errorf("failed to delete session: %w", err)
		}

		fmt.Printf("Session %s deleted from cache\n", decision.SessionID)

		return &models.DecisionResponse{
			Status:  "cancelled",
			Message: "Data was not saved and has been deleted",
			Data: map[string]interface{}{
				"session_id":        decision.SessionID,
				"fhr_records_count": len(session.Records.FetalHeartRate.Time),
				"uc_records_count":  len(session.Records.UterineContractions.Time), // Исправлено
				"prediction":        session.Prediction,
			},
		}, nil
	}

	// Получаем данные из кеша
	session, err := s.cacheRepo.GetSession(ctx, decision.SessionID)
	if err != nil {
		fmt.Printf("Failed to get session %s from cache: %v\n", decision.SessionID, err)
		return nil, fmt.Errorf("failed to get session data: %w", err)
	}

	// Сохраняем в PostgreSQL
	if err := s.dbRepo.SaveMedicalData(ctx, session); err != nil {
		fmt.Printf("Failed to save session %s to database: %v\n", decision.SessionID, err)
		return nil, fmt.Errorf("failed to save to database: %w", err)
	}

	// Обновляем статус в Redis
	session.Status = "saved"
	if err := s.cacheRepo.SaveSession(ctx, decision.SessionID, session); err != nil {
		fmt.Printf("Warning: failed to update session status in Redis for %s: %v\n", decision.SessionID, err)
	}

	fmt.Printf("Session %s successfully saved to database with %d FHR points and %d UC points\n",
		decision.SessionID, len(session.Records.FetalHeartRate.Time), len(session.Records.UterineContractions.Time))

	return &models.DecisionResponse{
		Status:  "saved",
		Message: "Data successfully saved to database",
		Data: map[string]interface{}{
			"session_id":        session.SessionID,
			"fhr_records_count": len(session.Records.FetalHeartRate.Time),
			"uc_records_count":  len(session.Records.UterineContractions.Time), // Исправлено
			"prediction":        session.Prediction,
			"created_at":        session.CreatedAt,
			"saved_at":          time.Now(),
		},
	}, nil
}

// Парсинг CSV
//func (s *MedicalService) parseCSV(file io.Reader) (*models.MedicalRecord, error) {
//	reader := csv.NewReader(file)
//	reader.TrimLeadingSpace = true
//	reader.Comma = ','          // Явно указываем разделитель
//	reader.FieldsPerRecord = -1 // Разрешаем разное количество полей
//
//	records, err := reader.ReadAll()
//	if err != nil {
//		return nil, fmt.Errorf("failed to read CSV: %w", err)
//	}
//
//	if len(records) == 0 {
//		return nil, fmt.Errorf("empty CSV file")
//	}
//
//	fmt.Printf("Read %d lines from CSV\n", len(records))
//
//	medicalRecord := &models.MedicalRecord{
//		FetalHeartRate: models.MetricRecord{
//			Time:  make([]float64, 0),
//			Value: make([]float64, 0),
//		},
//		UterineContractions: models.MetricRecord{
//			Time:  make([]float64, 0),
//			Value: make([]float64, 0),
//		},
//	}
//
//	// Пропускаем заголовок если есть
//	startIndex := 0
//	if isHeader(records[0]) {
//		fmt.Printf("Found header: %v\n", records[0])
//		startIndex = 1
//	}
//
//	validRecords := 0
//	for i := startIndex; i < len(records); i++ {
//		if len(records[i]) < 4 {
//			fmt.Printf("Warning: skipping line %d - not enough columns: %v\n", i+1, records[i])
//			continue
//		}
//
//		// Парсим FHR время (секунды)
//		fhrTime, err := strconv.ParseFloat(strings.TrimSpace(records[i][0]), 64)
//		if err != nil {
//			fmt.Printf("Warning: skipping line %d - invalid FHR time '%s': %v\n", i+1, records[i][0], err)
//			continue
//		}
//
//		// Парсим FHR значение
//		fhrValue, err := strconv.ParseFloat(strings.TrimSpace(records[i][1]), 64)
//		if err != nil {
//			fmt.Printf("Warning: skipping line %d - invalid FHR value '%s': %v\n", i+1, records[i][1], err)
//			continue
//		}
//
//		// Парсим UC время (секунды)
//		ucTime, err := strconv.ParseFloat(strings.TrimSpace(records[i][2]), 64)
//		if err != nil {
//			fmt.Printf("Warning: skipping line %d - invalid UC time '%s': %v\n", i+1, records[i][2], err)
//			continue
//		}
//
//		// Парсим UC значение
//		ucValue, err := strconv.ParseFloat(strings.TrimSpace(records[i][3]), 64)
//		if err != nil {
//			fmt.Printf("Warning: skipping line %d - invalid UC value '%s': %v\n", i+1, records[i][3], err)
//			continue
//		}
//
//		// Добавляем данные
//		medicalRecord.FetalHeartRate.Time = append(medicalRecord.FetalHeartRate.Time, fhrTime)
//		medicalRecord.FetalHeartRate.Value = append(medicalRecord.FetalHeartRate.Value, fhrValue)
//		medicalRecord.UterineContractions.Time = append(medicalRecord.UterineContractions.Time, ucTime)
//		medicalRecord.UterineContractions.Value = append(medicalRecord.UterineContractions.Value, ucValue)
//
//		validRecords++
//		fmt.Printf("Parsed record %d: FHR(time=%.1f, value=%.1f), UC(time=%.1f, value=%.1f)\n",
//			validRecords, fhrTime, fhrValue, ucTime, ucValue)
//	}
//
//	if validRecords == 0 {
//		return nil, fmt.Errorf("no valid medical records found in CSV - checked %d lines", len(records)-startIndex)
//	}
//
//	fmt.Printf("Successfully parsed %d medical records\n", validRecords)
//	return medicalRecord, nil
//}

func (s *MedicalService) ProcessDualCSV(ctx context.Context, bpmFile, ucFile io.Reader, sessionID string) (*models.UploadResponse, error) {
	fmt.Printf("Starting dual CSV processing for session: %s\n", sessionID)

	// Парсим BPM данные
	bpmData, err := s.parseSingleCSV(bpmFile, "BPM")
	if err != nil {
		return nil, fmt.Errorf("failed to parse BPM CSV: %w", err)
	}

	// Парсим UC данные
	ucData, err := s.parseSingleCSV(ucFile, "UC")
	if err != nil {
		return nil, fmt.Errorf("failed to parse UC CSV: %w", err)
	}

	fmt.Printf("Parsed %d BPM points and %d UC points for session %s\n",
		len(bpmData.Time), len(ucData.Time), sessionID)

	// Создаем объединенные медицинские данные
	medicalData := models.MedicalRecord{
		FetalHeartRate: models.MetricRecord{
			Time:  bpmData.Time,
			Value: bpmData.Value,
		},
		UterineContractions: models.MetricRecord{
			Time:  ucData.Time,
			Value: ucData.Value,
		},
	}

	var prediction float64
	var message string
	var processedData models.MedicalRecord

	// Если gRPC клиенты доступны, используем их для обработки
	if s.filterClient != nil && s.mlClient != nil {
		fmt.Printf("Using gRPC services for session %s\n", sessionID)

		// Конвертация в gRPC формат
		grpcData := s.convertMedicalDataToGRPC(medicalData)

		// Отправка в сервис фильтрации
		filterResp, err := s.filterClient.FilterData(ctx, &pb.FilterRequest{
			MedicalData: grpcData,
			SessionId:   sessionID,
		})
		if err != nil {
			fmt.Printf("Filter service error, using unfiltered data: %v\n", err)
			processedData = medicalData
			message = "Used unfiltered data due to filter service error"
		} else {
			// Конвертируем отфильтрованные данные обратно
			processedData = s.convertFromGRPCData(filterResp.FilteredData)
			message = "Data successfully filtered"
		}

		// Отправка в ML сервис
		predictResp, err := s.mlClient.Predict(ctx, &pb.PredictRequest{
			MedicalData: grpcData, // Отправляем оригинальные данные для prediction
			SessionId:   sessionID,
		})
		if err != nil {
			fmt.Printf("ML service error, using default prediction: %v\n", err)
			prediction = 0.5
			message += "; Used fallback prediction due to ML service error"
		} else {
			prediction = predictResp.Prediction
			message += "; Prediction calculated successfully"
		}

	} else {
		// gRPC сервисы недоступны - используем fallback
		fmt.Printf("gRPC services not available for session %s, using fallback processing\n", sessionID)
		processedData = medicalData
		prediction = 0.5
		message = "Data processed without gRPC services"
	}

	// Сохранение в Redis
	medicalSession := &models.MedicalSession{
		SessionID:  sessionID,
		Records:    processedData,
		Prediction: prediction,
		CreatedAt:  time.Now(),
		Status:     "pending",
	}

	if err := s.cacheRepo.SaveSession(ctx, sessionID, medicalSession); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	fmt.Printf("Successfully processed dual CSV session %s\n", sessionID)
	fmt.Printf("Final stats - BPM: %d points, UC: %d points, Prediction: %.4f\n",
		len(processedData.FetalHeartRate.Time), len(processedData.UterineContractions.Time), prediction)

	return &models.UploadResponse{
		SessionID:  sessionID,
		Status:     "processed",
		Records:    processedData,
		Prediction: prediction,
		Message:    message,
	}, nil
}

// Парсер для одиночного CSV файла (2 колонки: time_sec, value)
func (s *MedicalService) parseSingleCSV(file io.Reader, dataType string) (*models.MetricRecord, error) {
	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read %s CSV: %w", dataType, err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("empty %s CSV file", dataType)
	}

	fmt.Printf("Read %d lines from %s CSV\n", len(records), dataType)

	result := &models.MetricRecord{
		Time:  make([]float64, 0),
		Value: make([]float64, 0),
	}

	startIndex := 0
	if isHeader(records[0]) {
		fmt.Printf("Found %s header, skipping: %v\n", dataType, records[0])
		startIndex = 1
	}

	validRecords := 0
	for i := startIndex; i < len(records); i++ {
		if len(records[i]) < 2 {
			fmt.Printf("Warning: %s line %d has only %d columns, need 2\n", dataType, i+1, len(records[i]))
			continue
		}

		// Парсим время
		timeVal, err := strconv.ParseFloat(strings.TrimSpace(records[i][0]), 64)
		if err != nil {
			fmt.Printf("Warning: %s line %d - time error: %v\n", dataType, i+1, err)
			continue
		}

		// Парсим значение
		value, err := strconv.ParseFloat(strings.TrimSpace(records[i][1]), 64)
		if err != nil {
			fmt.Printf("Warning: %s line %d - value error: %v\n", dataType, i+1, err)
			continue
		}

		result.Time = append(result.Time, timeVal)
		result.Value = append(result.Value, value)
		validRecords++
	}

	if validRecords == 0 {
		return nil, fmt.Errorf("no valid %s records found in CSV", dataType)
	}

	fmt.Printf("Successfully parsed %d %s records\n", validRecords, dataType)
	return result, nil
}

// Конвертер из MedicalData в gRPC формат
func (s *MedicalService) convertMedicalDataToGRPC(medicalData models.MedicalRecord) *pb.MedicalData {
	return &pb.MedicalData{
		Bpm: &pb.TimeSeries{
			TimeSec: medicalData.FetalHeartRate.Time,
			Value:   medicalData.FetalHeartRate.Value,
		},
		Uterus: &pb.TimeSeries{
			TimeSec: medicalData.UterineContractions.Time,
			Value:   medicalData.UterineContractions.Value,
		},
	}
}

// Конвертер в gRPC формат
func (s *MedicalService) convertToGRPCData(medicalRecord *models.MedicalRecord) *pb.MedicalData {
	return &pb.MedicalData{
		Bpm: &pb.TimeSeries{
			TimeSec: medicalRecord.FetalHeartRate.Time,
			Value:   medicalRecord.FetalHeartRate.Value,
		},
		Uterus: &pb.TimeSeries{
			TimeSec: medicalRecord.UterineContractions.Time,
			Value:   medicalRecord.UterineContractions.Value,
		},
	}
}

// Конвертер из gRPC формата
func (s *MedicalService) convertFromGRPCData(grpcData *pb.MedicalData) models.MedicalRecord {
	return models.MedicalRecord{
		FetalHeartRate: models.MetricRecord{
			Time:  grpcData.Bpm.TimeSec,
			Value: grpcData.Bpm.Value,
		},
		UterineContractions: models.MetricRecord{
			Time:  grpcData.Uterus.TimeSec,
			Value: grpcData.Uterus.Value,
		},
	}
}

// Вспомогательные функции
func isHeader(row []string) bool {
	if len(row) == 0 {
		return false
	}

	firstCell := strings.ToLower(strings.TrimSpace(row[0]))
	headerIndicators := []string{"time", "fhr", "uc", "uterine", "contractions", "value"}

	for _, indicator := range headerIndicators {
		if strings.Contains(firstCell, indicator) {
			return true
		}
	}

	return false
}
