package iam

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jamesread/armature-iam/rbac"
	"github.com/jamesread/armature-iam/store"
	"github.com/jamesread/japella/internal/db"
)

// DBStore implements armature-iam store.Store on Japella's MySQL-backed db.DB.
type DBStore struct {
	DB *db.DB
}

var _ store.Store = (*DBStore)(nil)

func NewDBStore(dbc *db.DB) *DBStore {
	return &DBStore{DB: dbc}
}

func (s *DBStore) CountUserAccounts(ctx context.Context) (int, error) {
	var n int
	err := s.DB.ResilientGet(&n, `SELECT COUNT(*) FROM user_accounts`)
	if err != nil {
		return 0, fmt.Errorf("count user accounts: %w", err)
	}
	return n, nil
}

func (s *DBStore) GetUserByUsername(ctx context.Context, username string) (*store.UserAccountRow, error) {
	u := s.DB.GetUserByUsername(username)
	if u == nil {
		return nil, nil
	}
	return userRowFromAccount(u), nil
}

func (s *DBStore) GetUserByID(ctx context.Context, id int) (*store.UserAccountRow, error) {
	u := s.DB.GetUserByID(uint32(id))
	if u == nil {
		return nil, nil
	}
	return userRowFromAccount(u), nil
}

func (s *DBStore) ListUserAccounts(ctx context.Context) ([]store.UserAccountRow, error) {
	users, err := s.DB.SelectUsers()
	if err != nil {
		return nil, err
	}
	out := make([]store.UserAccountRow, 0, len(users))
	for _, u := range users {
		if row := userRowFromAccount(u); row != nil {
			out = append(out, *row)
		}
	}
	return out, nil
}

func (s *DBStore) CreateUserAccount(ctx context.Context, username, passwordHash, createdBy string) (int, error) {
	if createdBy == "" {
		createdBy = store.UserCreatedByAdmin
	}
	u, err := s.DB.CreateUserAccount(username, passwordHash, createdBy)
	if err != nil {
		return 0, err
	}
	return int(u.ID), nil
}

func (s *DBStore) DeleteUserAccount(ctx context.Context, id int) error {
	return s.DB.DeleteUserAccount(uint32(id))
}

func (s *DBStore) UpdateUserPassword(ctx context.Context, id int, passwordHash string) error {
	return s.DB.UpdateUserPassword(uint32(id), passwordHash)
}

func (s *DBStore) CreateSession(ctx context.Context, sid string, userID int, impersonatorID *int) error {
	if impersonatorID != nil {
		return s.DB.ImpersonateSession(sid, uint32(userID), uint32(*impersonatorID))
	}
	return s.DB.CreateSession(sid, uint32(userID))
}

func (s *DBStore) GetSessionBySID(ctx context.Context, sid string) (*store.SessionRow, error) {
	sess := s.DB.GetSessionByID(sid)
	if sess == nil {
		return nil, nil
	}
	row := &store.SessionRow{
		ID:            int(sess.ID),
		SID:           sess.SID,
		UserAccountID: int(sess.UserAccountID),
		CreatedAt:     formatTime(sess.CreatedAt),
		UpdatedAt:     formatTime(sess.UpdatedAt),
	}
	if sess.ImpersonatorUserID.Valid {
		v := int(sess.ImpersonatorUserID.Int32)
		row.ImpersonatorUserID = &v
	}
	return row, nil
}

func (s *DBStore) DeleteSession(ctx context.Context, sid string) error {
	return s.DB.DeleteSession(sid)
}

func (s *DBStore) DeleteSessionsForUser(ctx context.Context, userID int) error {
	_, err := s.DB.ResilientExec(`DELETE FROM sessions WHERE user_account_id = ?`, userID)
	return err
}

func (s *DBStore) ListAPIKeysForUser(ctx context.Context, userID int) ([]store.APIKeyRow, error) {
	rows := make([]db.ApiKey, 0)
	err := s.DB.ResilientSelect(&rows, `
		SELECT id, created_at, updated_at, name, read_only, key_value, user_account_id, last_used_at
		FROM api_keys WHERE user_account_id = ? ORDER BY name, id`, userID)
	if err != nil {
		return nil, fmt.Errorf("list api keys for user: %w", err)
	}
	out := make([]store.APIKeyRow, 0, len(rows))
	for _, k := range rows {
		out = append(out, store.APIKeyRow{
			ID:            int(k.ID),
			UserAccountID: int(k.UserAccountID),
			Name:          k.Name,
			KeyValue:      k.KeyValue,
			ReadOnly:      k.ReadOnly,
			LastUsedAt:    formatNullTime(k.LastUsedAt),
			CreatedAt:     formatTime(k.CreatedAt),
			UpdatedAt:     formatTime(k.UpdatedAt),
		})
	}
	return out, nil
}

func (s *DBStore) CreateAPIKey(ctx context.Context, userID int, name, keyValue string, readOnly bool) (int, error) {
	user := s.DB.GetUserByID(uint32(userID))
	if user == nil {
		return 0, sql.ErrNoRows
	}
	key, err := s.DB.CreateApiKey(user, name, keyValue, readOnly)
	if err != nil {
		return 0, err
	}
	return int(key.ID), nil
}

func (s *DBStore) DeleteAPIKey(ctx context.Context, id, userID int) error {
	res, err := s.DB.ResilientExec(`DELETE FROM api_keys WHERE id = ? AND user_account_id = ?`, id, userID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *DBStore) GetUserByAPIKey(ctx context.Context, keyValue string) (*store.UserAccountRow, bool, error) {
	type row struct {
		db.UserAccount
		ReadOnly bool `db:"read_only"`
	}
	var u row
	err := s.DB.ResilientGet(&u, `
		SELECT ua.id, ua.created_at, ua.updated_at, ua.username, ua.password_hash, ua.created_by, ua.last_login_at,
		       COALESCE(k.read_only, 0) AS read_only
		FROM api_keys k
		INNER JOIN user_accounts ua ON ua.id = k.user_account_id
		WHERE k.key_value = ?
		LIMIT 1`, keyValue)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("get user by api key: %w", err)
	}
	_ = s.TouchAPIKeyUsed(ctx, keyValue)
	acct := &db.UserAccount{
		Model:        db.Model{ID: u.ID, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt},
		Username:     u.Username,
		PasswordHash: u.PasswordHash,
		CreatedBy:    u.CreatedBy,
		LastLoginAt:  u.LastLoginAt,
	}
	return userRowFromAccount(acct), u.ReadOnly, nil
}

func (s *DBStore) TouchAPIKeyUsed(ctx context.Context, keyValue string) error {
	_, err := s.DB.ResilientExec(`UPDATE api_keys SET last_used_at = NOW(3), updated_at = NOW(3) WHERE key_value = ?`, keyValue)
	return err
}

func (s *DBStore) EnsureUserInEveryoneGroup(ctx context.Context, userID int) error {
	return s.EnsureUserInGroup(ctx, userID, rbac.GroupEveryone)
}

func (s *DBStore) EnsureUserInGroup(ctx context.Context, userID int, groupName string) error {
	g := s.DB.GetUserGroupByName(groupName)
	if g == nil {
		id, err := s.CreateUserGroup(ctx, groupName)
		if err != nil {
			return err
		}
		g = s.DB.GetUserGroupByID(uint32(id))
	}
	if g == nil {
		return fmt.Errorf("group %q not found", groupName)
	}
	return s.DB.AddUserGroupMember(uint32(userID), g.ID)
}

func (s *DBStore) CountUsersWithSuperuserViaGroups(ctx context.Context) (int, error) {
	var count int
	err := s.DB.ResilientGet(&count, `
		SELECT COUNT(DISTINCT ugm.user_account_id)
		FROM user_group_memberships ugm
		INNER JOIN rbac_group_roles gr ON gr.user_group_id = ugm.user_group_id
		INNER JOIN rbac_roles r ON r.id = gr.role_id
		WHERE r.name = ?`, rbac.RoleSuperuser)
	return count, err
}
