package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/paritoshyadav/socialnetwork/internal/service"
)

type handler struct {
	*service.Service
}

func New(s *service.Service) http.Handler {
	h := &handler{s}

	api := chi.NewRouter()

	api.Post("/login", h.login)
	api.Post("/users", h.createUser)

	api.Route("/api", func(r chi.Router) {
		r.Use(h.withAuth)
		r.Post("/login", h.login)
		r.Get("/auth", h.checkUserAuth)
		r.Route("/posts", func(r chi.Router) {
			r.Post("/", h.createPost)
			r.Post("/{postID}/toggle_likes", h.toggleLikePostHandler)
			r.Get("/{postID}", h.getPostHandler)
			r.Post("/{postID}/comments", h.createCommentHandler)

		})
		r.Get("/timeline", h.getTimeline)

		r.Route("/users", func(r chi.Router) {
			r.Post("/", h.createUser)
			r.Get("/", h.searchUser)
			r.Post("/{username}/follow", h.toggleFollow)
			r.Get("/{username}", h.userProfile)
			r.Get("/{username}/followers", h.followers)
			r.Get("/{username}/followings", h.followings)
			r.Get("/{username}/posts", h.getUserPostsHandler)
			r.Put("/avatar", h.updateAvatar)
		})

	})

	return api

}
