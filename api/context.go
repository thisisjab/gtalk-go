package api

import (
	"context"
	"net/http"

	"github.com/thisisjab/gchat-go/internal/data"
)

type contextKey string

const userContextKey = contextKey("user")

// contextSetUser sets the user in the request context.
func (s *APIServer) contextSetUser(r *http.Request, user *data.User) *http.Request {
	ctx := context.WithValue(r.Context(), userContextKey, user)
	return r.WithContext(ctx)
}

// contextGetUser retrieves the user from the request context.
func (s *APIServer) contextGetUser(r *http.Request) *data.User {
	user, ok := r.Context().Value(userContextKey).(*data.User)

	if !ok {
		panic("missing user value in request context")
	}

	return user
}
