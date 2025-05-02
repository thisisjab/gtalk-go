package api

import (
	"context"
	"net/http"

	"github.com/thisisjab/gchat-go/internal/data"
)

type contextKey string

const userContextKey = contextKey("user")

func (s *APIServer) contextSetUser(r *http.Request, user *data.User) *http.Request {
	ctx := context.WithValue(r.Context(), userContextKey, user)
	return r.WithContext(ctx)
}

func (s *APIServer) contextGetUser(r *http.Request) *data.User {
	user, ok := r.Context().Value(userContextKey).(*data.User)

	if !ok {
		panic("missing user value in request context")
	}

	return user
}
