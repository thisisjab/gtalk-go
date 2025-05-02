package data

import (
	"time"

	"github.com/google/uuid"
)

type ConversationParticipant struct {
	ConversationID uuid.UUID
	UserID         uuid.UUID
	CreatedAt      time.Time
}
