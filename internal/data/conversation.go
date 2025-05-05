package data

import (
	"context"
	"database/sql"
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
	// Name is null if the conversation is private
	Name *string `json:"name,omitempty"`
	Type string  `json:"type"`
}

type ConversationModel struct {
	DB *sql.DB
}

type ConversationWithPreview struct {
	Conversation
	Preview ConversationMessage `json:"preview"`
}

func (cm *ConversationModel) GetUserConversationsWithPreview(userID uuid.UUID, f filters.Filters) ([]*ConversationWithPreview, *filters.PaginationMetadata, error) {
	query := `
	SELECT
		count(*) OVER() AS total_records,
		c.id, c.name, c.type, c.created_at,
		m.id, m.content, m.type, m.sender_id, m.created_at, m.updated_at
	FROM conversations c
	JOIN conversation_participants ON c.id = conversation_participants.conversation_id
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
		var c Conversation
		var p ConversationMessage
		if err := rows.Scan(
			&totalRecords,
			&c.ID,
			&c.Name,
			&c.Type,
			&c.CreatedAt,
			&p.ID,
			&p.Content,
			&p.Type,
			&p.SenderID,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, nil, err
		}
		conversations = append(conversations, &ConversationWithPreview{Conversation: c, Preview: p})
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
