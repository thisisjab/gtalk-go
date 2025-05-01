package api

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (s *APIServer) routes() http.Handler {
	router := httprouter.New()

	return s.logRequestMiddleware(s.panciRecoveryMiddleware(router))
}
