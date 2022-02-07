package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/paritoshyadav/socialnetwork/internal/service"
)

type loginInput struct {
	Email string `validate:"required,email"`
}

func (h *handler) login(w http.ResponseWriter, r *http.Request) {
	var in loginInput
	err := json.NewDecoder(r.Body).Decode(&in)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = ValidateInput(in)
	if err != nil {
		http.Error(w, service.ErrValidations.Error(), http.StatusBadRequest)
		return
	}
	lo, err := h.Login(r.Context(), in.Email)
	if err == service.ErrUserNotFound {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err != nil {
		responseError(w, err)
		return
	}
	response(w, lo, http.StatusOK)

}

func (h *handler) checkUserAuth(w http.ResponseWriter, r *http.Request) {

	user, err := h.Service.AuthUser(r.Context())

	if err == service.ErrUserNotFound {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if err == service.ErrUnAuthorized {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if err != nil {
		responseError(w, err)
		return
	}

	response(w, user, http.StatusOK)

}

func (h *handler) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		token := r.Header.Get("Authorization")

		if !strings.HasPrefix(token, "Bearer") {
			next.ServeHTTP(w, r)
			return

		}

		token = token[7:] //remove prefix Bearer from token

		uid, err := h.Codec.DecodeAuthID(token)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
		}

		ctx := context.WithValue(r.Context(), service.KeyAuthUserID, uid)
		next.ServeHTTP(w, r.WithContext(ctx))

	})

}
