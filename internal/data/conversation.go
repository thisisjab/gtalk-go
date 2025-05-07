package data

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	filters "github.com/thisisjab/gchat-go/internal/filter"
)

const (
	ConversationTypePrivate = "private"
	ConversationTypeGroup   = "group"
)

type Conversation struct {
	BaseModel
	Type string `json:"type"`

	GroupMetadata *GroupMetadata `json:"group_metadata,omitempty"`
}

type GroupMetadata struct {
	OwnerID uuid.UUID `json:"owner_id"`
	Name    string    `json:"name"`
}

type ConversationModel struct {
	DB *sql.DB
}

type ConversationWithPreview struct {
	Conversation
	Preview *ConversationMessage `json:"preview"`
}

func (cm *ConversationModel) GetAllWithPreview(userID uuid.UUID, f filters.Filters) ([]*ConversationWithPreview, *filters.PaginationMetadata, error) {
	query := `
	SELECT
		count(*) OVER() AS total_records,
		c.id, c.type, c.created_at,
		gm.name, gm.owner_id,
		m.id, m.content, m.type, m.sender_id, m.created_at, m.updated_at
	FROM conversations c
	JOIN conversation_participants ON c.id = conversation_participants.conversation_id
	LEFT JOIN group_metadata gm ON gm.conversation_id = c.id
	LEFT JOIN LATERAL (
		SELECT *
		FROM conversation_messages
		WHERE conversation_messages.conversation_id = c.id
		ORDER BY conversation_messages.created_at DESC
		LIMIT 1
	) m ON true
	WHERE conversation_participants.user_id = $1
	LIMIT $2 OFFSET $3
	`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	args := []any{userID, f.Limit(), f.Offset()}

	rows, err := cm.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	totalRecords := 0
	conversations := make([]*ConversationWithPreview, 0)

	for rows.Next() {
		var (
			c Conversation

			// Group metadata
			groupOwnerID *uuid.UUID
			groupName    *string

			// Preview message
			previewMessageID        *uuid.UUID
			previewMessageContent   *string
			previewMessageType      *string
			previewMessageSenderID  *uuid.UUID
			previewMessageCreatedAt *time.Time
			previewMessageUpdatedAt *time.Time
		)

		if err := rows.Scan(
			&totalRecords,
			// Conversation
			&c.ID,
			&c.Type,
			&c.CreatedAt,
			// Group metadata
			&groupName,
			&groupOwnerID,
			// Preview message
			&previewMessageID,
			&previewMessageContent,
			&previewMessageType,
			&previewMessageSenderID,
			&previewMessageCreatedAt,
			&previewMessageUpdatedAt,
		); err != nil {
			return nil, nil, err
		}

		item := ConversationWithPreview{Conversation: c}

		if groupOwnerID != nil {
			item.GroupMetadata = &GroupMetadata{
				OwnerID: *groupOwnerID,
				Name:    *groupName,
			}
		}

		if previewMessageID != nil {
			item.Preview = &ConversationMessage{
				BaseModel: BaseModel{
					ID:        *previewMessageID,
					CreatedAt: *previewMessageCreatedAt,
					UpdatedAt: *previewMessageUpdatedAt,
				},
				Content:  *previewMessageContent,
				Type:     *previewMessageType,
				SenderID: *previewMessageSenderID,
			}
		}

		conversations = append(conversations, &item)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	paginationMetadata, err := filters.CalculatePaginationMetadata(totalRecords, f.Page, f.PageSize)
	if err != nil {
		return nil, nil, err
	}

	return conversations, paginationMetadata, nil
}

func (cm *ConversationModel) GetPrivateBetweenUsers(userID, otherUserID uuid.UUID) (*Conversation, error) {
	query := `
	SELECT c.id, c.created_at FROM conversations c
    JOIN conversation_participants cp1
        ON c.id = cp1.conversation_id
    JOIN conversation_participants cp2
        ON c .id = cp2.conversation_id
    JOIN users u1 ON u1.id = cp1.user_id
    JOIN users u2 ON u2.id = cp2.user_id
    WHERE cp1.user_id = $1
        AND cp2.user_id = $2
	`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conversation := &Conversation{}

	err := cm.DB.QueryRowContext(ctx, query, userID, otherUserID).Scan(&conversation.ID, &conversation.CreatedAt)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrNoRecordFound
		default:
			return nil, err
		}
	}

	return conversation, nil
}

func (cm *ConversationModel) CreateBetweenUsers(userID, otherUserID uuid.UUID) (*Conversation, error) {
	query := `
	INSERT INTO conversations (type)
	VALUES ('private')
	RETURNING id, type, created_at, updated_at
	`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := cm.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Create conversation
	conversation := &Conversation{}
	err = tx.QueryRowContext(ctx, query).Scan(&conversation.ID, &conversation.Type, &conversation.CreatedAt, &conversation.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// Add participants
	query = `INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2), ($1, $3)`
	_, err = tx.ExecContext(ctx, query, conversation.ID, userID, otherUserID)
	if err != nil {
		return nil, err
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return conversation, nil
}

func (cm *ConversationModel) Exists(conversationID uuid.UUID, conversationType string) (bool, error) {
	query := `
	SELECT EXISTS(
		SELECT 1
		FROM conversations c
		WHERE
			c.id = $1
		AND
			c.type = $2
	)
	`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	args := []any{conversationID, conversationType}

	var exists bool

	err := cm.DB.QueryRowContext(ctx, query, args...).Scan(&exists)

	if err != nil {
		return false, err
	}

	return exists, nil
}
