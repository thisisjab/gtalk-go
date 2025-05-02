package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/thisisjab/gchat-go/internal/data"
	"github.com/thisisjab/gchat-go/internal/validator"
)

func (s *APIServer) handlerPostUser(w http.ResponseWriter, r *http.Request) {
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

	_, err := s.models.Token.New(user.ID, 1*time.Hour, data.ScopeAccountActivation)
	if err != nil {
		s.serverErrorResponse(w, r, err)
		return
	}

	// TODO: send activation email

	if err := s.writeJSON(w, http.StatusCreated, envelope{"user": user}, nil); err != nil {
		s.serverErrorResponse(w, r, err)
		return
	}
}
