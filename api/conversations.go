package api

import (
	"errors"
	"net/http"

	"github.com/thisisjab/gchat-go/internal/data"
	"github.com/thisisjab/gchat-go/internal/filter"
	"github.com/thisisjab/gchat-go/internal/validator"
)

func (s *APIServer) handleConversationsGET(w http.ResponseWriter, r *http.Request) {
	user := s.contextGetUser(r)

	v := validator.New()
	f := filter.Filters{
		Page:     s.readIntQuery(r.URL.Query(), "page", 1, v),
		PageSize: s.readIntQuery(r.URL.Query(), "page_size", 10, v),
	}

	if !v.Valid() {
		s.failedValidationResponse(w, r, v.Errors())
		return
	}

	conversations, paginationMetadata, err := s.models.Conversation.GetAllWithPreview(user.ID, f)
	if err != nil {
		switch {
		case errors.Is(err, filter.InvalidPageError):
			s.badRequestResponse(w, r, err)
		default:
			s.serverErrorResponse(w, r, err)
		}
		return
	}

	if err := s.writeJSON(w, http.StatusOK, envelope{"conversations": conversations, "pagination": paginationMetadata}, nil); err != nil {
		s.serverErrorResponse(w, r, err)
		return
	}
}

func (s *APIServer) handlePrivateConversationMessagesGET(w http.ResponseWriter, r *http.Request) {
	user := s.contextGetUser(r)

	v := validator.New()
	f := filter.Filters{
		Page:     s.readIntQuery(r.URL.Query(), "page", 1, v),
		PageSize: s.readIntQuery(r.URL.Query(), "page_size", 10, v),
	}

	otherUserID, err := s.readUUIDParam("other_user_id", r)
	if err != nil {
		v.AddError("other_user_id", err.Error())
	}

	if !v.Valid() {
		s.failedValidationResponse(w, r, v.Errors())
		return
	}

	// Check other user exists
	if otherUserExists := s.models.User.ExistsByID(*otherUserID); !otherUserExists {
		s.notFoundResponse(w, r)

		return
	}

	// Get the conversation
	// If conversation doesn't exist, client just sees an empty list of messages
	conversation, err := s.models.Conversation.GetPrivateBetweenUsers(user.ID, *otherUserID)
	if err != nil && !errors.Is(err, data.ErrNoRecordFound) {
		s.serverErrorResponse(w, r, err)
		return
	}

	messages, paginationMetadata, err := s.models.ConversationMessage.GetAllForPrivate(conversation.ID, f)

	if err != nil {
		switch {
		case errors.Is(err, filter.InvalidPageError):
			s.badRequestResponse(w, r, err)
		default:
			s.serverErrorResponse(w, r, err)
		}
		return
	}

	if err := s.writeJSON(w, http.StatusOK, envelope{"messages": messages, "pagination": paginationMetadata}, nil); err != nil {
		s.serverErrorResponse(w, r, err)
		return
	}
}
