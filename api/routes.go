package api

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (s *APIServer) routes() http.Handler {
	router := NewRouter("/api/v1")

	// Healthcheck
	router.RegisterHandlerFunc(http.MethodGet, "/healthcheck", s.handleHealthCheckGET)

	// Authentication
	router.RegisterHandlerFunc(http.MethodPost, "/auth/token", s.handleCreateAccessTokenPOST)

	// Users
	router.RegisterHandlerFunc(http.MethodPost, "/users", s.handleUserPOST)
	router.RegisterHandlerFunc(http.MethodPost, "/users/account/activate", s.handleUserAccountActivatePOST)

	// Conversations
	router.RegisterHandlerFunc(http.MethodGet, "/conversations", s.requireActivatedUser(s.handleConversationsGET))

	// Middlewares
	router.RegisterMiddlewares(
		s.logRequestMiddleware,
		s.panciRecoveryMiddleware,
		s.corsMiddleware,
		s.rateLimitMiddleware,
		s.authenticate,
	)

	return router.All()
}

type Router struct {
	baseUrl     string
	middlewares []func(http.Handler) http.Handler
	router      *httprouter.Router
}

func NewRouter(baseUrl string) *Router {
	return &Router{
		baseUrl: baseUrl,
		router:  httprouter.New(),
	}
}

func (r *Router) RegisterMiddlewares(middlewares ...func(http.Handler) http.Handler) {
	r.middlewares = append(r.middlewares, middlewares...)
}

func (r *Router) RegisterHandlerFunc(method, path string, handler http.HandlerFunc) {
	r.router.HandlerFunc(method, r.baseUrl+path, handler)
}

func (r *Router) All() http.Handler {
	var res http.Handler = r.router

	for _, middleware := range r.middlewares {
		res = middleware(res)
	}

	return res
}
