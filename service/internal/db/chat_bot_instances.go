package db

import (
	"fmt"
	"strings"
)

func (db *DB) SelectChatBotInstances() ([]*ChatBotInstance, error) {
	ret := make([]*ChatBotInstance, 0)
	err := db.ResilientSelect(&ret, "SELECT * FROM chat_bot_instances ORDER BY protocol ASC, bot_id ASC")
	if err != nil {
		db.Logger().Errorf("Failed to select chat bot instances: %v", err)
		return nil, err
	}
	return ret, nil
}

func (db *DB) GetChatBotInstance(protocol, botID string) (*ChatBotInstance, error) {
	var inst ChatBotInstance
	err := db.ResilientGet(&inst, "SELECT * FROM chat_bot_instances WHERE protocol = ? AND bot_id = ?", protocol, botID)
	if err != nil {
		return nil, err
	}
	return &inst, nil
}

func (db *DB) CreateChatBotInstance(inst *ChatBotInstance) error {
	_, err := db.ResilientNamedExec(
		`INSERT INTO chat_bot_instances (protocol, bot_id, display_name, created_at, updated_at)
		 VALUES (:protocol, :bot_id, :display_name, NOW(), NOW())`,
		inst,
	)
	if err != nil {
		db.Logger().Errorf("Failed to create chat bot instance: %v", err)
		return err
	}
	return nil
}

func (db *DB) UpdateChatBotInstanceDisplayName(protocol, botID, displayName string) error {
	_, err := db.ResilientExec(
		`UPDATE chat_bot_instances SET display_name = ?, updated_at = NOW() WHERE protocol = ? AND bot_id = ?`,
		displayName, protocol, botID,
	)
	if err != nil {
		db.Logger().Errorf("Failed to update chat bot instance display name: %v", err)
		return err
	}
	return nil
}

func (db *DB) DeleteChatBotInstance(protocol, botID string) error {
	_, err := db.ResilientExec(
		`DELETE FROM chat_bot_instances WHERE protocol = ? AND bot_id = ?`,
		protocol, botID,
	)
	if err != nil {
		db.Logger().Errorf("Failed to delete chat bot instance: %v", err)
		return err
	}
	return nil
}

func (db *DB) DeleteCvarsByKeys(keys []string) error {
	for _, key := range keys {
		if key == "" {
			continue
		}
		_, err := db.ResilientExec(`DELETE FROM cvars WHERE key_name = ?`, key)
		if err != nil {
			return fmt.Errorf("delete cvar %s: %w", key, err)
		}
	}
	return nil
}

func (db *DB) UpsertBotCvar(key, title, description, cvarType, value string) error {
	existing := db.GetCvar(key)
	if existing != nil {
		return db.SetCvarString(key, value)
	}

	cvar := &Cvar{
		KeyName:     key,
		Title:       title,
		ValueString: value,
		Description: description,
		Category:    "Chat Bots",
		Type:        cvarType,
	}
	return db.InsertCvarIfNotExists(cvar)
}

func (db *DB) InsertBotPasswordCvar(key, title, description, value string) error {
	existing := db.GetCvar(key)
	if existing == nil {
		cvar := &Cvar{
			KeyName:     key,
			Title:       title,
			ValueString: value,
			Description: description,
			Category:    "Chat Bots",
			Type:        "password",
		}
		if err := db.InsertCvarIfNotExists(cvar); err != nil {
			return err
		}
	}
	if value != "" {
		return db.SetCvarString(key, value)
	}
	return nil
}

func (db *DB) InsertBotTextCvar(key, title, description, value string) error {
	existing := db.GetCvar(key)
	if existing == nil {
		cvar := &Cvar{
			KeyName:     key,
			Title:       title,
			ValueString: value,
			Description: description,
			Category:    "Chat Bots",
			Type:        "text",
		}
		if err := db.InsertCvarIfNotExists(cvar); err != nil {
			return err
		}
	}
	if value != "" {
		return db.SetCvarString(key, value)
	}
	return nil
}

func MaskPasswordCvarValue(cvarType, value string) string {
	if cvarType == "password" && value != "" {
		return strings.Repeat("*", 8)
	}
	return value
}
