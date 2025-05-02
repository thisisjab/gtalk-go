package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/thisisjab/gchat-go/internal/data"
	"github.com/thisisjab/gchat-go/internal/validator"
)

func (s *APIServer) handleUserPOST(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string  `json:"username"`
		Email    string  `json:"email"`
		Bio      *string `json:"bio"`
		Password string  `json:"password"`
	}

	if err := s.readJSON(w, r, &input); err != nil {
		s.badRequestResponse(w, r, err)
		return
	}

	user := &data.User{
		Username: input.Username,
		Email:    input.Email,
		Bio:      input.Bio,
		IsActive: false,
	}
	user.Password.Set(input.Password)

	v := validator.New()
	data.ValidateUser(v, user)

	if !v.Valid() {
		s.failedValidationResponse(w, r, v.Errors())
		return
	}

	// TODO: wrap in transaction
	if err := s.models.User.Insert(user); err != nil {
		switch {
		case errors.Is(err, data.ErrUserDuplicateUsername):
			v.AddError("username", "this username is already taken")
			s.failedValidationResponse(w, r, v.Errors())
		case errors.Is(err, data.ErrUserDuplicateEmail):
			v.AddError("email", "this email is already taken")
			s.failedValidationResponse(w, r, v.Errors())
		default:
			s.serverErrorResponse(w, r, err)
		}
		return
	}

	token, err := s.models.Token.New(user.ID, 1*time.Hour, data.ScopeAccountActivation)
	if err != nil {
		s.serverErrorResponse(w, r, err)
		return
	}

	s.background(func() {
		data := map[string]any{
			"activationToken": token.Plaintext,
			"userID":          user.ID,
		}

		err := s.mailer.Send(user.Email, "user_account_activation.tmpl", data)

		if err != nil {
			s.logger.Error("error sending welcome email to user", "user_id", user.ID, "error", err)
		}
	})

	if err := s.writeJSON(w, http.StatusCreated, envelope{"user": user}, nil); err != nil {
		s.serverErrorResponse(w, r, err)
		return
	}
}

func (s *APIServer) handleUserAccountActivatePOST(w http.ResponseWriter, r *http.Request) {
	var input struct {
		TokenPlaintext string `json:"token"`
	}

	if err := s.readJSON(w, r, &input); err != nil {
		s.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	data.ValidateTokenPlaintext(v, input.TokenPlaintext)

	if !v.Valid() {
		s.failedValidationResponse(w, r, v.Errors())
		return
	}

	user, err := s.models.User.GetFromToken(input.TokenPlaintext, data.ScopeAccountActivation)

	if err != nil {
		switch {
		case errors.Is(err, data.ErrNoRecordFound):
			s.badRequestResponse(w, r, errors.New("invalid token"))
		default:
			s.serverErrorResponse(w, r, err)
		}
		return
	}

	now := time.Now()
	user.EmailVerifiedAt = &now
	user.IsActive = true

	if err := s.models.User.Update(user); err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			s.editConflictResponse(w, r)
		default:
			s.serverErrorResponse(w, r, err)
		}
		return
	}

	err = s.models.Token.DeleteAllForUser(user.ID, data.ScopeAccountActivation)
	if err != nil {
		s.serverErrorResponse(w, r, err)
		return
	}

	err = s.writeJSON(w, http.StatusOK, envelope{"user": user}, nil)
	if err != nil {
		s.serverErrorResponse(w, r, err)
	}
}
