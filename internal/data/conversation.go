package data

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
