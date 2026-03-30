package db

import (
	"database/sql"
	"fmt"
)

type UserGroupWithCount struct {
	UserGroup
	MemberCount uint32 `db:"member_count"`
}

func (db *DB) SelectUserGroups() ([]*UserGroupWithCount, error) {
	ret := make([]*UserGroupWithCount, 0)
	err := db.ResilientSelect(&ret, `
		SELECT g.*, COUNT(m.user_account_id) AS member_count
		FROM user_groups g
		LEFT JOIN user_group_memberships m ON m.user_group_id = g.id
		GROUP BY g.id
		ORDER BY g.name`)
	if err != nil {
		db.Logger().Errorf("Failed to select user groups: %v", err)
		return nil, err
	}
	return ret, nil
}

func (db *DB) CreateUserGroup(name string) (uint32, error) {
	res, err := db.ResilientExec(
		`INSERT INTO user_groups (name, created_at, updated_at) VALUES (?, NOW(3), NOW(3))`,
		name)
	if err != nil {
		return 0, err
	}
	lid, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return uint32(lid), nil
}

func (db *DB) DeleteUserGroup(id uint32) error {
	if db.connx == nil {
		db.ReconnectDatabaseAndSetErrorMessage()
		return fmt.Errorf("database connection is not established")
	}

	tx, err := db.connx.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err = tx.Exec(`DELETE FROM user_group_memberships WHERE user_group_id = ?`, id); err != nil {
		return err
	}
	res, err := tx.Exec(`DELETE FROM user_groups WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return tx.Commit()
}

func (db *DB) SelectUserGroupMemberIDs(groupID uint32) ([]uint32, error) {
	var ids []uint32
	err := db.ResilientSelect(&ids,
		`SELECT user_account_id FROM user_group_memberships WHERE user_group_id = ? ORDER BY user_account_id`,
		groupID)
	return ids, err
}

func (db *DB) SetUserGroupMembers(groupID uint32, userIDs []uint32) error {
	if db.connx == nil {
		db.ReconnectDatabaseAndSetErrorMessage()
		return fmt.Errorf("database connection is not established")
	}

	tx, err := db.connx.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err = tx.Exec(`DELETE FROM user_group_memberships WHERE user_group_id = ?`, groupID); err != nil {
		return err
	}
	for _, uid := range userIDs {
		if _, err = tx.Exec(
			`INSERT INTO user_group_memberships (user_account_id, user_group_id, created_at, updated_at) VALUES (?, ?, NOW(3), NOW(3))`,
			uid, groupID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (db *DB) GetUserGroupByID(id uint32) *UserGroup {
	var g UserGroup
	err := db.ResilientGet(&g, `SELECT * FROM user_groups WHERE id = ? LIMIT 1`, id)
	if err != nil {
		if err != sql.ErrNoRows {
			db.Logger().Errorf("GetUserGroupByID: %v", err)
		}
		return nil
	}
	return &g
}
