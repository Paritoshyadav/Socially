package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/paritoshyadav/socialnetwork/internal/service"
)

type CommentInput struct {
	Content string `json:"content" validate:"required,min=1,max=40"`
}

//create comment handler
func (h *handler) createCommentHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var commentInput CommentInput
	err := json.NewDecoder(r.Body).Decode(&commentInput)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, service.ErrValidations.Error(), http.StatusBadRequest)
		return
	}
	commentInput.Content = strings.TrimSpace(commentInput.Content)
	err = ValidateInput(commentInput)
	if err != nil {
		http.Error(w, service.ErrValidations.Error(), http.StatusBadRequest)
		return
	}
	postID, err := strconv.ParseInt(chi.URLParam(r, "postID"), 10, 64)
	if err != nil {
		http.Error(w, service.ErrValidations.Error(), http.StatusBadRequest)
		return
	}
	out, err := h.CreateComment(ctx, commentInput.Content, postID)
	if err == service.ErrUnAuthorized {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if err == service.ErrUserNotFound {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err == service.ErrPostNotFound {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err != nil {
		responseError(w, err)
		return
	}
	response(w, out, http.StatusCreated)
}

//getCommentsHandler
func (h *handler) getCommentsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	postID, err := strconv.ParseInt(chi.URLParam(r, "postID"), 10, 64)
	if err != nil {
		http.Error(w, service.ErrValidations.Error(), http.StatusBadRequest)
		return
	}
	q := r.URL.Query()
	last, err := strconv.Atoi(q.Get("last"))
	if err != nil {
		http.Error(w, service.ErrValidations.Error(), http.StatusBadRequest)
		return

	}
	before := q.Get("before")

	out, err := h.GetPostComments(ctx, postID, last, before)
	if err == service.ErrUserNotFound {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err == service.ErrPostNotFound {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err != nil {
		responseError(w, err)
		return
	}
	response(w, out, http.StatusOK)

}
