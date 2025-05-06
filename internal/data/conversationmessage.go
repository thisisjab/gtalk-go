package data

import (
	"context"
	"database/sql"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/thisisjab/gchat-go/internal/filter"
	filters "github.com/thisisjab/gchat-go/internal/filter"
	"github.com/thisisjab/gchat-go/internal/validator"
)

const (
	TypeTextMessage  = "text"
	TypeImageMessage = "image"
	TypeVideoMessage = "video"
	TypeAudioMessage = "audio"
	TypeFileMessage  = "file"
)

type ConversationMessageModel struct {
	DB *sql.DB
}

type ConversationMessage struct {
	BaseModel
	ConversationID uuid.UUID `json:"-"`
	SenderID       uuid.UUID `json:"sender_id,omitempty"`
	Content        string    `json:"content"`
	Type           string    `json:"type"`
	CreatedAt      time.Time `json:"created_at"`
	// TODO: add attachment
}

type ConversationMessageWithSender struct {
	ConversationMessage
	// Since sender id is omitted when empty, don't need exclude here; it's already excluded in the query.
	Sender User `json:"sender"`
}

func ValidateConversationMessage(v *validator.Validator, cm *ConversationMessage) {
	v.Check(cm.ConversationID != uuid.Nil, "conversation_id", "must be provided")
	v.Check(cm.SenderID != uuid.Nil, "sender_id", "must be provided")

	v.Check(cm.Type != "", "type", "must be provided")
	v.Check(slices.Contains([]string{TypeTextMessage, TypeImageMessage, TypeVideoMessage, TypeAudioMessage, TypeFileMessage}, cm.Type), "type", "must be one of text, image, video, audio, or file")

	v.Check(cm.Content != "", "content", "must be provided")
	v.Check(len(cm.Content) <= 500, "content", "must not be more than 500 bytes long")
}

func (cmm *ConversationMessageModel) GetAllForPrivate(conversationID uuid.UUID, f filter.Filters) ([]*ConversationMessage, *filter.PaginationMetadata, error) {
	query := `
	SELECT count(*) OVER(), id, sender_id, type, content, created_at, updated_at
	FROM conversation_messages
	WHERE conversation_id = $1
	ORDER BY created_at DESC, id DESC
	LIMIT $2 OFFSET $3
	`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := cmm.DB.QueryContext(ctx, query, conversationID, f.Limit(), f.Offset())
	if err != nil {
		return nil, nil, err
	}

	defer rows.Close()

	messages := make([]*ConversationMessage, 0)
	totalRecords := 0

	for rows.Next() {
		m := &ConversationMessage{ConversationID: conversationID}

		err = rows.Scan(
			&totalRecords,
			&m.ID,
			&m.SenderID,
			&m.Type,
			&m.Content,
			&m.CreatedAt,
			&m.UpdatedAt,
		)

		if err != nil {
			return nil, nil, err
		}

		messages = append(messages, m)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	paginationMetadata, err := filters.CalculatePaginationMetadata(totalRecords, f.Page, f.PageSize)
	if err != nil {
		return nil, nil, err
	}

	return messages, paginationMetadata, nil
}

func (cmm *ConversationMessageModel) GetAllForGroup(conversationID uuid.UUID, f filter.Filters) ([]*ConversationMessageWithSender, *filter.PaginationMetadata, error) {
	query := `
	SELECT
		count(*) OVER(),
		m.id,
		m.type,
		m.content,
		m.created_at,
		m.updated_at,
		u.id,
		u.username,
		u.email,
		u.bio,
		u.is_active
	FROM conversation_messages m
	JOIN users u ON u.id = m.sender_id
	WHERE m.conversation_id = $1
	LIMIT $2 OFFSET $3
	`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := cmm.DB.QueryContext(ctx, query, conversationID, f.Limit(), f.Offset())
	if err != nil {
		return nil, nil, err
	}

	defer rows.Close()

	messages := make([]*ConversationMessageWithSender, 0)
	totalRecords := 0

	for rows.Next() {
		m := &ConversationMessageWithSender{
			ConversationMessage: ConversationMessage{
				BaseModel: BaseModel{
					ID: conversationID,
				},
			},
		}

		err = rows.Scan(
			&totalRecords,
			&m.ID,
			&m.Type,
			&m.Content,
			&m.CreatedAt,
			&m.UpdatedAt,
			&m.Sender.ID,
			&m.Sender.Username,
			&m.Sender.Email,
			&m.Sender.Bio,
			&m.Sender.IsActive,
		)

		if err != nil {
			return nil, nil, err
		}

		messages = append(messages, m)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	paginationMetadata, err := filters.CalculatePaginationMetadata(totalRecords, f.Page, f.PageSize)
	if err != nil {
		return nil, nil, err
	}

	return messages, paginationMetadata, nil
}

func (cmm *ConversationMessageModel) Insert(message *ConversationMessage) error {
	query := `
	INSERT INTO conversation_messages (conversation_id, sender_id, type, content)
	VALUES ($1, $2, $3, $4)
	RETURNING id, created_at, updated_at
	`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	args := []any{message.ConversationID, message.SenderID, message.Type, message.Content}

	return cmm.DB.QueryRowContext(ctx, query, args...).Scan(&message.ID, &message.CreatedAt, &message.UpdatedAt)
}
