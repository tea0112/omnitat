package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h *UserHandler) RegisterV1(r chi.Router) {
	r.Route("/users", func(r chi.Router) {
		r.Post("/", http.HandlerFunc(h.CreateUser).ServeHTTP)
	})
}
