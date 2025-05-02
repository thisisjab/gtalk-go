package data

import "database/sql"

type Models struct {
	Token TokenModel
	User  UserModel
}

func NewModels(db *sql.DB) *Models {
	return &Models{
		Token: TokenModel{DB: db},
		User:  UserModel{DB: db},
	}
}
