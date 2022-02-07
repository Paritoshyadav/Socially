package handler

import (
	"net/http"
	"strconv"

	"github.com/paritoshyadav/socialnetwork/internal/service"
)

//getTimeline handler
func (h *handler) getTimeline(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	last, err := strconv.Atoi(q.Get("last"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	before := q.Get("before")

	out, err := h.RetrieveTimelineItems(ctx, last, before)
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
