package dashboard

import (
	"context"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	pb "github.com/jamesread/japella/gen/protobuf"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net/http"
)

func StartRestGateway() {
	mux := newMux()

	err := http.ListenAndServe(":8081", mux)

	if err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}

func newMux() *runtime.ServeMux {
	mux := runtime.NewServeMux()

	ctx := context.Background()

	opts := []grpc.DialOption{grpc.WithInsecure()}

	err := pb.RegisterJapellaDashboardApiServiceHandlerFromEndpoint(ctx, mux, "localhost:50051", opts)

	if err != nil {
		log.Fatalf("Failed to register gateway: %v", err)
	}

	return mux
}
