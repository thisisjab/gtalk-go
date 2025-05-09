package data

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/thisisjab/gchat-go/internal/validator"
	"golang.org/x/crypto/bcrypt"
)

type UserModel struct {
	DB DBOperator
}

type User struct {
	BaseModel
	Username        string     `json:"username"`
	Email           string     `json:"email"`
	EmailVerifiedAt *time.Time `json:"-"`
	Bio             *string    `json:"bio"`
	IsActive        bool       `json:"is_active"`
	Password        password   `json:"-"`
}

var AnonymousUser = &User{}

func (u *User) IsAnonymous() bool {
	return u == AnonymousUser
}

type password struct {
	plaintext *string
	hash      []byte
}

func (p *password) Set(plaintextPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintextPassword), 12)
	if err != nil {
		return err
	}

	p.plaintext = &plaintextPassword
	p.hash = hash

	return nil
}

func (p *password) Matches(plaintextPassword string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(p.hash, []byte(plaintextPassword))

	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err
		}
	}

	return true, nil
}

var (
	ErrUserDoesNotExist      = errors.New("non-existing user")
	ErrUserDuplicateEmail    = errors.New("duplicate email")
	ErrUserDuplicateUsername = errors.New("duplicate username")
)

func ValidateEmail(v *validator.Validator, email string) {
	v.Check(email != "", "email", "must be provided")
	v.Check(validator.Matches(email, validator.EmailRX), "email", "must be a valid email address")
}

func ValidatePasswordPlaintext(v *validator.Validator, password string) {
	v.Check(password != "", "password", "must be provided")
	v.Check(len(password) >= 8, "password", "must be at least 8 bytes long")
	v.Check(len(password) <= 72, "password", "must be at most 72 bytes long")
}

func ValidateUser(v *validator.Validator, user *User) {
	v.Check(user.Username != "", "username", "must be provided")
	v.Check(len(user.Username) >= 5, "username", "must be at least 5 bytes long")
	v.Check(len(user.Username) <= 500, "username", "must be at most 500 bytes long")

	v.Check(user.Bio == nil || len(*user.Bio) <= 1000, "bio", "must be at most 1000 bytes long")

	ValidateEmail(v, user.Email)

	if user.Password.plaintext != nil {
		ValidatePasswordPlaintext(v, *user.Password.plaintext)
	}

	if user.Password.hash == nil {
		panic("missing password hash for user")
	}
}

func (m UserModel) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	query := `
		SELECT id, username, email, bio, password_hash, is_active, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	user := &User{}

	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Bio,
		&user.Password.hash,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrNoRecordFound
		default:
			return nil, err
		}
	}

	return user, nil
}

func (m UserModel) GetByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, email, password_hash, is_active
		FROM users
		WHERE email = $1
	`

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	user := &User{}

	err := m.DB.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Password.hash,
		&user.IsActive,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrNoRecordFound
		default:
			return nil, err
		}
	}

	return user, nil
}

func (m UserModel) Insert(ctx context.Context, user *User) error {
	query := `
		INSERT INTO users (username, email, password_hash, is_active, bio)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at, version
	`

	args := []any{user.Username, user.Email, user.Password.hash, user.IsActive, user.Bio}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt, &user.Version)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_username_key"`:
			return ErrUserDuplicateUsername
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrUserDuplicateEmail
		default:
			return err
		}
	}

	return nil
}

func (m UserModel) Update(ctx context.Context, user *User) error {
	query := `
	UPDATE users
	SET
		username = $1,
		email = $2,
		email_verified_at = $3,
		password_hash = $4,
		is_active = $5,
		bio = $6,
		version = version + 1
	WHERE id = $7 AND version = $8
	RETURNING version
	`

	args := []any{
		user.Username,
		user.Email,
		user.EmailVerifiedAt,
		user.Password.hash,
		user.IsActive,
		user.Bio,
		user.ID,
		user.Version,
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&user.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		case err.Error() == `pq: duplicate key value violates unique constraint "users_username_key"`:
			return ErrUserDuplicateUsername
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrUserDuplicateEmail
		default:
			return err
		}
	}

	return nil
}

func (m UserModel) GetFromToken(ctx context.Context, tokenPlaintext, scope string) (*User, error) {
	tokenHash := hashToken(tokenPlaintext)

	query := `
		SELECT
			u.id,
			u.username,
			u.email,
			u.email_verified_at,
			u.password_hash,
			u.bio,
			u.is_active,
			u.created_at,
			u.updated_at,
			u.version
		FROM users u
		INNER JOIN tokens t ON u.id = t.user_id
		WHERE t.hash = $1 AND t.scope = $2 AND t.expiry > $3
	`

	var user User

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	args := []any{tokenHash, scope, time.Now()}

	err := m.DB.QueryRowContext(ctx, query, args...).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.EmailVerifiedAt,
		&user.Password.hash,
		&user.Bio,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.Version,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrNoRecordFound
		default:
			return nil, err
		}
	}

	return &user, nil
}

func (m *UserModel) ExistsByID(ctx context.Context, id uuid.UUID) bool {
	query := `SELECT id FROM users WHERE id = $1`

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var foundID uuid.UUID

	err := m.DB.QueryRowContext(ctx, query, id).Scan(&foundID)

	return err == nil
}
