// internal/grpcclient/grpcclient.go
package grpcclient

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	telemetryv1 "new-fetal-emulator/proto/v1"
)

type GRPCClient struct {
	client    telemetryv1.DataServiceClient
	conn      *grpc.ClientConn
	sessionID string
}

func NewGRPCClient(serverAddr, sessionID string) (*GRPCClient, error) {
	conn, err := grpc.Dial(serverAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithTimeout(5*time.Second))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %w", err)
	}

	client := telemetryv1.NewDataServiceClient(conn)

	return &GRPCClient{
		client:    client,
		conn:      conn,
		sessionID: sessionID,
	}, nil
}

func (g *GRPCClient) PushSamples(ctx context.Context, samples <-chan telemetryv1.Sample) error {
	stream, err := g.client.PushSamples(ctx)
	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}

	go g.receiveAcks(stream)

	for sample := range samples {
		if err := stream.Send(&sample); err != nil {
			return fmt.Errorf("failed to send sample: %w", err)
		}
	}

	return stream.CloseSend()
}

func (g *GRPCClient) receiveAcks(stream telemetryv1.DataService_PushSamplesClient) {
	for {
		ack, err := stream.Recv()
		if err != nil {
			log.Printf("Failed to receive ack: %v", err)
			return
		}
		log.Printf("Received ack for session %s: received_cnt=%d",
			ack.SessionId, ack.ReceivedCnt)
	}
}

func (g *GRPCClient) Close() error {
	return g.conn.Close()
}
