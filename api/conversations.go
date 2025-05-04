package api

import "net/http"

func (s *APIServer) handleConversationsGET(w http.ResponseWriter, r *http.Request) {
	user := s.contextGetUser(r)

	conversations, err := s.models.Conversation.GetUserConversationsWithPreview(user.ID)
	if err != nil {
		s.serverErrorResponse(w, r, err)
		return
	}

	if err := s.writeJSON(w, http.StatusOK, envelope{"conversations": conversations}, nil); err != nil {
		s.serverErrorResponse(w, r, err)
		return
	}
}
