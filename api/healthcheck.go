package api

import (
	"fmt"
	"net/http"
)

func (s *APIServer) handleGetHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "OK")
}
