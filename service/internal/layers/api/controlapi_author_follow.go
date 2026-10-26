package api

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1"
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/layers/authentication"
)

// authorFollowTarget validates a follow request and loads its account and connector.
func (s *ControlApi) authorFollowTarget(au *authentication.AuthenticatedUser, socialAccountID uint32, authorID string) (*db.SocialAccount, connector.BaseConnector, error) {
	if au == nil || au.User == nil {
		return nil, nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	if socialAccountID == 0 || strings.TrimSpace(authorID) == "" {
		return nil, nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("social_account_id and author_id are required"))
	}
	if !s.userCanReadSocialAccount(au, socialAccountID) {
		return nil, nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("read access to this social account is required"))
	}
	sa, err := s.DB.GetSocialAccount(socialAccountID)
	if err != nil || sa == nil {
		return nil, nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("social account not found"))
	}
	if s.cc == nil {
		return sa, nil, nil
	}
	return sa, s.cc.Get(sa.Connector), nil
}

func followError(sa *db.SocialAccount, err error) error {
	if rl := connector.AsRateLimited(err); rl != nil {
		return connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("rate limited by %s; try again in %s", sa.Connector, rl.RetryAfter))
	}
	return connect.NewError(connect.CodeUnavailable, err)
}

func (s *ControlApi) GetAuthorFollow(ctx context.Context, req *connect.Request[controlv1.GetAuthorFollowRequest]) (*connect.Response[controlv1.GetAuthorFollowResponse], error) {
	au := s.getAuthenticatedUser(ctx)
	sa, svc, err := s.authorFollowTarget(au, req.Msg.GetSocialAccountId(), req.Msg.GetAuthorId())
	if err != nil {
		return nil, err
	}
	fc, ok := svc.(connector.ConnectorWithAuthorFollow)
	if !ok {
		return connect.NewResponse(&controlv1.GetAuthorFollowResponse{}), nil
	}

	state, err := fc.GetFollowState(toConnectorSA(s.freshSocialAccount(sa, svc)), req.Msg.GetAuthorId())
	if err != nil {
		return nil, followError(sa, err)
	}
	return connect.NewResponse(&controlv1.GetAuthorFollowResponse{
		Supported: true,
		Following: state.Following,
		Requested: state.Requested,
		CanChange: s.userCanPostWithSocialAccount(au, sa.ID),
	}), nil
}

func (s *ControlApi) SetAuthorFollow(ctx context.Context, req *connect.Request[controlv1.SetAuthorFollowRequest]) (*connect.Response[controlv1.SetAuthorFollowResponse], error) {
	au := s.getAuthenticatedUser(ctx)
	sa, svc, err := s.authorFollowTarget(au, req.Msg.GetSocialAccountId(), req.Msg.GetAuthorId())
	if err != nil {
		return nil, err
	}
	if !s.userCanPostWithSocialAccount(au, sa.ID) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("post access to this social account is required to follow as it"))
	}
	fc, ok := svc.(connector.ConnectorWithAuthorFollow)
	if !ok {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("following isn't available for %s", sa.Connector))
	}

	state, err := fc.SetFollowing(toConnectorSA(s.freshSocialAccount(sa, svc)), req.Msg.GetAuthorId(), req.Msg.GetFollow())
	if err != nil {
		return nil, followError(sa, err)
	}
	return connect.NewResponse(&controlv1.SetAuthorFollowResponse{Following: state.Following, Requested: state.Requested}), nil
}
