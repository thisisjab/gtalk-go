package api

import "net/http"

func (s *APIServer) logError(r *http.Request, err error) {
	// TODO: add origin ip to log messages

	s.logger.Error(err.Error(),
		"request_method", r.Method,
		"request_url", r.URL.String(),
	)
}

func (s *APIServer) errorResponse(w http.ResponseWriter, r *http.Request, status int, msg string) {
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
