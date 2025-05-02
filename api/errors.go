package api

import "net/http"

func (s *APIServer) logError(r *http.Request, err error) {
	// TODO: add origin ip to log messages

	s.logger.Error(err.Error(),
		"request_method", r.Method,
		"request_url", r.URL.String(),
	)
}

func (s *APIServer) errorResponse(w http.ResponseWriter, r *http.Request, status int, msg any) {
	env := envelope{"error": msg}

	err := s.writeJSON(w, status, env, nil)

	if err != nil {
		s.logError(r, err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *APIServer) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	s.logError(r, err)

	message := "Internal server error"
	s.errorResponse(w, r, http.StatusInternalServerError, message)
}

func (s *APIServer) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	s.errorResponse(w, r, http.StatusBadRequest, err.Error())
}

func (s *APIServer) failedValidationResponse(w http.ResponseWriter, r *http.Request, errors map[string]string) {
	s.errorResponse(w, r, http.StatusUnprocessableEntity, errors)
}

func (s *APIServer) editConflictResponse(w http.ResponseWriter, r *http.Request) {
	message := "edit conflict: please try again."
	s.errorResponse(w, r, http.StatusConflict, message)
}

func (s *APIServer) rateLimitExceededResponse(w http.ResponseWriter, r *http.Request) {
	message := "too many requests"
	s.errorResponse(w, r, http.StatusTooManyRequests, message)
}

func (s *APIServer) invalidCredentialsResponse(w http.ResponseWriter, r *http.Request) {
	message := "invalid credentials provided"
	s.errorResponse(w, r, http.StatusUnauthorized, message)
}

func (s *APIServer) invalidAuthenticationTokenResponse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWW-Authenticate", "Bearer") // To indicate a bearer token is expected.

	message := "invalid or missing authentication token"
	s.errorResponse(w, r, http.StatusUnauthorized, message)
}

func (s *APIServer) authenticationRequiredResponse(w http.ResponseWriter, r *http.Request) {
	message := "authentication is required"
	s.errorResponse(w, r, http.StatusUnauthorized, message)
}

func (s *APIServer) inactiveAccountResponse(w http.ResponseWriter, r *http.Request) {
	message := "account is deactivated"
	s.errorResponse(w, r, http.StatusForbidden, message)
}
