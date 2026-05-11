package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	api "github.com/kagent-dev/kagent/go/api/httpapi"
	"github.com/kagent-dev/kagent/go/core/internal/httpserver/errors"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

// CommentsHandler handles agent comment operations.
type CommentsHandler struct {
	*Base
}

// NewCommentsHandler creates a new comments handler.
func NewCommentsHandler(base *Base) *CommentsHandler {
	return &CommentsHandler{Base: base}
}

// HandleListComments handles GET /api/agents/{namespace}/{name}/comments
func (h *CommentsHandler) HandleListComments(w ErrorResponseWriter, r *http.Request) {
	log := ctrllog.FromContext(r.Context()).WithName("comments-handler").WithValues("operation", "list-comments")

	namespace, err := GetPathParam(r, "namespace")
	if err != nil {
		w.RespondWithError(errors.NewBadRequestError("Missing namespace", err))
		return
	}
	name, err := GetPathParam(r, "name")
	if err != nil {
		w.RespondWithError(errors.NewBadRequestError("Missing name", err))
		return
	}
	agentID := namespace + "/" + name

	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		parsed, err := strconv.Atoi(limitStr)
		if err != nil || parsed < 1 {
			w.RespondWithError(errors.NewBadRequestError("Invalid limit parameter", fmt.Errorf("limit must be a positive integer")))
			return
		}
		if parsed > 100 {
			parsed = 100
		}
		limit = parsed
	}

	comments, err := h.DatabaseService.ListAgentComments(r.Context(), agentID, limit)
	if err != nil {
		log.Error(err, "Failed to list agent comments")
		w.RespondWithError(errors.NewInternalServerError("Failed to list comments", err))
		return
	}

	resp := make([]api.CommentResponse, len(comments))
	for i, c := range comments {
		resp[i] = api.CommentResponse{
			ID:        c.ID,
			AgentID:   c.AgentID,
			UserID:    c.UserID,
			Content:   c.Content,
			CreatedAt: c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	log.Info("Comments listed successfully", "agentID", agentID, "count", len(resp))
	RespondWithJSON(w, http.StatusOK, api.NewResponse(resp, "Successfully listed comments", false))
}

// HandleCreateComment handles POST /api/agents/{namespace}/{name}/comments
func (h *CommentsHandler) HandleCreateComment(w ErrorResponseWriter, r *http.Request) {
	log := ctrllog.FromContext(r.Context()).WithName("comments-handler").WithValues("operation", "create-comment")

	namespace, err := GetPathParam(r, "namespace")
	if err != nil {
		w.RespondWithError(errors.NewBadRequestError("Missing namespace", err))
		return
	}
	name, err := GetPathParam(r, "name")
	if err != nil {
		w.RespondWithError(errors.NewBadRequestError("Missing name", err))
		return
	}
	agentID := namespace + "/" + name

	userID, err := GetUserID(r)
	if err != nil {
		log.Error(err, "Failed to get user ID")
		w.RespondWithError(errors.NewBadRequestError("Failed to get user ID", err))
		return
	}

	var req api.CreateCommentRequest
	if err := DecodeJSONBody(r, &req); err != nil {
		w.RespondWithError(errors.NewBadRequestError("Invalid request body", err))
		return
	}

	if req.Content == "" {
		w.RespondWithError(errors.NewBadRequestError("Content is required", nil))
		return
	}
	if len(req.Content) > 500 {
		w.RespondWithError(errors.NewBadRequestError("Content must not exceed 500 characters", nil))
		return
	}

	comment, err := h.DatabaseService.CreateAgentComment(r.Context(), agentID, userID, req.Content)
	if err != nil {
		log.Error(err, "Failed to create agent comment")
		w.RespondWithError(errors.NewInternalServerError("Failed to create comment", err))
		return
	}

	resp := api.CommentResponse{
		ID:        comment.ID,
		AgentID:   comment.AgentID,
		UserID:    comment.UserID,
		Content:   comment.Content,
		CreatedAt: comment.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	log.Info("Comment created successfully", "agentID", agentID, "commentID", comment.ID)
	RespondWithJSON(w, http.StatusCreated, api.NewResponse(resp, "Comment created successfully", false))
}

// HandleDeleteComment handles DELETE /api/agents/{namespace}/{name}/comments/{comment_id}
func (h *CommentsHandler) HandleDeleteComment(w ErrorResponseWriter, r *http.Request) {
	log := ctrllog.FromContext(r.Context()).WithName("comments-handler").WithValues("operation", "delete-comment")

	commentID, err := GetPathParam(r, "comment_id")
	if err != nil {
		w.RespondWithError(errors.NewBadRequestError("Missing comment_id", err))
		return
	}

	userID, err := GetUserID(r)
	if err != nil {
		log.Error(err, "Failed to get user ID")
		w.RespondWithError(errors.NewBadRequestError("Failed to get user ID", err))
		return
	}

	if err := h.DatabaseService.DeleteAgentComment(r.Context(), commentID, userID); err != nil {
		log.Error(err, "Failed to delete agent comment")
		w.RespondWithError(errors.NewInternalServerError("Failed to delete comment", err))
		return
	}

	log.Info("Comment deleted successfully", "commentID", commentID)
	RespondWithJSON(w, http.StatusOK, api.NewResponse(struct{}{}, "Comment deleted successfully", false))
}
