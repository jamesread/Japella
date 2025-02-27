package dashboard

import (
	pb "github.com/jamesread/japella/gen/protobuf"
	"google.golang.org/grpc"
	"net"
	log "github.com/sirupsen/logrus"

	"context"
)

type JapellaDashboardApi struct {

}

func StartGrpc() {
	lis, err := net.Listen("tcp", ":50051")

	grpcServer := grpc.NewServer()
	pb.RegisterJapellaDashboardApiServiceServer(grpcServer, &JapellaDashboardApi{})

	err = grpcServer.Serve(lis)

	if err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}

func (s *JapellaDashboardApi) SendMessage(ctx context.Context, in *pb.SendMessageRequest) (*pb.SendMessageResponse, error) {
	return &pb.SendMessageResponse{}, nil
}
