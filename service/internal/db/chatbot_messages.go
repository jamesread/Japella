package db

import "fmt"

type ChatBotConversationSummary struct {
	ConversationKey   string `db:"conversation_key"`
	ConversationTitle string `db:"conversation_title"`
	LastMessageAtUnix int64  `db:"last_message_at_unix"`
	LastMessage       string `db:"last_message"`
	LastDirection     string `db:"last_direction"`
}

func (db *DB) InsertChatBotMessage(msg *ChatBotMessage) error {
	_, err := db.ResilientNamedExec(
		`INSERT INTO chat_bot_messages
		(connector, identity, conversation_key, conversation_title, channel, author, content, direction, message_id, timestamp_unix, created_at, updated_at)
		VALUES
		(:connector, :identity, :conversation_key, :conversation_title, :channel, :author, :content, :direction, :message_id, :timestamp_unix, NOW(3), NOW(3))`,
		msg,
	)
	if err != nil {
		db.Logger().Errorf("Failed to insert chatbot message: %v", err)
		return err
	}
	return nil
}

func (db *DB) SelectChatBotConversations(connector string, identity string, limit int) ([]*ChatBotConversationSummary, error) {
	ret := make([]*ChatBotConversationSummary, 0)
	if limit <= 0 {
		limit = 100
	}

	query := `SELECT
		conversation_key,
		MAX(conversation_title) AS conversation_title,
		MAX(timestamp_unix) AS last_message_at_unix,
		SUBSTRING_INDEX(GROUP_CONCAT(content ORDER BY timestamp_unix DESC, id DESC SEPARATOR '\n'), '\n', 1) AS last_message,
		SUBSTRING_INDEX(GROUP_CONCAT(direction ORDER BY timestamp_unix DESC, id DESC SEPARATOR '\n'), '\n', 1) AS last_direction
		FROM chat_bot_messages
		WHERE connector = ? AND identity = ?
		GROUP BY conversation_key
		ORDER BY last_message_at_unix DESC
		LIMIT ?`

	err := db.ResilientSelect(&ret, query, connector, identity, limit)
	if err != nil {
		db.Logger().Errorf("Failed to select chatbot conversations: %v", err)
		return nil, err
	}
	return ret, nil
}

func (db *DB) SelectChatBotConversationMessages(connector string, identity string, conversationKey string, limit int) ([]*ChatBotMessage, error) {
	ret := make([]*ChatBotMessage, 0)
	if limit <= 0 {
		limit = 200
	}

	query := `SELECT * FROM chat_bot_messages
		WHERE connector = ? AND identity = ? AND conversation_key = ?
		ORDER BY timestamp_unix ASC, id ASC
		LIMIT ?`

	err := db.ResilientSelect(&ret, query, connector, identity, conversationKey, limit)
	if err != nil {
		db.Logger().Errorf("Failed to select chatbot conversation messages: %v", err)
		return nil, err
	}
	return ret, nil
}

func (db *DB) GetChatBotMessageByExternalID(connector string, identity string, messageID string) (*ChatBotMessage, error) {
	ret := &ChatBotMessage{}
	err := db.ResilientGet(
		ret,
		`SELECT * FROM chat_bot_messages
		WHERE connector = ? AND identity = ? AND message_id = ?
		ORDER BY id DESC
		LIMIT 1`,
		connector, identity, messageID,
	)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func BuildConversationKey(channel string, author string) string {
	return fmt.Sprintf("%s|%s", channel, author)
}
