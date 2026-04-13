package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h *AuthHandler) RegisterV1(r chi.Router) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/signin", http.HandlerFunc(h.SignIn))
		r.Post("/signup", http.HandlerFunc(h.SignUp))
		r.Post("/refresh", http.HandlerFunc(h.Refresh))
		r.Post("/logout", http.HandlerFunc(h.Logout))
	})
}
