package data

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/thisisjab/gchat-go/internal/filter"
	filters "github.com/thisisjab/gchat-go/internal/filter"
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
