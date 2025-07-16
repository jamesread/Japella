default:
	go build -ldflags="-s -w -X buildinfo.Commit=$(shell git rev-parse HEAD)"

test: tests

tests:
	go vet ./...
	mkdir -p reports
	go test ./... -v -coverprofile=reports/coverage.out -covermode=atomic
	go tool cover -html=reports/coverage.out -o reports/coverage.html
	cp reports/coverage.html /var/www/html/japella-coverage/index.html

dev:
	air

codestyle:
	go fmt ./...
	go vet ./...
	gocyclo -over 5 internal
	gocritic check

go-tools:
	go install "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway"
	go install "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2"
	go install "google.golang.org/grpc/cmd/protoc-gen-go-grpc"
	go install "google.golang.org/protobuf/cmd/protoc-gen-go"
	#go install "github.com/go-kod/kod/cmd/kod"
	go install "go.uber.org/mock/mockgen"

certs: cert-ca cert-server

cert-ca:
	openssl genrsa -out ca.key 4096
	openssl req -x509 -new -nodes -key ca.key -sha256 -out ca.crt -subj "/CN=Japella CA"

cert-server:
	openssl genrsa -out server.key 4096
	openssl req -new -key server.key -out server.csr -subj "/CN=Japella Server"
	openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 3650 -sha256
	cat ca.crt >> server.crt
