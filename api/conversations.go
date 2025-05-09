package api

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
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

	conversations, paginationMetadata, err := s.models.Conversation.GetAllWithPreview(r.Context(), user.ID, f)
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
	otherUser, err := s.models.User.GetByID(r.Context(), *otherUserID)

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
	conversation, err := s.models.Conversation.GetPrivateBetweenUsers(r.Context(), user.ID, *otherUserID)
	if err != nil && !errors.Is(err, data.ErrNoRecordFound) {
		s.serverErrorResponse(w, r, err)
		return
	}

	messages, paginationMetadata, err := s.models.ConversationMessage.GetAllForPrivate(r.Context(), conversation.ID, f)

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
		Type             string     `json:"type"`
		Content          string     `json:"content"`
		RepliedMessageID *uuid.UUID `json:"replied_message_id"`
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
	if otherUserExists := s.models.User.ExistsByID(r.Context(), *otherUserID); !otherUserExists {
		s.notFoundResponse(w, r)

		return
	}

	// Get the conversation
	conversation, err := s.models.Conversation.GetPrivateBetweenUsers(r.Context(), user.ID, *otherUserID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrNoRecordFound):
			// If conversation doesn't exist, create it.
			conversation, err = s.models.Conversation.CreateBetweenUsers(r.Context(), user.ID, *otherUserID)

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
		ConversationID:   conversation.ID,
		SenderID:         user.ID,
		Content:          input.Content,
		Type:             input.Type,
		RepliedMessageID: input.RepliedMessageID,
	}

	data.ValidateConversationMessage(v, msg)
	if !v.Valid() {
		s.failedValidationResponse(w, r, v.Errors())
		return
	}

	// Check replied message exists
	if input.RepliedMessageID != nil {
		msgExists, err := s.models.ConversationMessage.BelongsToConversation(r.Context(), *input.RepliedMessageID, conversation.ID, data.ConversationTypePrivate)

		if err != nil {
			s.serverErrorResponse(w, r, err)

			return
		}

		v.Check(msgExists, "replied_message_id", "does not exist")
	}

	if !v.Valid() {
		s.failedValidationResponse(w, r, v.Errors())
		return
	}

	if err := s.models.ConversationMessage.Insert(r.Context(), msg); err != nil {
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

	participationExists, err := s.models.ConversationParticipant.Exists(r.Context(), user.ID, *groupID, data.ConversationTypeGroup)

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

	messages, paginationMetadata, err := s.models.ConversationMessage.GetAllForGroup(r.Context(), *groupID, f)
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

// handleCreateGroupMessage handles the POST /conversations/group/:group_id/messages endpoint.
// It creates a new message in a group chat if group exists and current user is a member of the group.
func (s *APIServer) handleCreateGroupMessage(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Type             string     `json:"type"`
		Content          string     `json:"content"`
		RepliedMessageID *uuid.UUID `json:"replied_message_id"`
	}

	if err := s.readJSON(w, r, &input); err != nil {
		s.badRequestResponse(w, r, err)
		return
	}

	user := s.contextGetUser(r)

	v := validator.New()

	groupID := s.readUUIDParam("group_id", r, v)
	if !v.Valid() {
		s.failedValidationResponse(w, r, v.Errors())
		return
	}

	// Check group exists
	groupExists, err := s.models.Conversation.Exists(r.Context(), *groupID, data.ConversationTypeGroup)
	if err != nil {
		s.serverErrorResponse(w, r, err)
		return
	}

	if !groupExists {
		s.notFoundResponse(w, r)
		return
	}

	// Check user is a member of group.
	isParticipant, err := s.models.ConversationParticipant.Exists(r.Context(), user.ID, *groupID, data.ConversationTypeGroup)
	if err != nil {
		s.serverErrorResponse(w, r, err)
		return
	}

	if !isParticipant {
		s.permissionDeniedResponse(w, r)
		return
	}

	// Prepare and validate message before inserting
	msg := &data.ConversationMessage{
		ConversationID:   *groupID,
		SenderID:         user.ID,
		Content:          input.Content,
		Type:             input.Type,
		RepliedMessageID: input.RepliedMessageID,
	}

	data.ValidateConversationMessage(v, msg)
	if !v.Valid() {
		s.failedValidationResponse(w, r, v.Errors())
		return
	}

	// Check replied message exists
	if input.RepliedMessageID != nil {
		msgExists, err := s.models.ConversationMessage.BelongsToConversation(r.Context(), *input.RepliedMessageID, *groupID, data.ConversationTypeGroup)

		if err != nil {
			s.serverErrorResponse(w, r, err)

			return
		}

		v.Check(msgExists, "replied_message_id", "does not exist")
	}

	if !v.Valid() {
		s.failedValidationResponse(w, r, v.Errors())
		return
	}

	if err := s.models.ConversationMessage.Insert(r.Context(), msg); err != nil {
		s.serverErrorResponse(w, r, err)
		return
	}

	if err := s.writeJSON(w, http.StatusCreated, envelope{"message": msg}, nil); err != nil {
		s.serverErrorResponse(w, r, err)
		return
	}
}

// handleCreateGroup handles the POST /conversations/group endpoint.
// It creates a group.
func (s *APIServer) handleCreateGroup(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name string `json:"name"`
	}

	if err := s.readJSON(w, r, &input); err != nil {
		s.badRequestResponse(w, r, err)

		return
	}

	groupMetadata := data.GroupMetadata{
		OwnerID: s.contextGetUser(r).ID,
		Name:    input.Name,
	}

	v := validator.New()

	data.ValidateGroupMetadata(v, groupMetadata)
	if !v.Valid() {
		s.failedValidationResponse(w, r, v.Errors())

		return
	}

	group := data.Conversation{
		Type:          data.ConversationTypeGroup,
		GroupMetadata: &groupMetadata,
	}

	err := s.models.Conversation.CreateGroup(r.Context(), &group)
	if err != nil {
		s.serverErrorResponse(w, r, err)

		return
	}

	if err := s.writeJSON(w, http.StatusCreated, envelope{"group": group}, nil); err != nil {
		s.serverErrorResponse(w, r, err)

		return
	}
}

func (s *APIServer) handleAddGroupParticipant(w http.ResponseWriter, r *http.Request) {
	user := s.contextGetUser(r)

	v := validator.New()

	groupID := s.readUUIDParam("group_id", r, v)

	if !v.Valid() {
		s.failedValidationResponse(w, r, v.Errors())

		return
	}

	group, err := s.models.Conversation.Get(r.Context(), *groupID, data.ConversationTypeGroup)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrNoRecordFound):
			s.notFoundResponse(w, r)
		default:
			s.serverErrorResponse(w, r, err)
		}

		return
	}

	// Check if current user is owner of group before proceeding
	if group.GroupMetadata.OwnerID != user.ID {
		s.permissionDeniedResponse(w, r)

		return
	}

	var input struct {
		ParticipantID uuid.UUID `json:"user_id"`
	}

	if err := s.readJSON(w, r, &input); err != nil {
		s.badRequestResponse(w, r, err)

		return
	}

	err = s.models.ConversationParticipant.AddParticipant(r.Context(), groupID, &input.ParticipantID)

	if err != nil {
		switch {
		case errors.Is(err, data.ErrUserDoesNotExist):
			v.AddError("user_id", "does not exist")
		case errors.Is(err, data.ErrConversationParticipantDuplicate):
			v.AddError("user_id", "already a participant")
		default:
			s.serverErrorResponse(w, r, err)
			return
		}
	}

	if !v.Valid() {
		s.failedValidationResponse(w, r, v.Errors())

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
