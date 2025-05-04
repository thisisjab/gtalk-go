package api

import (
	"net/http"

	"github.com/thisisjab/gchat-go/internal/filter"
	"github.com/thisisjab/gchat-go/internal/validator"
)

func (s *APIServer) handleConversationsGET(w http.ResponseWriter, r *http.Request) {
	user := s.contextGetUser(r)

	v := validator.New()
	f := filter.Filters{
		Page:     s.readInt(r.URL.Query(), "page", 1, v),
		PageSize: s.readInt(r.URL.Query(), "page_size", 10, v),
	}

	if !v.Valid() {
		s.failedValidationResponse(w, r, v.Errors())
		return
	}

	conversations, paginationMetadata, err := s.models.Conversation.GetUserConversationsWithPreview(user.ID, f)
	if err != nil {
		s.serverErrorResponse(w, r, err)
		return
	}

	if err := s.writeJSON(w, http.StatusOK, envelope{"conversations": conversations, "pagination": paginationMetadata}, nil); err != nil {
		s.serverErrorResponse(w, r, err)
		return
	}
}
