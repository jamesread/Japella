default:
	go build -o japella-adaptor-discord github.com/jamesread/japella/cmd/japella-adaptor-discord/
	go build -o japella-bot-utils-discord github.com/jamesread/japella/cmd/japella-bot-utils/

grpc:
	buf generate
