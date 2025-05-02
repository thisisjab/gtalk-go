package data

import (
	"github.com/google/uuid"
)

type ConversationMessage struct {
	BaseModel
	ConversationID uuid.UUID
	SenderID       uuid.UUID
	Content        string
	Type           string
	// TODO: add attachment
}
