package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/paritoshyadav/socialnetwork/internal/service"
)

// get notification hanlder
func (h *handler) getNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	before := q.Get("before")
	last, err := strconv.Atoi(q.Get("last"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	notifications, err := h.Notifications(ctx, last, before)

	if err == service.ErrUnAuthorized {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err != nil {
		responseError(w, err)
		return
	}
	response(w, notifications, http.StatusOK)
}

//mark all notifications as read
func (h *handler) markAllNotificationsAsReadHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	err := h.MarkNotificationsRead(ctx)
	if err == service.ErrUnAuthorized {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err != nil {
		responseError(w, err)
		return
	}
	response(w, nil, http.StatusOK)

}

func (h *handler) markNotificationAsReadHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	notificationId, err := strconv.ParseInt(chi.URLParam(r, "notificationID"), 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = h.MarkNotificationRead(ctx, notificationId)
	if err == service.ErrUnAuthorized {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err != nil {
		responseError(w, err)
		return
	}
	response(w, nil, http.StatusOK)

}
