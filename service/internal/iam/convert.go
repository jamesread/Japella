package iam

import (
	"database/sql"
	"time"

	"github.com/jamesread/armature-iam/rbac"
	"github.com/jamesread/armature-iam/store"
	"github.com/jamesread/japella/internal/db"
)

func userRowFromAccount(u *db.UserAccount) *store.UserAccountRow {
	if u == nil {
		return nil
	}
	return &store.UserAccountRow{
		ID:           int(u.ID),
		Username:     u.Username,
		PasswordHash: u.PasswordHash,
		CreatedBy:    u.CreatedBy,
		CreatedAt:    formatTime(u.CreatedAt),
		UpdatedAt:    formatTime(u.UpdatedAt),
	}
}

func userAccountFromRow(u *store.UserAccountRow) *db.UserAccount {
	if u == nil {
		return nil
	}
	return &db.UserAccount{
		Model:        db.Model{ID: uint32(u.ID), CreatedAt: parseTime(u.CreatedAt), UpdatedAt: parseTime(u.UpdatedAt)},
		Username:     u.Username,
		PasswordHash: u.PasswordHash,
		CreatedBy:    u.CreatedBy,
	}
}

func effectiveRBACFromArmature(e *rbac.EffectiveRBAC) *db.EffectiveRBAC {
	if e == nil {
		return nil
	}
	perms := make(map[string]bool, len(e.Permissions))
	for k, v := range e.Permissions {
		perms[k] = v
	}
	return &db.EffectiveRBAC{
		IsSuperuser: e.IsSuperuser,
		Permissions: perms,
		RoleNames:   append([]string(nil), e.RoleNames...),
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("2006-01-02 15:04:05")
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse("2006-01-02 15:04:05", s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func formatNullTime(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return formatTime(t.Time)
}

func intSliceToUint32(in []int) []uint32 {
	out := make([]uint32, len(in))
	for i, v := range in {
		out[i] = uint32(v)
	}
	return out
}

func uint32SliceToInt(in []uint32) []int {
	out := make([]int, len(in))
	for i, v := range in {
		out[i] = int(v)
	}
	return out
}
