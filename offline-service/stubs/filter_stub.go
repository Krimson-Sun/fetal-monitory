package stubs

import (
	"context"
	"log"
	"net"

	"google.golang.org/grpc"
	"offline-service/internal/pb"
)

type FilterStub struct {
	pb.UnimplementedFilterServiceServer
}

func (s *FilterStub) FilterData(ctx context.Context, req *pb.FilterRequest) (*pb.FilterResponse, error) {
	log.Printf("Received %d records for filtering (Session: %s)", len(req.MedicalData.Bpm.TimeSec)+len(req.MedicalData.Uterus.TimeSec), req.SessionId)

	// Просто возвращаем те же данные без изменений
	return &pb.FilterResponse{
		FilteredData: req.MedicalData,
		SessionId:    req.SessionId,
	}, nil
}

func startFilterStub() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterFilterServiceServer(s, &FilterStub{})

	log.Printf("Filter stub server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
