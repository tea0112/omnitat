package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h *AuthHandler) RegisterV1(r chi.Router) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/login", http.HandlerFunc(h.Login))
	})
}
