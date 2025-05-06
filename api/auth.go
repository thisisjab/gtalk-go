package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/thisisjab/gchat-go/internal/data"
	"github.com/thisisjab/gchat-go/internal/validator"
)

// authenticate middleware checks the authorization header and puts a user (either anonymous or authenticated) in the context.
func (s *APIServer) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Authorization")

		authorizationHeader := r.Header.Get("Authorization")

		if authorizationHeader == "" {
			r = s.contextSetUser(r, data.AnonymousUser)
			next.ServeHTTP(w, r)
			return
		}

		headerParts := strings.Split(authorizationHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			s.invalidAuthenticationTokenResponse(w, r)
			return
		}

		token := headerParts[1]

		v := validator.New()

		if data.ValidateTokenPlaintext(v, token); !v.Valid() {
			s.invalidAuthenticationTokenResponse(w, r)
			return
		}

		user, err := s.models.User.GetFromToken(token, data.ScopeAuthenticationAccess)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrNoRecordFound):
				s.invalidAuthenticationTokenResponse(w, r)
			default:
				s.serverErrorResponse(w, r, err)
			}

			return
		}

		r = s.contextSetUser(r, user)

		next.ServeHTTP(w, r)
	})
}

// requireAuthenticatedUser middleware ensures the user is authenticated.
func (s *APIServer) requireAuthenticatedUser(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := s.contextGetUser(r)

		if user.IsAnonymous() {
			s.authenticationRequiredResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// requireActivatedUser middleware ensures the user is authenticated and activated.
func (s *APIServer) requireActivatedUser(next http.HandlerFunc) http.HandlerFunc {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := s.contextGetUser(r)

		if !user.IsActive {
			s.inactiveAccountResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})

	return s.requireAuthenticatedUser(fn)
}
