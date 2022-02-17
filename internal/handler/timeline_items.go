package handler

import (
	"encoding/json"
	"mime"
	"net/http"
	"strconv"

	"github.com/paritoshyadav/socialnetwork/internal/service"
)

//getTimeline handler
func (h *handler) getTimeline(w http.ResponseWriter, r *http.Request) {
	// check if header accept event stream
	if a, _, err := mime.ParseMediaType(r.Header.Get("Accept")); err == nil && a == "text/event-stream" {

		h.subscribeToTimeline(w, r)
		return
	}

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

func (h *handler) subscribeToTimeline(w http.ResponseWriter, r *http.Request) {
	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}
	tt, err := h.SubscribeToTimeline(r.Context())
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

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for ti := range tt {
		err := json.NewEncoder(w).Encode(ti)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		f.Flush()
	}

}
