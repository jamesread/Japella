default:
	air

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
	#go install "github.com/go-kod/kod/cmd/kod"
	go install "go.uber.org/mock/mockgen"

webui-dist:
	cd webui.dev && npm install
	cd webui.dev && npx parcel build --public-url "."
	mv webui.dev/dist webui
