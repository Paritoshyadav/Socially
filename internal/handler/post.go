package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/paritoshyadav/socialnetwork/internal/service"
)

type CreatePostInput struct {
	Content   string  `json:"content" validate:"required,min=1,max=5"`
	SpoilerOf *string `json:"spoiler_of" validate:"omitempty,min=1,max=5"`
	NSFW      bool    `json:"nsfw"`
}

// handler createpost

func (h *handler) createPost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var postInput CreatePostInput
	err := json.NewDecoder(r.Body).Decode(&postInput)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	err = ValidateInput(postInput)
	if err != nil {
		http.Error(w, service.ErrValidations.Error(), http.StatusBadRequest)
		fmt.Println(err)
		return
	}
	timelineItem, err := h.CreatePost(ctx, postInput.Content, postInput.SpoilerOf, postInput.NSFW)

	if err == service.ErrUnAuthorized {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err != nil {
		responseError(w, err)
		return
	}

	log.Print(timelineItem.ID)

	response(w, timelineItem, http.StatusCreated)
}

//toggle like post handler
func (h *handler) toggleLikePostHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	postID, err := strconv.ParseInt(chi.URLParam(r, "postID"), 10, 64)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	out, err := h.TogglePostLike(ctx, postID)
	if err == service.ErrUnAuthorized {
		http.Error(w, err.Error(), http.StatusUnauthorized)
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

//get user posts handler
func (h *handler) getUserPostsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	username := chi.URLParam(r, "username")

	q := r.URL.Query()
	last, err := strconv.Atoi(q.Get("last"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	before := q.Get("before")

	out, err := h.PostsByUser(ctx, username, last, before)
	if err == service.ErrUnAuthorized {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if err == service.ErrUserNotFound {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err != nil {
		responseError(w, err)
		return
	}
	response(w, out, http.StatusOK)
}

//get post by id handler
func (h *handler) getPostHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	postID, err := strconv.ParseInt(chi.URLParam(r, "postID"), 10, 64)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	out, err := h.Post(ctx, postID)
	if err == service.ErrUnAuthorized {
		http.Error(w, err.Error(), http.StatusUnauthorized)
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