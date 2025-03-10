package controlapi

import (
	"connectrpc.com/connect"
	"context"

	"net/http"

	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1"
	"github.com/jamesread/japella/gen/japella/controlapi/v1/controlv1connect"
)

type ControlApi struct{}

func (s ControlApi) GetStatus(ctx context.Context, req *connect.Request[controlv1.GetStatusRequest]) (*connect.Response[controlv1.GetStatusResponse], error) {
	res := connect.NewResponse(&controlv1.GetStatusResponse{
		Status: "OK!",
	})

	return res, nil
}

func (s ControlApi) SendMessage(ctx context.Context, req *connect.Request[controlv1.SendMessageRequest]) (*connect.Response[controlv1.SendMessageResponse], error) {
	res := connect.NewResponse(&controlv1.SendMessageResponse{})

	return res, nil
}

func GetNewHandler() (string, http.Handler) {
	server := ControlApi{}

	path, handler := controlv1connect.NewJapellaControlApiServiceHandler(server)

	return path, handler
}
