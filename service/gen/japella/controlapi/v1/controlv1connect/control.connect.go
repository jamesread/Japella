// Code generated by protoc-gen-connect-go. DO NOT EDIT.
//
// Source: japella/controlapi/v1/control.proto

package controlv1connect

import (
	connect "connectrpc.com/connect"
	context "context"
	errors "errors"
	v1 "github.com/jamesread/japella/gen/japella/controlapi/v1"
	http "net/http"
	strings "strings"
)

// This is a compile-time assertion to ensure that this generated file and the connect package are
// compatible. If you get a compiler error that this constant is not defined, this code was
// generated with a version of connect newer than the one compiled into your binary. You can fix the
// problem by either regenerating this code with an older version of connect or updating the connect
// version compiled into your binary.
const _ = connect.IsAtLeastVersion1_13_0

const (
	// JapellaControlApiServiceName is the fully-qualified name of the JapellaControlApiService service.
	JapellaControlApiServiceName = "japella.controlapi.v1.JapellaControlApiService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// JapellaControlApiServiceGetStatusProcedure is the fully-qualified name of the
	// JapellaControlApiService's GetStatus RPC.
	JapellaControlApiServiceGetStatusProcedure = "/japella.controlapi.v1.JapellaControlApiService/GetStatus"
	// JapellaControlApiServiceSendMessageProcedure is the fully-qualified name of the
	// JapellaControlApiService's SendMessage RPC.
	JapellaControlApiServiceSendMessageProcedure = "/japella.controlapi.v1.JapellaControlApiService/SendMessage"
)

// These variables are the protoreflect.Descriptor objects for the RPCs defined in this package.
var (
	japellaControlApiServiceServiceDescriptor           = v1.File_japella_controlapi_v1_control_proto.Services().ByName("JapellaControlApiService")
	japellaControlApiServiceGetStatusMethodDescriptor   = japellaControlApiServiceServiceDescriptor.Methods().ByName("GetStatus")
	japellaControlApiServiceSendMessageMethodDescriptor = japellaControlApiServiceServiceDescriptor.Methods().ByName("SendMessage")
)

// JapellaControlApiServiceClient is a client for the japella.controlapi.v1.JapellaControlApiService
// service.
type JapellaControlApiServiceClient interface {
	GetStatus(context.Context, *connect.Request[v1.GetStatusRequest]) (*connect.Response[v1.GetStatusResponse], error)
	SendMessage(context.Context, *connect.Request[v1.SendMessageRequest]) (*connect.Response[v1.SendMessageResponse], error)
}

// NewJapellaControlApiServiceClient constructs a client for the
// japella.controlapi.v1.JapellaControlApiService service. By default, it uses the Connect protocol
// with the binary Protobuf Codec, asks for gzipped responses, and sends uncompressed requests. To
// use the gRPC or gRPC-Web protocols, supply the connect.WithGRPC() or connect.WithGRPCWeb()
// options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewJapellaControlApiServiceClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) JapellaControlApiServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &japellaControlApiServiceClient{
		getStatus: connect.NewClient[v1.GetStatusRequest, v1.GetStatusResponse](
			httpClient,
			baseURL+JapellaControlApiServiceGetStatusProcedure,
			connect.WithSchema(japellaControlApiServiceGetStatusMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
		sendMessage: connect.NewClient[v1.SendMessageRequest, v1.SendMessageResponse](
			httpClient,
			baseURL+JapellaControlApiServiceSendMessageProcedure,
			connect.WithSchema(japellaControlApiServiceSendMessageMethodDescriptor),
			connect.WithClientOptions(opts...),
		),
	}
}

// japellaControlApiServiceClient implements JapellaControlApiServiceClient.
type japellaControlApiServiceClient struct {
	getStatus   *connect.Client[v1.GetStatusRequest, v1.GetStatusResponse]
	sendMessage *connect.Client[v1.SendMessageRequest, v1.SendMessageResponse]
}

// GetStatus calls japella.controlapi.v1.JapellaControlApiService.GetStatus.
func (c *japellaControlApiServiceClient) GetStatus(ctx context.Context, req *connect.Request[v1.GetStatusRequest]) (*connect.Response[v1.GetStatusResponse], error) {
	return c.getStatus.CallUnary(ctx, req)
}

// SendMessage calls japella.controlapi.v1.JapellaControlApiService.SendMessage.
func (c *japellaControlApiServiceClient) SendMessage(ctx context.Context, req *connect.Request[v1.SendMessageRequest]) (*connect.Response[v1.SendMessageResponse], error) {
	return c.sendMessage.CallUnary(ctx, req)
}

// JapellaControlApiServiceHandler is an implementation of the
// japella.controlapi.v1.JapellaControlApiService service.
type JapellaControlApiServiceHandler interface {
	GetStatus(context.Context, *connect.Request[v1.GetStatusRequest]) (*connect.Response[v1.GetStatusResponse], error)
	SendMessage(context.Context, *connect.Request[v1.SendMessageRequest]) (*connect.Response[v1.SendMessageResponse], error)
}

// NewJapellaControlApiServiceHandler builds an HTTP handler from the service implementation. It
// returns the path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewJapellaControlApiServiceHandler(svc JapellaControlApiServiceHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	japellaControlApiServiceGetStatusHandler := connect.NewUnaryHandler(
		JapellaControlApiServiceGetStatusProcedure,
		svc.GetStatus,
		connect.WithSchema(japellaControlApiServiceGetStatusMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	japellaControlApiServiceSendMessageHandler := connect.NewUnaryHandler(
		JapellaControlApiServiceSendMessageProcedure,
		svc.SendMessage,
		connect.WithSchema(japellaControlApiServiceSendMessageMethodDescriptor),
		connect.WithHandlerOptions(opts...),
	)
	return "/japella.controlapi.v1.JapellaControlApiService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case JapellaControlApiServiceGetStatusProcedure:
			japellaControlApiServiceGetStatusHandler.ServeHTTP(w, r)
		case JapellaControlApiServiceSendMessageProcedure:
			japellaControlApiServiceSendMessageHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedJapellaControlApiServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedJapellaControlApiServiceHandler struct{}

func (UnimplementedJapellaControlApiServiceHandler) GetStatus(context.Context, *connect.Request[v1.GetStatusRequest]) (*connect.Response[v1.GetStatusResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("japella.controlapi.v1.JapellaControlApiService.GetStatus is not implemented"))
}

func (UnimplementedJapellaControlApiServiceHandler) SendMessage(context.Context, *connect.Request[v1.SendMessageRequest]) (*connect.Response[v1.SendMessageResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("japella.controlapi.v1.JapellaControlApiService.SendMessage is not implemented"))
}
