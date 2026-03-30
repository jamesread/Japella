package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1"
	log "github.com/sirupsen/logrus"
)

func (s *ControlApi) ListUserGroups(ctx context.Context, req *connect.Request[controlv1.ListUserGroupsRequest]) (*connect.Response[controlv1.ListUserGroupsResponse], error) {
	groups, err := s.DB.SelectUserGroups()
	if err != nil {
		log.Errorf("ListUserGroups: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list user groups"))
	}
	out := make([]*controlv1.UserGroup, 0, len(groups))
	for _, g := range groups {
		out = append(out, &controlv1.UserGroup{
			Id:          g.ID,
			Name:        g.Name,
			MemberCount: g.MemberCount,
		})
	}
	return connect.NewResponse(&controlv1.ListUserGroupsResponse{Groups: out}), nil
}

func (s *ControlApi) CreateUserGroup(ctx context.Context, req *connect.Request[controlv1.CreateUserGroupRequest]) (*connect.Response[controlv1.CreateUserGroupResponse], error) {
	name := strings.TrimSpace(req.Msg.Name)
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name is required"))
	}

	id, err := s.DB.CreateUserGroup(name)
	if err != nil {
		log.Errorf("CreateUserGroup: %v", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("could not create group (duplicate name or invalid data)"))
	}
	return connect.NewResponse(&controlv1.CreateUserGroupResponse{
		StandardResponse: &controlv1.StandardResponse{Success: true, Message: "Group created"},
		GroupId:          id,
	}), nil
}

func (s *ControlApi) DeleteUserGroup(ctx context.Context, req *connect.Request[controlv1.DeleteUserGroupRequest]) (*connect.Response[controlv1.DeleteUserGroupResponse], error) {
	if req.Msg.GroupId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("group_id is required"))
	}
	if err := s.DB.DeleteUserGroup(req.Msg.GroupId); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("group not found"))
		}
		log.Errorf("DeleteUserGroup: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delete group"))
	}
	return connect.NewResponse(&controlv1.DeleteUserGroupResponse{
		StandardResponse: &controlv1.StandardResponse{Success: true, Message: "Group deleted"},
	}), nil
}

func (s *ControlApi) GetUserGroupMembers(ctx context.Context, req *connect.Request[controlv1.GetUserGroupMembersRequest]) (*connect.Response[controlv1.GetUserGroupMembersResponse], error) {
	if req.Msg.GroupId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("group_id is required"))
	}
	if s.DB.GetUserGroupByID(req.Msg.GroupId) == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("group not found"))
	}
	ids, err := s.DB.SelectUserGroupMemberIDs(req.Msg.GroupId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load group members"))
	}
	return connect.NewResponse(&controlv1.GetUserGroupMembersResponse{UserIds: ids}), nil
}

func (s *ControlApi) SetUserGroupMembers(ctx context.Context, req *connect.Request[controlv1.SetUserGroupMembersRequest]) (*connect.Response[controlv1.SetUserGroupMembersResponse], error) {
	if req.Msg.GroupId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("group_id is required"))
	}
	if s.DB.GetUserGroupByID(req.Msg.GroupId) == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("group not found"))
	}
	if err := s.DB.SetUserGroupMembers(req.Msg.GroupId, req.Msg.UserIds); err != nil {
		log.Errorf("SetUserGroupMembers: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to set group members"))
	}
	return connect.NewResponse(&controlv1.SetUserGroupMembersResponse{
		StandardResponse: &controlv1.StandardResponse{Success: true, Message: "Group members updated"},
	}), nil
}
