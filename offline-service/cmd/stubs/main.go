package main

import (
	"context"
	"log"
	"net"

	"google.golang.org/grpc"
	"offline-service/internal/pb"
)

// FilterStub - заглушка сервиса фильтрации
type FilterStub struct {
	pb.UnimplementedFilterServiceServer
}

func (s *FilterStub) FilterData(ctx context.Context, req *pb.FilterRequest) (*pb.FilterResponse, error) {
	// Проверяем на nil перед доступом к полям
	if req.MedicalData == nil {
		log.Printf("🔧 FilterStub: Received nil medical data for session %s", req.SessionId)
		// Возвращаем пустые данные вместо паники
		return &pb.FilterResponse{
			FilteredData: &pb.MedicalData{
				Bpm:    &pb.TimeSeries{TimeSec: []float64{}, Value: []float64{}},
				Uterus: &pb.TimeSeries{TimeSec: []float64{}, Value: []float64{}},
			},
			SessionId: req.SessionId,
			Status:    "success",
			Message:   "Received empty data, returning empty response",
		}, nil
	}

	// Безопасный доступ к данным
	bpmPoints := 0
	uterusPoints := 0

	if req.MedicalData.Bpm != nil {
		bpmPoints = len(req.MedicalData.Bpm.TimeSec)
	}
	if req.MedicalData.Uterus != nil {
		uterusPoints = len(req.MedicalData.Uterus.TimeSec)
	}

	log.Printf("🔧 FilterStub: Received data for session %s", req.SessionId)
	log.Printf("   BPM points: %d, Uterus points: %d", bpmPoints, uterusPoints)

	// Создаем безопасную копию данных
	filteredData := &pb.MedicalData{}

	if req.MedicalData.Bpm != nil {
		filteredData.Bpm = &pb.TimeSeries{
			TimeSec: req.MedicalData.Bpm.TimeSec,
			Value:   req.MedicalData.Bpm.Value,
		}
	} else {
		filteredData.Bpm = &pb.TimeSeries{TimeSec: []float64{}, Value: []float64{}}
	}

	if req.MedicalData.Uterus != nil {
		filteredData.Uterus = &pb.TimeSeries{
			TimeSec: req.MedicalData.Uterus.TimeSec,
			Value:   req.MedicalData.Uterus.Value,
		}
	} else {
		filteredData.Uterus = &pb.TimeSeries{TimeSec: []float64{}, Value: []float64{}}
	}

	// Просто возвращаем те же данные без изменений
	return &pb.FilterResponse{
		FilteredData: filteredData,
		SessionId:    req.SessionId,
		Status:       "success",
		Message:      "Data filtered successfully",
	}, nil
}

// MLStub - заглушка ML сервиса
type MLStub struct {
	pb.UnimplementedMLServiceServer
}

func (s *MLStub) Predict(ctx context.Context, req *pb.PredictRequest) (*pb.PredictResponse, error) {
	// Проверяем на nil перед доступом к полям
	if req.MedicalData == nil {
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

	if req.MedicalData.Bpm != nil {
		bpmPoints = len(req.MedicalData.Bpm.TimeSec)
	}
	if req.MedicalData.Uterus != nil {
		uterusPoints = len(req.MedicalData.Uterus.TimeSec)
	}

	log.Printf("🤖 MLStub: Received prediction request for session %s", req.SessionId)
	log.Printf("   BPM points: %d, Uterus points: %d", bpmPoints, uterusPoints)

	// Генерируем "реалистичный" предикт на основе данных
	prediction := s.calculatePrediction(req.MedicalData)

	log.Printf("🤖 MLStub: Prediction for session %s: %.4f", req.SessionId, prediction)

	return &pb.PredictResponse{
		Prediction: prediction,
		SessionId:  req.SessionId,
		Status:     "success",
		Message:    "Prediction calculated successfully",
	}, nil
}

func (s *MLStub) calculatePrediction(medicalData *pb.MedicalData) float64 {
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
