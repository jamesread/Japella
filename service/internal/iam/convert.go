package iam

import (
	"database/sql"
	"time"

	"github.com/jamesread/japella/internal/intconv"

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

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("2006-01-02 15:04:05")
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
		out[i] = intconv.Uint32(v)
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
