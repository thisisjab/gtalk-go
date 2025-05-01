package api

import (
	"net/http"
)

func (s *APIServer) handleGetHealthCheck(w http.ResponseWriter, r *http.Request) {
	err := s.writeJSON(w, http.StatusOK, envelope{"status": "available",
		"system_info": map[string]string{
			"environment": s.config.Environment,
			"version":     s.config.Version,
		},
	}, nil)

	if err != nil {
		s.serverErrorResponse(w, r, err)
	}
}
