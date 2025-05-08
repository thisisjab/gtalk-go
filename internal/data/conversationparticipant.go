package data

import (
	"context"
	"database/sql"
	"errors"
	"strings"
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

var (
	ErrConversationParticipantDuplicate = errors.New("duplicate participant")
)

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

func (cp *ConversationParticipantModel) AddParticipant(conversationID, userID *uuid.UUID) error {
	query := `
		INSERT INTO conversation_participants(conversation_id, user_id) VALUES ($1, $2)
	`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := cp.DB.ExecContext(ctx, query, conversationID, userID)
	if err != nil {
		errorText := err.Error()

		switch {
		case strings.Contains(errorText, `pq: duplicate key value violates unique constraint "unique_participant"`):
			return ErrConversationParticipantDuplicate
		case strings.Contains(errorText, `pq: insert or update on table "conversation_participants" violates foreign key constraint "conversation_participants_user_id_fkey"`):
			return ErrUserDoesNotExist
		case strings.Contains(errorText, `pq: insert or update on table "conversation_participants" violates foreign key constraint "conversation_participants_conversation_id_fkey"`):
			return ErrConversationDoesNotExist
		default:
			return err
		}
	}

	return nil
}
