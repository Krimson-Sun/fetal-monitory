package stubs

import (
	"context"
	"log"
	"math/rand"
	"net"

	"google.golang.org/grpc"
	"offline-service/internal/pb"
)

type MLStub struct {
	pb.UnimplementedMLServiceServer
}

func (s *MLStub) Predict(ctx context.Context, req *pb.PredictRequest) (*pb.PredictResponse, error) {
	log.Printf("Received prediction request for %d records (Session: %s)", len(req.MedicalData.Bpm.TimeSec)+len(req.MedicalData.Uterus.TimeSec), req.SessionId)

	// Генерируем случайный предикт между 0 и 1
	prediction := rand.Float64()

	return &pb.PredictResponse{
		Prediction: prediction,
		SessionId:  req.SessionId,
	}, nil
}

func startMLStub() {
	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterMLServiceServer(s, &MLStub{})

	log.Printf("ML stub server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
