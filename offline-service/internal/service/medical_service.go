package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
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

func (s *MedicalService) ProcessDualCSV(ctx context.Context, bpmFile, ucFile io.Reader, sessionID string) (*models.UploadResponse, error) {
	var prediction float64
	var message string

	// Парсим CSV файлы
	bpmData, err := s.parseSingleCSV(bpmFile, "BPM")
	if err != nil {
		return nil, fmt.Errorf("failed to parse BPM CSV: %w", err)
	}

	ucData, err := s.parseSingleCSV(ucFile, "UC")
	if err != nil {
		return nil, fmt.Errorf("failed to parse UC CSV: %w", err)
	}

	log.Printf("Parsed %d BPM points and %d UC points for session %s",
		len(bpmData.TimeSec), len(ucData.TimeSec), sessionID)

	// Создаем медицинскую запись
	medicalRecord := models.MedicalRecord{
		FetalHeartRate:      *bpmData,
		UterineContractions: *ucData,
	}

	// Конвертируем в protobuf для gRPC
	pbMedicalRecord := s.convertMedicalRecordToGRPC(medicalRecord)
	// Вызываем фильтр-сервис для анализа КТГ
	filterResp, err := s.filterClient.FilterData(ctx, &pb.FilterRequest{
		MedicalRecord: pbMedicalRecord,
		SessionId:     sessionID,
	})
	if err != nil {
		return nil, fmt.Errorf("filter service failed: %w", err)
	}

	// Конвертируем ответ фильтра в нашу модель анализа
	processedData := s.convertFilterResponseToAnalysis(filterResp)
	message = "Data successfully filtered"

	// Получаем прогноз от ML сервиса
	if s.mlClient != nil {
		mlResp, err := s.mlClient.Predict(ctx, &pb.PredictRequest{
			MedicalRecord: &pb.MedicalRecord{
				Bpm:    filterResp.FilteredBmpBatch,
				Uterus: filterResp.FilteredUterusBatch,
			},
			SessionId: sessionID,
		})
		if err != nil {
			return nil, fmt.Errorf("ML services not available for session %s", sessionID)
		} else {
			prediction = mlResp.Prediction
			message += "; Prediction calculated successfully"
		}
	} else {
		return nil, fmt.Errorf("ML services not available for session %s", sessionID)
	}

	// Сохранение в Redis
	medicalSession := &models.MedicalSession{
		SessionID: sessionID,
		Records: models.MedicalRecord{
			FetalHeartRate:      processedData.FilteredBMPBatch,
			UterineContractions: processedData.FilteredUterusBatch},
		Prediction: prediction,
		CreatedAt:  time.Now(),
		Status:     "completed",
	}

	if err := s.cacheRepo.SaveSession(ctx, sessionID, medicalSession); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return &models.UploadResponse{
		SessionID:  sessionID,
		Status:     "success",
		Records:    *processedData,
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
		TimeSec: make([]float64, 0),
		Value:   make([]float64, 0),
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

		result.TimeSec = append(result.TimeSec, timeVal)
		result.Value = append(result.Value, value)
		validRecords++
	}

	if validRecords == 0 {
		return nil, fmt.Errorf("no valid %s records found in CSV", dataType)
	}

	fmt.Printf("Successfully parsed %d %s records\n", validRecords, dataType)
	return result, nil
}

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
				"fhr_records_count": len(session.Records.FetalHeartRate.TimeSec),
				"uc_records_count":  len(session.Records.UterineContractions.TimeSec), // Исправлено
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

	if err := s.dbRepo.SaveSession(ctx, session); err != nil {
		log.Printf("Warning: failed to save to PostgreSQL: %v", err)
		// Но продолжаем работу - данные есть в Redis
	}

	// Обновляем статус в Redis
	session.Status = "saved"
	if err := s.cacheRepo.SaveSession(ctx, decision.SessionID, session); err != nil {
		fmt.Printf("Warning: failed to update session status in Redis for %s: %v\n", decision.SessionID, err)
	}

	fmt.Printf("Session %s successfully saved to database with %d FHR points and %d UC points\n",
		decision.SessionID, len(session.Records.FetalHeartRate.TimeSec), len(session.Records.UterineContractions.TimeSec))

	return &models.DecisionResponse{
		Status:  "saved",
		Message: "Data successfully saved to database",
		Data: map[string]interface{}{
			"session_id":        session.SessionID,
			"fhr_records_count": len(session.Records.FetalHeartRate.TimeSec),
			"uc_records_count":  len(session.Records.UterineContractions.TimeSec), // Исправлено
			"prediction":        session.Prediction,
			"created_at":        session.CreatedAt,
			"saved_at":          time.Now(),
		},
	}, nil
}

// Конвертер из MedicalRecord в gRPC формат
func (s *MedicalService) convertMedicalRecordToGRPC(MedicalRecord models.MedicalRecord) *pb.MedicalRecord {
	return &pb.MedicalRecord{
		Bpm: &pb.MetricRecord{
			TimeSec: MedicalRecord.FetalHeartRate.TimeSec,
			Value:   MedicalRecord.FetalHeartRate.Value,
		},
		Uterus: &pb.MetricRecord{
			TimeSec: MedicalRecord.UterineContractions.TimeSec,
			Value:   MedicalRecord.UterineContractions.Value,
		},
	}
}

// Конвертер из gRPC формата
func (s *MedicalService) convertFromGRPCToMetric(grpcData *pb.MetricRecord) models.MetricRecord {
	return models.MetricRecord{
		TimeSec: grpcData.TimeSec,
		Value:   grpcData.Value,
	}
}

func (s *MedicalService) convertFilterResponseToAnalysis(resp *pb.FilterResponse) *models.CTGAnalysis {
	return &models.CTGAnalysis{
		STV:                   resp.Stv,
		LTV:                   resp.Ltv,
		BaselineHeartRate:     resp.BaselineHeartRate,
		Accelerations:         s.convertPBAccelerations(resp.Accelerations),
		Decelerations:         s.convertPBDecelerations(resp.Decelerations),
		Contractions:          s.convertPBContractions(resp.Contractions),
		STVs:                  resp.Stvs,
		STVsWindowDuration:    resp.StvsWindowDuration,
		LTVs:                  resp.Ltvs,
		LTVsWindowDuration:    resp.LtvsWindowDuration,
		TotalDecelerations:    resp.TotalDecelerations,
		LateDecelerations:     resp.LateDecelerations,
		LateDecelerationRatio: resp.LateDecelerationRatio,
		TotalAccelerations:    resp.TotalAccelerations,
		AccelDecelRatio:       resp.AccelDecelRatio,
		TotalContractions:     resp.TotalContractions,
		STVTrend:              resp.StvTrend,
		BPMTrend:              resp.BpmTrend,
		DataPoints:            resp.DataPoints,
		TimeSpanSec:           resp.TimeSpanSec,
		FilteredBMPBatch:      s.convertFromGRPCToMetric(resp.FilteredBmpBatch),
		FilteredUterusBatch:   s.convertFromGRPCToMetric(resp.FilteredUterusBatch),
	}
}

func (s *MedicalService) convertPBAccelerations(pbAccels []*pb.Acceleration) []models.Acceleration {
	accelerations := make([]models.Acceleration, len(pbAccels))
	for i, accel := range pbAccels {
		accelerations[i] = models.Acceleration{
			StartTime: accel.Start,
			EndTime:   accel.End,
			Amplitude: accel.Amplitude,
			Duration:  accel.Duration,
		}
	}
	return accelerations
}

func (s *MedicalService) convertPBDecelerations(pbDecels []*pb.Deceleration) []models.Deceleration {
	decelerations := make([]models.Deceleration, len(pbDecels))
	for i, decel := range pbDecels {
		decelerations[i] = models.Deceleration{
			StartTime: decel.Start,
			EndTime:   decel.End,
			Amplitude: decel.Amplitude,
			Duration:  decel.Duration,
			IsLate:    decel.IsLate,
		}
	}
	return decelerations
}

func (s *MedicalService) convertPBContractions(pbContractions []*pb.Contraction) []models.Contraction {
	contractions := make([]models.Contraction, len(pbContractions))
	for i, cont := range pbContractions {
		contractions[i] = models.Contraction{
			StartTime: cont.Start,
			EndTime:   cont.End,
			Amplitude: cont.Amplitude,
			Duration:  cont.Duration,
		}
	}
	return contractions
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
