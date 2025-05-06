package data

import "database/sql"

type Models struct {
	Conversation            ConversationModel
	ConversationMessage     ConversationMessageModel
	ConversationParticipant ConversationParticipantModel
	Token                   TokenModel
	User                    UserModel
}

func NewModels(db *sql.DB) *Models {
	return &Models{
		Conversation:            ConversationModel{DB: db},
		ConversationMessage:     ConversationMessageModel{DB: db},
		ConversationParticipant: ConversationParticipantModel{DB: db},
		Token:                   TokenModel{DB: db},
		User:                    UserModel{DB: db},
	}
}
