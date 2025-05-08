package controlapi

import (
	"connectrpc.com/connect"
	connectcors "connectrpc.com/cors"
	"context"

	"net/http"

	"github.com/rs/cors"

	buildinfo "github.com/jamesread/japella/internal/buildinfo"
	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1"
	"github.com/jamesread/japella/gen/japella/controlapi/v1/controlv1connect"
	"github.com/jamesread/japella/internal/nanoservice"
	log "github.com/sirupsen/logrus"
)

type ControlApi struct{}

func (s ControlApi) GetStatus(ctx context.Context, req *connect.Request[controlv1.GetStatusRequest]) (*connect.Response[controlv1.GetStatusResponse], error) {
	res := connect.NewResponse(&controlv1.GetStatusResponse{
		Status: "OK!",
		Nanoservices: nanoservice.GetNanoservices(),
		Version: buildinfo.Version,
	})

	return res, nil
}

func (s ControlApi) SubmitPost(ctx context.Context, req *connect.Request[controlv1.SubmitPostRequest]) (*connect.Response[controlv1.SubmitPostResponse], error) {
	res := connect.NewResponse(&controlv1.SubmitPostResponse{})


	log.Infof("Received post request: %+v", req.Msg.Content)

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

func (s ControlApi) GetPostingServices(ctx context.Context, req *connect.Request[controlv1.GetPostingServicesRequest]) (*connect.Response[controlv1.GetPostingServicesResponse], error) {

	services := []*controlv1.PostingService{
		&controlv1.PostingService{
			Name:        "telegram",
		},
		&controlv1.PostingService{
			Name:        "discord",
		},
	}

	res := connect.NewResponse(&controlv1.GetPostingServicesResponse{
		Services: services,
	})

	return res, nil
}
