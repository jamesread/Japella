package api

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1"
	"github.com/jamesread/japella/internal/connector/dummy"
)

func (s *ControlApi) dummyConnector() (*dummy.Connector, error) {
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
	return d, nil
}

func (s *ControlApi) dummyAccountIdentity(id uint32, fallback string) string {
	sa, err := s.DB.GetSocialAccount(id)
	if err == nil && sa != nil {
		return sa.Identity
	}
	return fallback
}

func (s *ControlApi) CreateDummySocialAccount(ctx context.Context, req *connect.Request[controlv1.CreateDummySocialAccountRequest]) (*connect.Response[controlv1.CreateDummySocialAccountResponse], error) {
	au, err := s.requireAuthenticatedUser(ctx)
	if err != nil {
		return nil, err
	}
	d, err := s.dummyConnector()
	if err != nil {
		return nil, err
	}
	id, err := d.CreateAccount(req.Msg.Identity, au.User.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create dummy account: %w", err))
	}
	return connect.NewResponse(&controlv1.CreateDummySocialAccountResponse{
		Id: id, Identity: s.dummyAccountIdentity(id, req.Msg.Identity),
	}), nil
}
