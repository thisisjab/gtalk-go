package data

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type BaseModel struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
	Version   int64     `json:"-"`
}

var (
	ErrNoRecordFound = errors.New("no record found")
	ErrEditConflict  = errors.New("edit conflict")
)
