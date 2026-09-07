package api

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1"
	"github.com/jamesread/japella/internal/connector/dummy"
)

func (s *ControlApi) CreateDummySocialAccount(ctx context.Context, req *connect.Request[controlv1.CreateDummySocialAccountRequest]) (*connect.Response[controlv1.CreateDummySocialAccountResponse], error) {
	au := s.getAuthenticatedUser(ctx)
	if au == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	if s.cc == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("connectors not ready"))
	}
	svc := s.cc.Get(dummy.Protocol)
	if svc == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("dummy connector is not running"))
	}
	d, ok := svc.(*dummy.Connector)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("dummy connector type mismatch"))
	}

	id, err := d.CreateAccount(req.Msg.Identity, au.User.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create dummy account: %w", err))
	}
	sa, err := s.DB.GetSocialAccount(id)
	identity := req.Msg.Identity
	if err == nil && sa != nil {
		identity = sa.Identity
	}
	return connect.NewResponse(&controlv1.CreateDummySocialAccountResponse{
		Id:       id,
		Identity: identity,
	}), nil
}
