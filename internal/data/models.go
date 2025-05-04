package data

import "database/sql"

type Models struct {
	Conversation ConversationModel
	Token        TokenModel
	User         UserModel
}

func NewModels(db *sql.DB) *Models {
	return &Models{
		Conversation: ConversationModel{DB: db},
		Token:        TokenModel{DB: db},
		User:         UserModel{DB: db},
	}
}
