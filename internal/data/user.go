package data

import "time"

type User struct {
	BaseModel
	Email           string
	EmailVerifiedAt *time.Time
	DisplayName     string
	Bio             *string
}
