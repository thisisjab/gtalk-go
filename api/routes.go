package api

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (s *APIServer) routes() http.Handler {
	router := NewRouter("/api/v1")
	router.router.MethodNotAllowed = http.HandlerFunc(s.methodNotAllowedResponse)
	router.router.NotFound = http.HandlerFunc(s.notFoundResponse)

	// HealthCheck
	router.RegisterHandlerFunc(http.MethodGet, "/healthcheck", s.handleHealthCheck)

	// Authentication
	router.RegisterHandlerFunc(http.MethodPost, "/auth/token", s.handleCreateAccessToken)

	// Users
	router.RegisterHandlerFunc(http.MethodPost, "/users", s.handleCreateUser)
	router.RegisterHandlerFunc(http.MethodPost, "/users/account/activate", s.handleActivateUserAccount)

	// Conversations
	router.RegisterHandlerFunc(http.MethodGet, "/conversations", s.requireActivatedUser(s.handleListConversations))
	router.RegisterHandlerFunc(http.MethodPost, "/conversations/group", s.requireActivatedUser(s.handleCreateGroup))
	router.RegisterHandlerFunc(http.MethodPost, "/conversations/group/:group_id/participants", s.requireActivatedUser(s.handleAddGroupParticipant))

	// Conversation Messages
	router.RegisterHandlerFunc(http.MethodGet, "/conversations/private/:other_user_id/messages", s.requireActivatedUser(s.handleListPrivateConversationMessages))
	router.RegisterHandlerFunc(http.MethodPost, "/conversations/private/:other_user_id/messages", s.requireActivatedUser(s.handleCreatePrivateMessage))
	router.RegisterHandlerFunc(http.MethodGet, "/conversations/group/:group_id/messages", s.requireActivatedUser(s.handleListGroupMessages))
	router.RegisterHandlerFunc(http.MethodPost, "/conversations/group/:group_id/messages", s.requireActivatedUser(s.handleCreateGroupMessage))

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
