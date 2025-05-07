package api

import (
	"errors"
	"net/http"

	"github.com/thisisjab/gchat-go/internal/data"
	"github.com/thisisjab/gchat-go/internal/filter"
	"github.com/thisisjab/gchat-go/internal/validator"
)

// handleListConversations handles the GET /conversations endpoint.
// It lists all conversations (group/private) for the authenticated user.
func (s *APIServer) handleListConversations(w http.ResponseWriter, r *http.Request) {
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

	filter.ValidateFilters(v, f)
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

// handleListPrivateConversationMessages handles the GET /conversations/private/:other_user_id/messages endpoint.
// It lists all messages in a private chat if other_user_id is a valid user id.
// Response includes `other_user` as well.
func (s *APIServer) handleListPrivateConversationMessages(w http.ResponseWriter, r *http.Request) {
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

	filter.ValidateFilters(v, f)
	if !v.Valid() {
		s.failedValidationResponse(w, r, v.Errors())
		return
	}

	otherUserID := s.readUUIDParam("other_user_id", r, v)
	if !v.Valid() {
		s.failedValidationResponse(w, r, v.Errors())
		return
	}

	// Check other user exists
	otherUser, err := s.models.User.GetByID(*otherUserID)

	if err != nil {
		switch {
		case errors.Is(err, data.ErrNoRecordFound):
			s.notFoundResponse(w, r)
		default:
			s.serverErrorResponse(w, r, err)
		}
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

	if err := s.writeJSON(w, http.StatusOK, envelope{"messages": messages, "other_user": otherUser, "pagination": paginationMetadata}, nil); err != nil {
		s.serverErrorResponse(w, r, err)
		return
	}
}

// handleCreatePrivateMessage handles the POST /conversations/private/:other_user_id/messages endpoint.
// It creates a new message in a private chat if `other_user_id` is a valid user id.
func (s *APIServer) handleCreatePrivateMessage(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Type    string `json:"type"`
		Content string `json:"content"`
	}

	if err := s.readJSON(w, r, &input); err != nil {
		s.badRequestResponse(w, r, err)
		return
	}

	user := s.contextGetUser(r)

	v := validator.New()

	otherUserID := s.readUUIDParam("other_user_id", r, v)
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
	conversation, err := s.models.Conversation.GetPrivateBetweenUsers(user.ID, *otherUserID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrNoRecordFound):
			// If conversation doesn't exist, create it.
			conversation, err = s.models.Conversation.CreateBetweenUsers(user.ID, *otherUserID)

			if err != nil {
				s.serverErrorResponse(w, r, err)
				return
			}
		default:
			s.serverErrorResponse(w, r, err)
			return
		}
	}

	// Prepare and validate message before inserting
	msg := &data.ConversationMessage{
		ConversationID: conversation.ID,
		SenderID:       user.ID,
		Content:        input.Content,
		Type:           input.Type,
	}

	data.ValidateConversationMessage(v, msg)
	if !v.Valid() {
		s.failedValidationResponse(w, r, v.Errors())
		return
	}

	if err := s.models.ConversationMessage.Insert(msg); err != nil {
		s.serverErrorResponse(w, r, err)
		return
	}

	if err := s.writeJSON(w, http.StatusCreated, envelope{"message": msg}, nil); err != nil {
		s.serverErrorResponse(w, r, err)
		return
	}
}

// handleListGroupMessages handles the GET /conversations/groups/:group_id/messages endpoint.
// It retrieves a list of messages in a group conversation including the group information.
// If user is not a member of the group, a 404 is raised.
func (s *APIServer) handleListGroupMessages(w http.ResponseWriter, r *http.Request) {
	user := s.contextGetUser(r)

	v := validator.New()

	groupID := s.readUUIDParam("group_id", r, v)
	if !v.Valid() {
		s.failedValidationResponse(w, r, v.Errors())
		return
	}

	participationExists, err := s.models.ConversationParticipant.Exists(user.ID, *groupID, data.ConversationTypeGroup)

	if err != nil {
		s.serverErrorResponse(w, r, err)
		return
	}

	if !participationExists {
		s.notFoundResponse(w, r)
		return
	}

	f := filter.Filters{
		Page:         s.readIntQuery(r.URL.Query(), "page", 1, v),
		PageSize:     s.readIntQuery(r.URL.Query(), "page_size", 10, v),
		Sort:         "id",
		SortSafeList: []string{"id"},
	}

	filter.ValidateFilters(v, f)

	if !v.Valid() {
		s.failedValidationResponse(w, r, v.Errors())
		return
	}

	messages, paginationMetadata, err := s.models.ConversationMessage.GetAllForGroup(*groupID, f)
	if err != nil {
		s.serverErrorResponse(w, r, err)
		return
	}

	// TODO: include `group` in response.
	if err := s.writeJSON(w, http.StatusOK, envelope{"messages": messages, "pagination": paginationMetadata}, nil); err != nil {
		s.serverErrorResponse(w, r, err)
		return
	}
}
