package api

import (
	"fmt"
	"net/http"
)

func (s *APIServer) panciRecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					w.Header().Set("Connection", "close")

					s.serverErrorResponse(w, r, fmt.Errorf("%s", err))
				}
			}()

			next.ServeHTTP(w, r)
		},
	)
}

func (s *APIServer) logRequestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.logger.Info("incoming request",
			"method", r.Method,
			"url", r.URL.String(),
		)

		next.ServeHTTP(w, r)
	})
}
