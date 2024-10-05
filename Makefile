default:
	go build -o japella-adaptor-discord github.com/jamesread/japella/cmd/japella-adaptor-discord/
	go build -o japella-adaptor-telegram github.com/jamesread/japella/cmd/japella-adaptor-telegram/
	go build -o japella-adaptor-mastodon github.com/jamesread/japella/cmd/japella-adaptor-mastodon/
	go build -o japella-bot-utils github.com/jamesread/japella/cmd/japella-bot-utils/
	go build -o japella-bot-watcher-prometheus github.com/jamesread/japella/cmd/japella-bot-watcher-prometheus/
	go build -o japella-bot-dblogger github.com/jamesread/japella/cmd/japella-bot-dblogger/
	go build -o japella-dashboard github.com/jamesread/japella/cmd/japella-dashboard/

localcontainers:
	buildah bud -f Dockerfile.japella-adaptor-discord 			-t registry.k8s.teratan.lan/japella-adaptor-discord
	buildah bud -f Dockerfile.japella-adaptor-telegram  		-t registry.k8s.teratan.lan/japella-adaptor-telegram
	buildah bud -f Dockerfile.japella-bot-utils         		-t registry.k8s.teratan.lan/japella-bot-utils
	buildah bud -f Dockerfile.japella-bot-watcher-prometheus  	-t registry.k8s.teratan.lan/japella-bot-watcher-prometheus
	buildah bud -f Dockerfile.japella-bot-dblogger 				-t registry.k8s.teratan.lan/japella-bot-dblogger

grpc: go-tools
	buf generate

go-tools:
	go install "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway"
	go install "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2"
	go install "google.golang.org/grpc/cmd/protoc-gen-go-grpc"
	go install "google.golang.org/protobuf/cmd/protoc-gen-go"
