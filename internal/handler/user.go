package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/paritoshyadav/socialnetwork/internal/service"
)

type createUserInfoRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Username string `json:"username" validate:"required,alphanum"`
}

func (h *handler) createUser(w http.ResponseWriter, r *http.Request) {
	var userInput *createUserInfoRequest
	defer r.Body.Close()
	err := json.NewDecoder(r.Body).Decode(&userInput)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = ValidateInput(userInput)
	if err != nil {
		http.Error(w, service.ErrValidations.Error(), http.StatusBadRequest)
		return
	}
	err = h.CreateUser(r.Context(), userInput.Email, userInput.Username)
	if err == service.ErrEmailTaken || err == service.ErrUsernameTaken {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusCreated)

}

func ValidateUsername(username string) error {
	return validator.New().Var(username, "required,alphanum")
}

func (h *handler) toggleFollow(w http.ResponseWriter, r *http.Request) {

	username := chi.URLParam(r, "username")
	err := ValidateUsername(username)
	if err != nil {
		http.Error(w, service.ErrValidations.Error(), http.StatusBadRequest)
		return
	}

	out, err := h.ToggleFollow(r.Context(), username)
	if err == service.ErrUnAuthorized {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return

	}

	response(w, out, http.StatusOK)

}

func (h *handler) userProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	username := chi.URLParam(r, "username")
	err := ValidateUsername(username)
	if err != nil {
		http.Error(w, service.ErrValidations.Error(), http.StatusBadRequest)
		return
	}
	out, err := h.User(ctx, username)
	if err == service.ErrUnAuthorized {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return

	}

	response(w, out, http.StatusOK)

}

func (h *handler) updateAvatar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	r.Body = http.MaxBytesReader(w, r.Body, service.MaxAvatarBytes)
	defer r.Body.Close()
	out, err := h.UpdateAvatar(ctx, r.Body)
	if err == service.ErrUnAuthorized {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if err == service.ErrUnSpportedAvatarFormat {
		http.Error(w, err.Error(), http.StatusUnsupportedMediaType)
		return
	}

	if err != nil {
		responseError(w, err)
		return

	}
	fmt.Fprint(w, out)
	// response(w, out, http.StatusOK)

}

func (h *handler) searchUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	search := q.Get("search")
	first, err := strconv.Atoi(q.Get("first"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	after := q.Get("after")

	profiles, err := h.Users(ctx, search, first, after)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response(w, profiles, http.StatusOK)

}

func (h *handler) followers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	username := chi.URLParam(r, "username")
	err := ValidateUsername(username)
	if err != nil {
		http.Error(w, service.ErrValidations.Error(), http.StatusBadRequest)
		return
	}

	first, err := strconv.Atoi(q.Get("first"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	after := q.Get("after")

	profiles, err := h.UserFollowers(ctx, username, first, after)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response(w, profiles, http.StatusOK)

}

func (h *handler) followings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	username := chi.URLParam(r, "username")
	err := ValidateUsername(username)
	if err != nil {
		http.Error(w, service.ErrValidations.Error(), http.StatusBadRequest)
		return
	}
	first, err := strconv.Atoi(q.Get("first"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	after := q.Get("after")

	profiles, err := h.UserFollowings(ctx, username, first, after)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response(w, profiles, http.StatusOK)

}
