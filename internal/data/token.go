package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"time"

	"github.com/google/uuid"
	"github.com/thisisjab/gchat-go/internal/validator"
)

const (
	ScopeAccountActivation    = "account:activation"
	ScopeAuthenticationAccess = "auth:access"
)

type Token struct {
	Plaintext string    `json:"value"`
	Hash      []byte    `json:"-"`
	UserID    uuid.UUID `json:"user_id"`
	Expiry    time.Time `json:"expiry"`
	Scope     string    `json:"scope"`
	CreatedAt time.Time `json:"-"`
}

type TokenModel struct {
	DB DBOperator
}

func hashToken(tokenPlaintext string) []byte {
	hash := sha256.Sum256([]byte(tokenPlaintext))
	return hash[:]
}

func generateToken(userID uuid.UUID, ttl time.Duration, scope string) (*Token, error) {
	token := &Token{
		UserID: userID,
		Expiry: time.Now().Add(ttl),
		Scope:  scope,
	}

	randomBytes := make([]byte, 16)

	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}

	token.Plaintext = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)
	token.Hash = hashToken(token.Plaintext)

	return token, nil
}

func ValidateTokenPlaintext(v *validator.Validator, tokenPlaintext string) {
	v.Check(tokenPlaintext != "", "token", "must be provided")
	v.Check(len(tokenPlaintext) == 26, "token", "must be 26 bytes long")
}

func (m TokenModel) New(ctx context.Context, userID uuid.UUID, ttl time.Duration, scope string) (*Token, error) {
	token, err := generateToken(userID, ttl, scope)
	if err != nil {
		return nil, err
	}

	err = m.Insert(ctx, token)
	return token, err
}

func (m TokenModel) Insert(ctx context.Context, token *Token) error {
	query := `
	INSERT INTO tokens (user_id, hash, expiry, scope)
	VALUES ($1, $2, $3, $4)
	`

	args := []any{
		token.UserID,
		token.Hash,
		token.Expiry,
		token.Scope,
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, args...)

	return err
}

func (m TokenModel) DeleteAllForUser(ctx context.Context, userID uuid.UUID, scope string) error {
	query := `DELETE FROM tokens WHERE user_id=$1 AND scope = $2`

	args := []any{
		userID,
		scope,
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, args...)

	return err
}
