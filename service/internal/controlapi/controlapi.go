package controlapi

import (
	"connectrpc.com/connect"
	connectcors "connectrpc.com/cors"
	"context"

	"net/http"

	"github.com/rs/cors"

	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1"
	"github.com/jamesread/japella/gen/japella/controlapi/v1/controlv1connect"

	"os"
	"strings"
)

type ControlApi struct{}

func (s ControlApi) GetStatus(ctx context.Context, req *connect.Request[controlv1.GetStatusRequest]) (*connect.Response[controlv1.GetStatusResponse], error) {
	res := connect.NewResponse(&controlv1.GetStatusResponse{
		Status: "OK!",
		Nanoservices: strings.Split(os.Getenv("JAPELLA_NANOSERVICES"), ","),
	})

	return res, nil
}

func (s ControlApi) SendMessage(ctx context.Context, req *connect.Request[controlv1.SendMessageRequest]) (*connect.Response[controlv1.SendMessageResponse], error) {
	res := connect.NewResponse(&controlv1.SendMessageResponse{})

	return res, nil
}

func withCors(h http.Handler) http.Handler {
	middleware := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: connectcors.AllowedMethods(),
		AllowedHeaders: connectcors.AllowedHeaders(),
		ExposedHeaders: connectcors.ExposedHeaders(),
	})

	return middleware.Handler(h)
}

func GetNewHandler() (string, http.Handler) {
	server := ControlApi{}

	path, handler := controlv1connect.NewJapellaControlApiServiceHandler(server)

	return path, withCors(handler)
}
