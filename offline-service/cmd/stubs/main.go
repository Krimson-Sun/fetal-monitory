package main

import (
	"context"
	"log"
	"math"
	"math/rand"
	"net"
	"time"

	"google.golang.org/grpc"
	"offline-service/internal/pb"
)

// FilterStub - заглушка сервиса фильтрации с полным анализом КТГ
type FilterStub struct {
	pb.UnimplementedFilterServiceServer
}

func (s *FilterStub) FilterData(ctx context.Context, req *pb.FilterRequest) (*pb.FilterResponse, error) {
	// Проверяем на nil перед доступом к полям
	if req.MedicalRecord == nil {
		log.Printf("🔧 FilterStub: Received nil medical data for session %s", req.SessionId)
		return s.createEmptyResponse(req.SessionId), nil
	}

	// Безопасный доступ к данным
	bpmPoints := 0
	uterusPoints := 0

	if req.MedicalRecord.Bpm != nil {
		bpmPoints = len(req.MedicalRecord.Bpm.TimeSec)
	}
	if req.MedicalRecord.Uterus != nil {
		uterusPoints = len(req.MedicalRecord.Uterus.TimeSec)
	}

	log.Printf("🔧 FilterStub: Received data for session %s", req.SessionId)
	log.Printf("   BPM points: %d, Uterus points: %d", bpmPoints, uterusPoints)

	// Генерируем полный анализ КТГ
	analysis := s.generateCTGAnalysis(req.MedicalRecord, req.SessionId)

	log.Printf("🔧 FilterStub: Generated CTG analysis for session %s", req.SessionId)
	log.Printf("   STV: %.2f, LTV: %.2f, Baseline: %.1f", analysis.Stv, analysis.Ltv, analysis.BaselineHeartRate)
	log.Printf("   Accelerations: %d, Decelerations: %d, Contractions: %d",
		len(analysis.Accelerations), len(analysis.Decelerations), len(analysis.Contractions))

	return analysis, nil
}

func (s *FilterStub) createEmptyResponse(sessionID string) *pb.FilterResponse {
	return &pb.FilterResponse{
		Stv:                   0,
		Ltv:                   0,
		BaselineHeartRate:     140.0,
		Accelerations:         []*pb.Acceleration{},
		Decelerations:         []*pb.Deceleration{},
		Contractions:          []*pb.Contraction{},
		Stvs:                  []float64{},
		StvsWindowDuration:    3.75,
		Ltvs:                  []float64{},
		LtvsWindowDuration:    60.0,
		TotalDecelerations:    0,
		LateDecelerations:     0,
		LateDecelerationRatio: 0,
		TotalAccelerations:    0,
		AccelDecelRatio:       0,
		TotalContractions:     0,
		StvTrend:              0,
		BpmTrend:              0,
		DataPoints:            0,
		TimeSpanSec:           0,
		FilteredBmpBatch:      &pb.MetricRecord{TimeSec: []float64{}, Value: []float64{}},
		FilteredUterusBatch:   &pb.MetricRecord{TimeSec: []float64{}, Value: []float64{}},
		SessionId:             sessionID,
		Status:                "success",
		Message:               "Received empty data, returning empty analysis",
	}
}

func (s *FilterStub) generateCTGAnalysis(medicalData *pb.MedicalRecord, sessionID string) *pb.FilterResponse {
	bpmValues := []float64{}
	ucValues := []float64{}
	times := []float64{}

	if medicalData.Bpm != nil {
		bpmValues = medicalData.Bpm.Value
		times = medicalData.Bpm.TimeSec
	}
	if medicalData.Uterus != nil {
		ucValues = medicalData.Uterus.Value
	}

	baseline := s.calculateBaseline(bpmValues)
	timeSpan := s.calculateTimeSpan(times)

	return &pb.FilterResponse{
		Stv:                   rand.Float64()*4 + 6,   // 6-10 ms (норма)
		Ltv:                   rand.Float64()*20 + 25, // 25-45 bpm
		BaselineHeartRate:     baseline,
		Accelerations:         s.generateAccelerations(times, baseline),
		Decelerations:         s.generateDecelerations(times, baseline),
		Contractions:          s.generateContractions(times, ucValues),
		Stvs:                  s.generateSTVSeries(times),
		StvsWindowDuration:    3.75,
		Ltvs:                  s.generateLTVSeries(times),
		LtvsWindowDuration:    60.0,
		TotalDecelerations:    int64(rand.Intn(3)),
		LateDecelerations:     int64(rand.Intn(2)),
		LateDecelerationRatio: rand.Float64() * 0.3,
		TotalAccelerations:    int64(rand.Intn(6) + 3),
		AccelDecelRatio:       rand.Float64()*3 + 1, // 1-4
		TotalContractions:     int64(rand.Intn(8) + 2),
		StvTrend:              rand.Float64()*0.4 - 0.2, // -0.2 to +0.2
		BpmTrend:              rand.Float64()*8 - 4,     // -4 to +4 bpm/min
		DataPoints:            int64(len(times)),
		TimeSpanSec:           timeSpan,
		FilteredBmpBatch:      s.applyFiltering(medicalData).Bpm,
		FilteredUterusBatch:   s.applyFiltering(medicalData).Uterus,
		SessionId:             sessionID,
		Status:                "success",
		Message:               "CTG analysis completed successfully",
	}
}

func (s *FilterStub) calculateBaseline(values []float64) float64 {
	if len(values) == 0 {
		return 140.0
	}

	// Используем медиану для более устойчивого расчета
	sorted := make([]float64, len(values))
	copy(sorted, values)

	for i := range sorted {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	return sorted[len(sorted)/2]
}

func (s *FilterStub) calculateTimeSpan(times []float64) float64 {
	if len(times) < 2 {
		return 0
	}
	return times[len(times)-1] - times[0]
}

func (s *FilterStub) generateAccelerations(times []float64, baseline float64) []*pb.Acceleration {
	if len(times) < 20 {
		return []*pb.Acceleration{}
	}

	var accelerations []*pb.Acceleration
	numAccels := 2 + rand.Intn(3) // 2-4 акселерации

	for i := 0; i < numAccels; i++ {
		startIdx := 10 + rand.Intn(len(times)-20)
		duration := 20.0 + rand.Float64()*40.0 // 20-60 секунд

		accel := &pb.Acceleration{
			Start:     times[startIdx],
			End:       times[startIdx] + duration,
			Amplitude: baseline + 10 + rand.Float64()*15, // +10-25 bpm
			Duration:  duration,
		}
		accelerations = append(accelerations, accel)
	}

	return accelerations
}

func (s *FilterStub) generateDecelerations(times []float64, baseline float64) []*pb.Deceleration {
	if len(times) < 30 {
		return []*pb.Deceleration{}
	}

	var decelerations []*pb.Deceleration
	numDecels := rand.Intn(2) // 0-1 децелерации (норма)

	for i := 0; i < numDecels; i++ {
		startIdx := 15 + rand.Intn(len(times)-30)
		duration := 30.0 + rand.Float64()*60.0 // 30-90 секунд

		decel := &pb.Deceleration{
			Start:     times[startIdx],
			End:       times[startIdx] + duration,
			Amplitude: baseline - 10 - rand.Float64()*15, // -10-25 bpm
			Duration:  duration,
			IsLate:    false,
		}
		decelerations = append(decelerations, decel)
	}

	return decelerations
}

func (s *FilterStub) generateContractions(times, ucValues []float64) []*pb.Contraction {
	if len(times) < 60 {
		return []*pb.Contraction{}
	}

	var contractions []*pb.Contraction

	// Генерируем схватки каждые 2-3 минуты
	currentTime := times[0] + 60.0 // начинаем через минуту
	for currentTime < times[len(times)-1]-120 {
		duration := 50.0 + rand.Float64()*40.0 // 50-90 секунд
		gap := 120.0 + rand.Float64()*60.0     // 2-3 минуты между схватками

		contraction := &pb.Contraction{
			Start:     currentTime,
			End:       currentTime + duration,
			Amplitude: 40.0 + rand.Float64()*35.0, // 40-75 единиц
			Duration:  duration,
		}
		contractions = append(contractions, contraction)

		currentTime += duration + gap
	}

	return contractions
}

func (s *FilterStub) generateSTVSeries(times []float64) []float64 {
	if len(times) == 0 {
		return []float64{}
	}

	// STV рассчитывается для 3.75-секундных окон
	numWindows := int(math.Ceil(float64(len(times)) / 15.0))
	series := make([]float64, numWindows)

	for i := range series {
		// Реалистичные значения STV с небольшими вариациями
		baseSTV := 8.0 + rand.Float64()*2.0 // 8-10 ms
		noise := rand.Float64()*1.0 - 0.5   // -0.5 to +0.5
		series[i] = math.Max(2.0, baseSTV+noise)
	}

	return series
}

func (s *FilterStub) generateLTVSeries(times []float64) []float64 {
	if len(times) == 0 {
		return []float64{}
	}

	// LTV рассчитывается для минутных окон
	numWindows := int(math.Ceil(float64(len(times)) / 240.0))
	series := make([]float64, numWindows)

	for i := range series {
		// Реалистичные значения LTV
		baseLTV := 30.0 + rand.Float64()*15.0 // 30-45 bpm
		noise := rand.Float64()*3.0 - 1.5     // -1.5 to +1.5
		series[i] = math.Max(10.0, baseLTV+noise)
	}

	return series
}

func (s *FilterStub) applyFiltering(medicalData *pb.MedicalRecord) *pb.MedicalRecord {
	bpmTimes := []float64{}
	bpmValues := []float64{}
	ucTimes := []float64{}
	ucValues := []float64{}

	if medicalData.Bpm != nil {
		bpmTimes = medicalData.Bpm.TimeSec
		bpmValues = medicalData.Bpm.Value
	}
	if medicalData.Uterus != nil {
		ucTimes = medicalData.Uterus.TimeSec
		ucValues = medicalData.Uterus.Value
	}

	// Применяем простой фильтр для BPM данных
	filteredBPM := s.lowPassFilter(bpmValues, 0.3)

	return &pb.MedicalRecord{
		Bpm: &pb.MetricRecord{
			TimeSec: bpmTimes,
			Value:   filteredBPM,
		},
		Uterus: &pb.MetricRecord{
			TimeSec: ucTimes,
			Value:   ucValues, // UC данные обычно не фильтруем
		},
	}
}

func (s *FilterStub) lowPassFilter(values []float64, alpha float64) []float64 {
	if len(values) == 0 {
		return values
	}

	filtered := make([]float64, len(values))
	filtered[0] = values[0]

	for i := 1; i < len(values); i++ {
		filtered[i] = alpha*values[i] + (1-alpha)*filtered[i-1]
	}

	return filtered
}

// MLStub - заглушка ML сервиса (остается без изменений)
type MLStub struct {
	pb.UnimplementedMLServiceServer
}

func (s *MLStub) Predict(ctx context.Context, req *pb.PredictRequest) (*pb.PredictResponse, error) {
	// Проверяем на nil перед доступом к полям
	if req.MedicalRecord == nil {
		log.Printf("🤖 MLStub: Received nil medical data for session %s", req.SessionId)
		return &pb.PredictResponse{
			Prediction: 0.5,
			SessionId:  req.SessionId,
			Status:     "success",
			Message:    "Received empty data, using default prediction",
		}, nil
	}

	// Безопасный доступ к данным
	bpmPoints := 0
	uterusPoints := 0

	if req.MedicalRecord.Bpm != nil {
		bpmPoints = len(req.MedicalRecord.Bpm.TimeSec)
	}
	if req.MedicalRecord.Uterus != nil {
		uterusPoints = len(req.MedicalRecord.Uterus.TimeSec)
	}

	log.Printf("🤖 MLStub: Received prediction request for session %s", req.SessionId)
	log.Printf("   BPM points: %d, Uterus points: %d", bpmPoints, uterusPoints)

	// Генерируем "реалистичный" предикт на основе данных
	prediction := s.calculatePrediction(req.MedicalRecord)

	log.Printf("🤖 MLStub: Prediction for session %s: %.4f", req.SessionId, prediction)

	return &pb.PredictResponse{
		Prediction: prediction,
		SessionId:  req.SessionId,
		Status:     "success",
		Message:    "Prediction calculated successfully",
	}, nil
}

func (s *MLStub) calculatePrediction(medicalData *pb.MedicalRecord) float64 {
	// Безопасный расчет с проверкой на nil
	if medicalData == nil || medicalData.Bpm == nil || len(medicalData.Bpm.Value) == 0 {
		return 0.5 // нейтральный предикт по умолчанию
	}

	var totalBPM, totalUterus float64
	var bpmCount, uterusCount int

	// Считаем BPM
	if medicalData.Bpm != nil && len(medicalData.Bpm.Value) > 0 {
		for _, val := range medicalData.Bpm.Value {
			totalBPM += float64(val)
		}
		bpmCount = len(medicalData.Bpm.Value)
	}

	// Считаем Uterus
	if medicalData.Uterus != nil && len(medicalData.Uterus.Value) > 0 {
		for _, val := range medicalData.Uterus.Value {
			totalUterus += float64(val)
		}
		uterusCount = len(medicalData.Uterus.Value)
	}

	// Если нет данных, возвращаем значение по умолчанию
	if bpmCount == 0 {
		return 0.5
	}

	avgBPM := totalBPM / float64(bpmCount)
	avgUterus := 0.0
	if uterusCount > 0 {
		avgUterus = totalUterus / float64(uterusCount)
	}

	// Простая "модель" на основе средних значений
	prediction := 0.3 + (avgBPM-140)*0.001 + (avgUterus-15)*0.01

	// Ограничиваем между 0 и 1
	if prediction < 0 {
		prediction = 0
	}
	if prediction > 1 {
		prediction = 1
	}

	return prediction
}

func startFilterStub() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("❌ Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterFilterServiceServer(s, &FilterStub{})

	log.Printf("🔧 Filter stub server listening at %v", lis.Addr())

	if err := s.Serve(lis); err != nil {
		log.Fatalf("❌ Failed to serve filter stub: %v", err)
	}
}

func startMLStub() {
	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("❌ Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterMLServiceServer(s, &MLStub{})

	log.Printf("🤖 ML stub server listening at %v", lis.Addr())

	if err := s.Serve(lis); err != nil {
		log.Fatalf("❌ Failed to serve ML stub: %v", err)
	}
}

func main() {
	log.Println("🚀 Starting gRPC stubs...")

	// Инициализируем random seed для реалистичных данных
	rand.Seed(time.Now().UnixNano())

	// Запускаем оба сервиса в отдельных горутинах
	go startFilterStub()
	go startMLStub()

	log.Println("✅ All gRPC stubs are running!")
	log.Println("   - Filter service: localhost:50051")
	log.Println("   - ML service: localhost:50052")
	log.Println("   Press Ctrl+C to stop")

	// Бесконечное ожидание
	select {}
}
