default:
	go build -o japella-adaptor-discord github.com/jamesread/japella/cmd/japella-adaptor-discord/
	go build -o japella-bot-utils-discord github.com/jamesread/japella/cmd/japella-bot-utils/

go-tools:
	go install "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway"
	go install "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2"
	go install "google.golang.org/grpc/cmd/protoc-gen-go-grpc"
	go install "google.golang.org/protobuf/cmd/protoc-gen-go"

grpc:
	buf generate
