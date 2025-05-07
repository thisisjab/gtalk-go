package data

import (
	"context"
	"database/sql"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/thisisjab/gchat-go/internal/filter"
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
	ConversationID   uuid.UUID  `json:"-"`
	SenderID         uuid.UUID  `json:"sender_id,omitempty"`
	Content          string     `json:"content"`
	RepliedMessageID *uuid.UUID `json:"-"`
	Type             string     `json:"type"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	// TODO: add attachment
}

type ConversationMessageWithRepliedMessage struct {
	ConversationMessage
	RepliedMessage *ConversationMessage `json:"replied_message"`
}

type ConversationMessageWithRepliedMessageAndSender struct {
	ConversationMessage
	// Since sender id is omitted when empty, don't need exclude here; it's already excluded in the query.
	Sender         User                 `json:"sender"`
	RepliedMessage *ConversationMessage `json:"replied_message"`
}

func ValidateConversationMessage(v *validator.Validator, cm *ConversationMessage) {
	v.Check(cm.ConversationID != uuid.Nil, "conversation_id", "must be provided")
	v.Check(cm.SenderID != uuid.Nil, "sender_id", "must be provided")

	v.Check(cm.Type != "", "type", "must be provided")
	v.Check(slices.Contains([]string{TypeTextMessage, TypeImageMessage, TypeVideoMessage, TypeAudioMessage, TypeFileMessage}, cm.Type), "type", "must be one of text, image, video, audio, or file")

	v.Check(cm.Content != "", "content", "must be provided")
	v.Check(len(cm.Content) <= 500, "content", "must not be more than 500 bytes long")
}

func (cmm *ConversationMessageModel) GetAllForPrivate(conversationID uuid.UUID, f filter.Filters) ([]*ConversationMessageWithRepliedMessage, *filter.PaginationMetadata, error) {
	query := `
	SELECT
		count(*) OVER(),
		cm.id, cm.sender_id, cm.type, cm.content, cm.created_at, cm.updated_at,
		r.id, r.sender_id, r.type, r.content, r.created_at, r.updated_at
	FROM conversation_messages cm
	LEFT JOIN conversation_messages r
	ON cm.replied_message_id = r.id
	WHERE cm.conversation_id = $1
	ORDER BY cm.created_at DESC, cm.id DESC
	LIMIT $2 OFFSET $3
	`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := cmm.DB.QueryContext(ctx, query, conversationID, f.Limit(), f.Offset())
	if err != nil {
		return nil, nil, err
	}

	defer rows.Close()

	messages := make([]*ConversationMessageWithRepliedMessage, 0)
	totalRecords := 0

	for rows.Next() {
		m := &ConversationMessageWithRepliedMessage{
			ConversationMessage: ConversationMessage{
				BaseModel: BaseModel{
					ID: conversationID,
				},
			},
		}

		var (
			repliedMessageID        *uuid.UUID
			repliedMessageSenderID  *uuid.UUID
			repliedMessageType      *string
			repliedMessageContent   *string
			repliedMessageCreatedAt *time.Time
			repliedMessageUpdatedAt *time.Time
		)

		err = rows.Scan(
			&totalRecords,
			&m.ID,
			&m.SenderID,
			&m.Type,
			&m.Content,
			&m.CreatedAt,
			&m.UpdatedAt,
			&repliedMessageID,
			&repliedMessageSenderID,
			&repliedMessageType,
			&repliedMessageContent,
			&repliedMessageCreatedAt,
			&repliedMessageUpdatedAt,
		)

		if err != nil {
			return nil, nil, err
		}

		if repliedMessageID != nil {
			m.RepliedMessage = &ConversationMessage{
				BaseModel: BaseModel{
					ID: *repliedMessageID,
				},
				ConversationID: conversationID,
				SenderID:       *repliedMessageSenderID,
				Type:           *repliedMessageType,
				Content:        *repliedMessageContent,
				CreatedAt:      *repliedMessageCreatedAt,
				UpdatedAt:      *repliedMessageUpdatedAt,
			}
		}

		messages = append(messages, m)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	paginationMetadata, err := filter.CalculatePaginationMetadata(totalRecords, f.Page, f.PageSize)
	if err != nil {
		return nil, nil, err
	}

	return messages, paginationMetadata, nil
}

func (cmm *ConversationMessageModel) GetAllForGroup(conversationID uuid.UUID, f filter.Filters) ([]*ConversationMessageWithRepliedMessageAndSender, *filter.PaginationMetadata, error) {
	query := `
	SELECT
		count(*) OVER(),
		m.id, m.type, m.content, m.created_at, m.updated_at,
		u.id, u.username, u.email, u.bio, u.is_active,
		r.id, r.sender_id, r.type, r.content, r.created_at, r.updated_at
	FROM conversation_messages m
	JOIN users u ON u.id = m.sender_id
	LEFT JOIN conversation_messages r ON m.replied_message_id = r.id
	WHERE m.conversation_id = $1
	ORDER BY m.created_at DESC, m.id ASC
	LIMIT $2 OFFSET $3
	`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := cmm.DB.QueryContext(ctx, query, conversationID, f.Limit(), f.Offset())
	if err != nil {
		return nil, nil, err
	}

	defer rows.Close()

	messages := make([]*ConversationMessageWithRepliedMessageAndSender, 0)
	totalRecords := 0

	for rows.Next() {
		m := &ConversationMessageWithRepliedMessageAndSender{
			ConversationMessage: ConversationMessage{
				BaseModel: BaseModel{
					ID: conversationID,
				},
			},
		}

		var (
			repliedMessageID        *uuid.UUID
			repliedMessageSenderID  *uuid.UUID
			repliedMessageType      *string
			repliedMessageContent   *string
			repliedMessageCreatedAt *time.Time
			repliedMessageUpdatedAt *time.Time
		)

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
			&repliedMessageID,
			&repliedMessageSenderID,
			&repliedMessageType,
			&repliedMessageContent,
			&repliedMessageCreatedAt,
			&repliedMessageUpdatedAt,
		)

		if err != nil {
			return nil, nil, err
		}

		if repliedMessageID != nil {
			m.RepliedMessage = &ConversationMessage{
				BaseModel: BaseModel{
					ID: *repliedMessageID,
				},
				ConversationID: conversationID,
				SenderID:       *repliedMessageSenderID,
				Type:           *repliedMessageType,
				Content:        *repliedMessageContent,
				CreatedAt:      *repliedMessageCreatedAt,
				UpdatedAt:      *repliedMessageUpdatedAt,
			}
		}

		messages = append(messages, m)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	paginationMetadata, err := filter.CalculatePaginationMetadata(totalRecords, f.Page, f.PageSize)
	if err != nil {
		return nil, nil, err
	}

	return messages, paginationMetadata, nil
}

func (cmm *ConversationMessageModel) Insert(message *ConversationMessage) error {
	query := `
	INSERT INTO conversation_messages (conversation_id, sender_id, type, content, replied_message_id)
	VALUES ($1, $2, $3, $4, $5)
	RETURNING id, created_at, updated_at
	`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	args := []any{message.ConversationID, message.SenderID, message.Type, message.Content, message.RepliedMessageID}

	return cmm.DB.QueryRowContext(ctx, query, args...).Scan(&message.ID, &message.CreatedAt, &message.UpdatedAt)
}

func (cmm *ConversationMessageModel) BelongsToConversation(messageID, conversationID uuid.UUID, conversationType string) (bool, error) {
	query := `
	SELECT EXISTS(
		SELECT 1
		FROM conversation_messages m
		JOIN conversations c ON c.id = m.conversation_id
		WHERE
			m.id = $1
		AND
			m.conversation_id = $2
		AND
			c.type = $3
	)
	`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	args := []any{messageID, conversationID, conversationType}

	var exists bool

	err := cmm.DB.QueryRowContext(ctx, query, args...).Scan(&exists)

	if err != nil {
		return false, err
	}

	return exists, nil
}
