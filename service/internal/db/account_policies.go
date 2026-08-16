package db

import (
	"database/sql"
	"fmt"
)

const SubmissionSourceMCP = "mcp"
const SubmissionSourceUI = "ui"

const PostStatePendingApproval = "pending_approval"
const PostStateRejected = "rejected"
const PostStateScheduled = "scheduled"
const PostStateDraft = "draft"
const PostStateCompleted = "completed"
const PostStateError = "error"

func (db *DB) CreateAccountPolicy(policy *AccountPolicy) (uint32, error) {
	res, err := db.ResilientExec(
		`INSERT INTO account_policies (name, description, apply_to_mcp, apply_to_ui, created_at, updated_at)
		 VALUES (?, ?, ?, ?, NOW(3), NOW(3))`,
		policy.Name, policy.Description, policy.ApplyToMCP, policy.ApplyToUI)
	if err != nil {
		db.Logger().Errorf("CreateAccountPolicy: %v", err)
		return 0, err
	}
	lid, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return uint32(lid), nil
}

func (db *DB) UpdateAccountPolicy(policy *AccountPolicy) error {
	_, err := db.ResilientExec(
		`UPDATE account_policies SET name = ?, description = ?, apply_to_mcp = ?, apply_to_ui = ?, updated_at = NOW(3) WHERE id = ?`,
		policy.Name, policy.Description, policy.ApplyToMCP, policy.ApplyToUI, policy.ID)
	if err != nil {
		db.Logger().Errorf("UpdateAccountPolicy: %v", err)
		return err
	}
	return nil
}

func (db *DB) DeleteAccountPolicy(id uint32) error {
	res, err := db.ResilientExec(`DELETE FROM account_policies WHERE id = ?`, id)
	if err != nil {
		db.Logger().Errorf("DeleteAccountPolicy: %v", err)
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (db *DB) GetAccountPolicy(id uint32) (*AccountPolicy, error) {
	var p AccountPolicy
	err := db.ResilientGet(&p, `SELECT * FROM account_policies WHERE id = ? LIMIT 1`, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		db.Logger().Errorf("GetAccountPolicy: %v", err)
		return nil, err
	}
	return &p, nil
}

func (db *DB) SelectAccountPolicies() ([]*AccountPolicy, error) {
	ret := make([]*AccountPolicy, 0)
	err := db.ResilientSelect(&ret, `SELECT * FROM account_policies ORDER BY name`)
	if err != nil {
		db.Logger().Errorf("SelectAccountPolicies: %v", err)
		return nil, err
	}
	return ret, nil
}

func (db *DB) GetAccountPolicySocialAccountIDs(policyID uint32) ([]uint32, error) {
	ids := make([]uint32, 0)
	err := db.ResilientSelect(&ids,
		`SELECT social_account_id FROM account_policy_social_accounts WHERE account_policy_id = ? ORDER BY social_account_id`,
		policyID)
	if err != nil {
		db.Logger().Errorf("GetAccountPolicySocialAccountIDs: %v", err)
		return nil, err
	}
	return ids, nil
}

func (db *DB) SetAccountPolicySocialAccounts(policyID uint32, socialAccountIDs []uint32) error {
	if db.connx == nil {
		db.ReconnectDatabaseAndSetErrorMessage()
		return fmt.Errorf("database connection is not established")
	}
	tx, err := db.connx.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err = tx.Exec(`DELETE FROM account_policy_social_accounts WHERE account_policy_id = ?`, policyID); err != nil {
		return err
	}
	for _, saID := range socialAccountIDs {
		if _, err = tx.Exec(
			`INSERT INTO account_policy_social_accounts (account_policy_id, social_account_id, created_at, updated_at) VALUES (?, ?, NOW(3), NOW(3))`,
			policyID, saID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (db *DB) GetAccountPolicyApprovalStages(policyID uint32) ([]*AccountPolicyApprovalStage, error) {
	ret := make([]*AccountPolicyApprovalStage, 0)
	err := db.ResilientSelect(&ret, `
		SELECT s.*, u.username AS username, g.name AS user_group_name
		FROM account_policy_approval_stages s
		LEFT JOIN user_accounts u ON u.id = s.user_id
		LEFT JOIN user_groups g ON g.id = s.user_group_id
		WHERE s.account_policy_id = ?
		ORDER BY s.stage_order`, policyID)
	if err != nil {
		db.Logger().Errorf("GetAccountPolicyApprovalStages: %v", err)
		return nil, err
	}
	return ret, nil
}

func (db *DB) SetAccountPolicyApprovalStages(policyID uint32, stages []*AccountPolicyApprovalStage) error {
	if db.connx == nil {
		db.ReconnectDatabaseAndSetErrorMessage()
		return fmt.Errorf("database connection is not established")
	}
	tx, err := db.connx.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err = tx.Exec(`DELETE FROM account_policy_approval_stages WHERE account_policy_id = ?`, policyID); err != nil {
		return err
	}
	for _, st := range stages {
		if _, err = tx.Exec(
			`INSERT INTO account_policy_approval_stages
			 (account_policy_id, stage_order, user_id, user_group_id, created_at, updated_at)
			 VALUES (?, ?, ?, ?, NOW(3), NOW(3))`,
			policyID, st.StageOrder, nullInt32OrNil(st.UserID), nullInt32OrNil(st.UserGroupID)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func nullInt32OrNil(v sql.NullInt32) any {
	if v.Valid {
		return v.Int32
	}
	return nil
}

// GetAccountPolicyForSocialAccount returns the policy attached to a social account, if any.
func (db *DB) GetAccountPolicyForSocialAccount(socialAccountID uint32) (*AccountPolicy, error) {
	var p AccountPolicy
	err := db.ResilientGet(&p, `
		SELECT ap.* FROM account_policies ap
		INNER JOIN account_policy_social_accounts apsa ON apsa.account_policy_id = ap.id
		WHERE apsa.social_account_id = ?
		LIMIT 1`, socialAccountID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		db.Logger().Errorf("GetAccountPolicyForSocialAccount: %v", err)
		return nil, err
	}
	return &p, nil
}

func (db *DB) GetApprovalStageByPolicyAndOrder(policyID uint32, stageOrder uint32) (*AccountPolicyApprovalStage, error) {
	var st AccountPolicyApprovalStage
	err := db.ResilientGet(&st, `
		SELECT s.*, u.username AS username, g.name AS user_group_name
		FROM account_policy_approval_stages s
		LEFT JOIN user_accounts u ON u.id = s.user_id
		LEFT JOIN user_groups g ON g.id = s.user_group_id
		WHERE s.account_policy_id = ? AND s.stage_order = ?
		LIMIT 1`, policyID, stageOrder)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		db.Logger().Errorf("GetApprovalStageByPolicyAndOrder: %v", err)
		return nil, err
	}
	return &st, nil
}

func (db *DB) CountApprovalStages(policyID uint32) (int, error) {
	var n int
	err := db.ResilientGet(&n, `SELECT COUNT(*) FROM account_policy_approval_stages WHERE account_policy_id = ?`, policyID)
	return n, err
}

func (db *DB) IsUserInUserGroup(userID uint32, groupID uint32) (bool, error) {
	var n int
	err := db.ResilientGet(&n,
		`SELECT COUNT(*) FROM user_group_memberships WHERE user_account_id = ? AND user_group_id = ?`,
		userID, groupID)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// CanUserApproveStage returns whether userID may approve the given stage (ignores submitter check).
func (db *DB) CanUserApproveStage(userID uint32, stage *AccountPolicyApprovalStage) (bool, error) {
	if stage == nil {
		return false, nil
	}
	if stage.UserID.Valid && uint32(stage.UserID.Int32) == userID {
		return true, nil
	}
	if stage.UserGroupID.Valid {
		return db.IsUserInUserGroup(userID, uint32(stage.UserGroupID.Int32))
	}
	return false, nil
}

func (db *DB) RecordPostApproval(postID uint32, stageID uint32, approvedByUserID uint32) error {
	_, err := db.ResilientExec(
		`INSERT INTO post_approvals (post_id, stage_id, approved_by_user_id, created_at) VALUES (?, ?, ?, NOW(3))`,
		postID, stageID, approvedByUserID)
	if err != nil {
		db.Logger().Errorf("RecordPostApproval: %v", err)
		return err
	}
	return nil
}

func (db *DB) AdvancePostApprovalStage(postID uint32, nextStage uint32) error {
	_, err := db.ResilientExec(
		`UPDATE posts SET approval_stage = ?, updated_at = NOW(3) WHERE id = ?`,
		nextStage, postID)
	return err
}

func (db *DB) SetPostState(postID uint32, state string) error {
	_, err := db.ResilientExec(`UPDATE posts SET state = ?, updated_at = NOW(3) WHERE id = ?`, state, postID)
	return err
}

func (db *DB) MarkPostScheduled(postID uint32) error {
	return db.SetPostState(postID, PostStateScheduled)
}

// SelectPendingApprovalsVisibleToUser returns pending_approval posts the user should see:
// - all pending posts if viewAll is true
// - otherwise posts they submitted, or where they are the current-stage assignee (user or group)
func (db *DB) SelectPendingApprovalsVisibleToUser(userID uint32, viewAll bool) ([]*Post, error) {
	ret := make([]*Post, 0)
	query := `
		SELECT p.id, p.social_account_id, p.status, p.state, p.content, p.post_url, p.remote_id,
		       p.scheduled_at, p.created_at, p.campaign_id AS campaign_id, c.name AS campaign_name,
		       p.submission_source, p.submitted_by_user_id, p.account_policy_id, p.approval_stage
		FROM posts p
		LEFT JOIN campaigns c ON p.campaign_id = c.id
		LEFT JOIN account_policy_approval_stages s
		  ON s.account_policy_id = p.account_policy_id AND s.stage_order = p.approval_stage
		WHERE p.state = ?`
	args := []any{PostStatePendingApproval}

	if !viewAll {
		query += `
		  AND (
		    p.submitted_by_user_id = ?
		    OR s.user_id = ?
		    OR (s.user_group_id IS NOT NULL AND EXISTS (
		      SELECT 1 FROM user_group_memberships m
		      WHERE m.user_group_id = s.user_group_id AND m.user_account_id = ?
		    ))
		  )`
		args = append(args, userID, userID, userID)
	}

	query += ` ORDER BY p.id DESC`

	err := db.ResilientSelect(&ret, query, args...)
	if err != nil {
		db.Logger().Errorf("SelectPendingApprovalsVisibleToUser: %v", err)
		return nil, err
	}
	return ret, nil
}
