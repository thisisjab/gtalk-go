package data

import (
	"time"

	"github.com/google/uuid"
)

type ConversationMessage struct {
	BaseModel
	ConversationID uuid.UUID `json:"-"`
	SenderID       uuid.UUID `json:"sender_id,omitempty"`
	Content        string    `json:"content"`
	Type           string    `json:"type"`
	CreatedAt      time.Time `json:"created_at"`
	// TODO: add attachment
}
