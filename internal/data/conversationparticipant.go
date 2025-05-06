package data

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type ConversationParticipant struct {
	ConversationID uuid.UUID
	UserID         uuid.UUID
	CreatedAt      time.Time
}

type ConversationParticipantModel struct {
	DB *sql.DB
}

func (cpm *ConversationParticipantModel) Exists(userID uuid.UUID, conversationID uuid.UUID, conversationType string) (bool, error) {
	query := `
	SELECT EXISTS(
		SELECT 1
		FROM conversation_participants cp
		JOIN conversations c ON cp.conversation_id = c.id
		WHERE
			cp.user_id = $1
		AND
			cp.conversation_id = $2
		AND
			c.type = $3
	)
	`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	args := []any{userID, conversationID, conversationType}

	var exists bool

	err := cpm.DB.QueryRowContext(ctx, query, args...).Scan(&exists)

	if err != nil {
		return false, err
	}

	return exists, nil
}
