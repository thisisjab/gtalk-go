package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/thisisjab/gchat-go/internal/data"
	"github.com/thisisjab/gchat-go/internal/validator"
)

func (s *APIServer) handleCreateAccessToken(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email             string `json:"email"`
		PasswordPlaintext string `json:"password"`
	}

	if err := s.readJSON(w, r, &input); err != nil {
		s.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()

	data.ValidateEmail(v, input.Email)
	data.ValidatePasswordPlaintext(v, input.PasswordPlaintext)

	if !v.Valid() {
		s.failedValidationResponse(w, r, v.Errors())
		return
	}

	user, err := s.models.User.GetByEmail(input.Email)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrNoRecordFound):
			s.invalidCredentialsResponse(w, r)
		default:
			s.serverErrorResponse(w, r, err)
		}
		return
	}

	match, err := user.Password.Matches(input.PasswordPlaintext)
	if err != nil {
		s.serverErrorResponse(w, r, err)
		return
	}

	if !match {
		s.invalidCredentialsResponse(w, r)
		return
	}

	token, err := s.models.Token.New(user.ID, 24*time.Hour, data.ScopeAuthenticationAccess)
	if err != nil {
		s.serverErrorResponse(w, r, err)
		return
	}

	err = s.writeJSON(w, http.StatusCreated, envelope{"token": token.Plaintext, "expiry": token.Expiry}, nil)
	if err != nil {
		s.serverErrorResponse(w, r, err)
		return
	}
}
