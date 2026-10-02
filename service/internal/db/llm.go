package db

import (
	"database/sql"
	"fmt"
)

type LlmConnection struct {
	Model
	Name      string `db:"name"`
	BaseURL   string `db:"base_url"`
	APIKey    string `db:"api_key"`
	ModelName string `db:"model"`
	Enabled   bool   `db:"enabled"`
}

type LlmTool struct {
	Model
	Name             string `db:"name"`
	Description      string `db:"description"`
	LlmConnectionID  uint32 `db:"llm_connection_id"`
	Prompt           string `db:"prompt"`
	Enabled          bool   `db:"enabled"`
	SortOrder        int32  `db:"sort_order"`
	ConnectionName   string `db:"connection_name"`
}

type LlmToolRun struct {
	ID               uint32         `db:"id"`
	PostID           uint32         `db:"post_id"`
	LlmToolID        sql.NullInt32  `db:"llm_tool_id"`
	ToolName         string         `db:"tool_name"`
	ConnectionName   string         `db:"connection_name"`
	ModelName        string         `db:"model"`
	InputContent     string         `db:"input_content"`
	OutputContent    string         `db:"output_content"`
	Status           string         `db:"status"`
	ErrorMessage     string         `db:"error_message"`
	CreatedByUserID  sql.NullInt32  `db:"created_by_user_id"`
	CreatedAt        sql.NullTime   `db:"created_at"`
}

const (
	LlmToolRunStatusOK    = "ok"
	LlmToolRunStatusError = "error"
)

func (db *DB) ListLlmConnections() ([]*LlmConnection, error) {
	ret := make([]*LlmConnection, 0)
	err := db.ResilientSelect(&ret, `
		SELECT id, name, base_url, api_key, model, enabled, created_at, updated_at
		FROM llm_connections
		ORDER BY name ASC`)
	if err != nil {
		db.Logger().Errorf("ListLlmConnections: %v", err)
		return nil, err
	}
	return ret, nil
}

func (db *DB) GetLlmConnection(id uint32) (*LlmConnection, error) {
	var c LlmConnection
	err := db.ResilientGet(&c, `
		SELECT id, name, base_url, api_key, model, enabled, created_at, updated_at
		FROM llm_connections WHERE id = ?`, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		db.Logger().Errorf("GetLlmConnection: %v", err)
		return nil, err
	}
	return &c, nil
}

func (db *DB) CreateLlmConnection(name, baseURL, apiKey, model string, enabled bool) (uint32, error) {
	enabledInt := 0
	if enabled {
		enabledInt = 1
	}
	res, err := db.ResilientExec(`
		INSERT INTO llm_connections (name, base_url, api_key, model, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, NOW(3), NOW(3))`, name, baseURL, apiKey, model, enabledInt)
	if err != nil {
		db.Logger().Errorf("CreateLlmConnection: %v", err)
		return 0, err
	}
	lastID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return uint32(lastID), nil
}

func (db *DB) UpdateLlmConnection(id uint32, name, baseURL, apiKey, model string, enabled bool, keepAPIKey bool) error {
	enabledInt := 0
	if enabled {
		enabledInt = 1
	}
	var err error
	if keepAPIKey {
		_, err = db.ResilientExec(`
			UPDATE llm_connections
			SET name = ?, base_url = ?, model = ?, enabled = ?, updated_at = NOW(3)
			WHERE id = ?`, name, baseURL, model, enabledInt, id)
	} else {
		_, err = db.ResilientExec(`
			UPDATE llm_connections
			SET name = ?, base_url = ?, api_key = ?, model = ?, enabled = ?, updated_at = NOW(3)
			WHERE id = ?`, name, baseURL, apiKey, model, enabledInt, id)
	}
	if err != nil {
		db.Logger().Errorf("UpdateLlmConnection: %v", err)
		return err
	}
	return nil
}

func (db *DB) DeleteLlmConnection(id uint32) error {
	_, err := db.ResilientExec(`DELETE FROM llm_connections WHERE id = ?`, id)
	if err != nil {
		db.Logger().Errorf("DeleteLlmConnection: %v", err)
		return err
	}
	return nil
}

func (db *DB) ListLlmTools() ([]*LlmTool, error) {
	ret := make([]*LlmTool, 0)
	err := db.ResilientSelect(&ret, `
		SELECT t.id, t.name, t.description, t.llm_connection_id, t.prompt, t.enabled, t.sort_order,
		       t.created_at, t.updated_at, COALESCE(c.name, '') AS connection_name
		FROM llm_tools t
		LEFT JOIN llm_connections c ON c.id = t.llm_connection_id
		ORDER BY t.sort_order ASC, t.name ASC`)
	if err != nil {
		db.Logger().Errorf("ListLlmTools: %v", err)
		return nil, err
	}
	return ret, nil
}

func (db *DB) ListEnabledLlmTools() ([]*LlmTool, error) {
	ret := make([]*LlmTool, 0)
	err := db.ResilientSelect(&ret, `
		SELECT t.id, t.name, t.description, t.llm_connection_id, t.prompt, t.enabled, t.sort_order,
		       t.created_at, t.updated_at, COALESCE(c.name, '') AS connection_name
		FROM llm_tools t
		INNER JOIN llm_connections c ON c.id = t.llm_connection_id
		WHERE t.enabled = 1 AND c.enabled = 1
		ORDER BY t.sort_order ASC, t.name ASC`)
	if err != nil {
		db.Logger().Errorf("ListEnabledLlmTools: %v", err)
		return nil, err
	}
	return ret, nil
}

func (db *DB) GetLlmTool(id uint32) (*LlmTool, error) {
	var t LlmTool
	err := db.ResilientGet(&t, `
		SELECT t.id, t.name, t.description, t.llm_connection_id, t.prompt, t.enabled, t.sort_order,
		       t.created_at, t.updated_at, COALESCE(c.name, '') AS connection_name
		FROM llm_tools t
		LEFT JOIN llm_connections c ON c.id = t.llm_connection_id
		WHERE t.id = ?`, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		db.Logger().Errorf("GetLlmTool: %v", err)
		return nil, err
	}
	return &t, nil
}

func (db *DB) CreateLlmTool(name, description string, connectionID uint32, prompt string, enabled bool, sortOrder int32) (uint32, error) {
	enabledInt := 0
	if enabled {
		enabledInt = 1
	}
	res, err := db.ResilientExec(`
		INSERT INTO llm_tools (name, description, llm_connection_id, prompt, enabled, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, NOW(3), NOW(3))`,
		name, description, connectionID, prompt, enabledInt, sortOrder)
	if err != nil {
		db.Logger().Errorf("CreateLlmTool: %v", err)
		return 0, err
	}
	lastID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return uint32(lastID), nil
}

func (db *DB) UpdateLlmTool(id uint32, name, description string, connectionID uint32, prompt string, enabled bool, sortOrder int32) error {
	enabledInt := 0
	if enabled {
		enabledInt = 1
	}
	_, err := db.ResilientExec(`
		UPDATE llm_tools
		SET name = ?, description = ?, llm_connection_id = ?, prompt = ?, enabled = ?, sort_order = ?, updated_at = NOW(3)
		WHERE id = ?`,
		name, description, connectionID, prompt, enabledInt, sortOrder, id)
	if err != nil {
		db.Logger().Errorf("UpdateLlmTool: %v", err)
		return err
	}
	return nil
}

func (db *DB) DeleteLlmTool(id uint32) error {
	_, err := db.ResilientExec(`DELETE FROM llm_tools WHERE id = ?`, id)
	if err != nil {
		db.Logger().Errorf("DeleteLlmTool: %v", err)
		return err
	}
	return nil
}

func (db *DB) CountLlmToolsForConnection(connectionID uint32) (int, error) {
	var count int
	err := db.ResilientGet(&count, `SELECT COUNT(*) FROM llm_tools WHERE llm_connection_id = ?`, connectionID)
	if err != nil {
		db.Logger().Errorf("CountLlmToolsForConnection: %v", err)
		return 0, err
	}
	return count, nil
}

func (db *DB) CreateLlmToolRun(run *LlmToolRun) (uint32, error) {
	if run == nil {
		return 0, fmt.Errorf("run is required")
	}
	res, err := db.ResilientExec(`
		INSERT INTO llm_tool_runs (
			post_id, llm_tool_id, tool_name, connection_name, model, input_content, output_content,
			status, error_message, created_by_user_id, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(3))`,
		run.PostID, run.LlmToolID, run.ToolName, run.ConnectionName, run.ModelName, run.InputContent, run.OutputContent,
		run.Status, run.ErrorMessage, run.CreatedByUserID)
	if err != nil {
		db.Logger().Errorf("CreateLlmToolRun: %v", err)
		return 0, err
	}
	lastID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return uint32(lastID), nil
}

func (db *DB) ListLlmToolRunsByPost(postID uint32) ([]*LlmToolRun, error) {
	ret := make([]*LlmToolRun, 0)
	err := db.ResilientSelect(&ret, `
		SELECT id, post_id, llm_tool_id, tool_name, connection_name, model, input_content, output_content,
		       status, error_message, created_by_user_id, created_at
		FROM llm_tool_runs
		WHERE post_id = ?
		ORDER BY id DESC`, postID)
	if err != nil {
		db.Logger().Errorf("ListLlmToolRunsByPost: %v", err)
		return nil, err
	}
	return ret, nil
}

func (db *DB) CopyLlmToolRuns(fromPostID, toPostID uint32) error {
	if fromPostID == 0 || toPostID == 0 || fromPostID == toPostID {
		return nil
	}
	_, err := db.ResilientExec(`
		INSERT INTO llm_tool_runs (
			post_id, llm_tool_id, tool_name, connection_name, model, input_content, output_content,
			status, error_message, created_by_user_id, created_at
		)
		SELECT ?, llm_tool_id, tool_name, connection_name, model, input_content, output_content,
		       status, error_message, created_by_user_id, created_at
		FROM llm_tool_runs
		WHERE post_id = ?
		ORDER BY id ASC`, toPostID, fromPostID)
	if err != nil {
		db.Logger().Errorf("CopyLlmToolRuns from %d to %d: %v", fromPostID, toPostID, err)
		return err
	}
	return nil
}
