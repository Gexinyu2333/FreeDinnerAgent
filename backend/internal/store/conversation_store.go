package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Conversation struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Title     string    `json:"title"`
	Channel   string    `json:"channel"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Message struct {
	ID             string          `json:"id"`
	ConversationID string          `json:"conversation_id"`
	UserID         string          `json:"user_id"`
	Role           string          `json:"role"`
	Content        string          `json:"content"`
	TokenCount     int             `json:"token_count"`
	IsAnchor       bool            `json:"is_anchor"`
	AnchorReason   *string         `json:"anchor_reason"`
	Metadata       json.RawMessage `json:"metadata"`
	CreatedAt      time.Time       `json:"created_at"`
}

type SendMessageResult struct {
	TurnID           string  `json:"turn_id"`
	UserMessage      Message `json:"user_message"`
	AssistantMessage Message `json:"assistant_message"`
}

type ConversationStore struct {
	db *pgxpool.Pool
}

func NewConversationStore(db *pgxpool.Pool) *ConversationStore {
	return &ConversationStore{db: db}
}

func (s *ConversationStore) Create(ctx context.Context, userID, title string) (Conversation, error) {
	query := `
		INSERT INTO conversations (id, user_id, title, channel)
		VALUES ($1, $2, $3, 'web')
		RETURNING id, user_id, title, channel, status, created_at, updated_at
	`
	return scanConversation(s.db.QueryRow(ctx, query, uuid.NewString(), userID, title))
}

func (s *ConversationStore) List(ctx context.Context, userID string) ([]Conversation, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, title, channel, status, created_at, updated_at
		FROM conversations
		WHERE user_id = $1 AND status <> 'deleted'
		ORDER BY updated_at DESC, created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	conversations := make([]Conversation, 0)
	for rows.Next() {
		conversation, err := scanConversation(rows)
		if err != nil {
			return nil, err
		}
		conversations = append(conversations, conversation)
	}
	return conversations, rows.Err()
}

func (s *ConversationStore) FindByID(ctx context.Context, userID, conversationID string) (Conversation, error) {
	query := `
		SELECT id, user_id, title, channel, status, created_at, updated_at
		FROM conversations
		WHERE id = $1 AND user_id = $2 AND status <> 'deleted'
	`
	return scanConversation(s.db.QueryRow(ctx, query, conversationID, userID))
}

func (s *ConversationStore) ListMessages(ctx context.Context, userID, conversationID string) ([]Message, error) {
	if _, err := s.FindByID(ctx, userID, conversationID); err != nil {
		return nil, err
	}

	rows, err := s.db.Query(ctx, `
		SELECT id, conversation_id, user_id, role, content, token_count, is_anchor, anchor_reason, metadata, created_at
		FROM messages
		WHERE conversation_id = $1 AND user_id = $2
		ORDER BY created_at ASC
	`, conversationID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := make([]Message, 0)
	for rows.Next() {
		message, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	return messages, rows.Err()
}

func (s *ConversationStore) CreateUserMessage(ctx context.Context, userID, conversationID, content string) (Message, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Message{}, err
	}
	defer tx.Rollback(ctx)

	if _, err := scanConversation(tx.QueryRow(ctx, `
		SELECT id, user_id, title, channel, status, created_at, updated_at
		FROM conversations
		WHERE id = $1 AND user_id = $2 AND status = 'active'
	`, conversationID, userID)); err != nil {
		return Message{}, err
	}

	userMessage, err := scanMessage(tx.QueryRow(ctx, `
		INSERT INTO messages (id, conversation_id, user_id, role, content, metadata)
		VALUES ($1, $2, $3, 'user', $4, '{}'::jsonb)
		RETURNING id, conversation_id, user_id, role, content, token_count, is_anchor, anchor_reason, metadata, created_at
	`, uuid.NewString(), conversationID, userID, content))
	if err != nil {
		return Message{}, err
	}

	if _, err := tx.Exec(ctx, `UPDATE conversations SET updated_at = NOW() WHERE id = $1 AND user_id = $2`, conversationID, userID); err != nil {
		return Message{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Message{}, err
	}

	return userMessage, nil
}

func (s *ConversationStore) CreateAssistantMessage(ctx context.Context, userID, conversationID, content string, metadata json.RawMessage) (Message, error) {
	if len(metadata) == 0 {
		metadata = json.RawMessage(`{}`)
	}

	message, err := scanMessage(s.db.QueryRow(ctx, `
		INSERT INTO messages (id, conversation_id, user_id, role, content, metadata)
		VALUES ($1, $2, $3, 'assistant', $4, $5)
		RETURNING id, conversation_id, user_id, role, content, token_count, is_anchor, anchor_reason, metadata, created_at
	`, uuid.NewString(), conversationID, userID, content, metadata))
	if err != nil {
		return Message{}, err
	}

	if _, err := s.db.Exec(ctx, `UPDATE conversations SET updated_at = NOW() WHERE id = $1 AND user_id = $2`, conversationID, userID); err != nil {
		return Message{}, err
	}
	return message, nil
}

func scanConversation(row pgx.Row) (Conversation, error) {
	var conversation Conversation
	if err := row.Scan(
		&conversation.ID,
		&conversation.UserID,
		&conversation.Title,
		&conversation.Channel,
		&conversation.Status,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Conversation{}, ErrNotFound
		}
		return Conversation{}, err
	}
	return conversation, nil
}

func scanMessage(row pgx.Row) (Message, error) {
	var message Message
	if err := row.Scan(
		&message.ID,
		&message.ConversationID,
		&message.UserID,
		&message.Role,
		&message.Content,
		&message.TokenCount,
		&message.IsAnchor,
		&message.AnchorReason,
		&message.Metadata,
		&message.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Message{}, ErrNotFound
		}
		return Message{}, err
	}
	return message, nil
}
