package iam

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jamesread/armature-iam/rbac"
	"github.com/jamesread/armature-iam/store"
	"github.com/jamesread/japella/internal/db"
)

func (s *DBStore) ListUserGroups(ctx context.Context) ([]store.UserGroupRow, error) {
	groups, err := s.DB.SelectUserGroups()
	if err != nil {
		return nil, err
	}
	out := make([]store.UserGroupRow, 0, len(groups))
	for _, g := range groups {
		out = append(out, store.UserGroupRow{
			ID:          int(g.ID),
			Name:        g.Name,
			MemberCount: int(g.MemberCount),
			CreatedAt:   formatTime(g.CreatedAt),
			UpdatedAt:   formatTime(g.UpdatedAt),
		})
	}
	return out, nil
}

func (s *DBStore) GetUserGroupByName(ctx context.Context, name string) (*store.UserGroupRow, error) {
	g := s.DB.GetUserGroupByName(name)
	if g == nil {
		return nil, nil
	}
	return userGroupRowFromModel(g, 0), nil
}

func (s *DBStore) GetUserGroupByID(ctx context.Context, id int) (*store.UserGroupRow, error) {
	g := s.DB.GetUserGroupByID(uint32(id))
	if g == nil {
		return nil, nil
	}
	members, _ := s.ListUserGroupMemberIDs(ctx, id)
	return userGroupRowFromModel(g, len(members)), nil
}

func userGroupRowFromModel(g *db.UserGroup, memberCount int) *store.UserGroupRow {
	if g == nil {
		return nil
	}
	return &store.UserGroupRow{
		ID:          int(g.ID),
		Name:        g.Name,
		MemberCount: memberCount,
		CreatedAt:   formatTime(g.CreatedAt),
		UpdatedAt:   formatTime(g.UpdatedAt),
	}
}

func (s *DBStore) CreateUserGroup(ctx context.Context, name string) (int, error) {
	id, err := s.DB.CreateUserGroup(name)
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func (s *DBStore) DeleteUserGroup(ctx context.Context, id int) error {
	g := s.DB.GetUserGroupByID(uint32(id))
	if g == nil {
		return sql.ErrNoRows
	}
	if rbac.IsSystemGroup(g.Name) {
		return fmt.Errorf("%w %q", store.ErrSystemGroup, g.Name)
	}
	return s.DB.DeleteUserGroup(uint32(id))
}

func (s *DBStore) ListUserGroupMemberIDs(ctx context.Context, groupID int) ([]int, error) {
	ids, err := s.DB.SelectUserGroupMemberIDs(uint32(groupID))
	if err != nil {
		return nil, err
	}
	return uint32SliceToInt(ids), nil
}

func (s *DBStore) ListUserGroupIDsForUser(ctx context.Context, userID int) ([]int, error) {
	ids, err := s.DB.SelectUserGroupIDsForUser(uint32(userID))
	if err != nil {
		return nil, err
	}
	return uint32SliceToInt(ids), nil
}

func (s *DBStore) SetUserGroupMembers(ctx context.Context, groupID int, userIDs []int) error {
	if err := s.DB.SetUserGroupMembers(uint32(groupID), intSliceToUint32(userIDs)); err != nil {
		return err
	}
	return s.ensureSuperuserCoverage(ctx)
}
