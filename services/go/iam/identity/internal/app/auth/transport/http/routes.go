package http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	httpapi "github.com/tea0112/omnitat/services/go/iam/identity/internal/http"
)

func (h *AuthHandler) RegisterV1(r chi.Router) {
	signInLimiter := httpapi.NewRateLimiter(httpapi.RateLimitConfig{Requests: 5, Window: time.Minute, Burst: 5, Message: "too many signin attempts"})
	signUpLimiter := httpapi.NewRateLimiter(httpapi.RateLimitConfig{Requests: 5, Window: time.Minute, Burst: 5, Message: "too many signup attempts"})
	refreshLimiter := httpapi.NewRateLimiter(httpapi.RateLimitConfig{Requests: 10, Window: time.Minute, Burst: 10, Message: "too many refresh attempts"})
	authMiddleware := httpapi.RequireBearerAuth(h.jwtAccessSecret)

	r.Route("/auth", func(r chi.Router) {
		r.With(signInLimiter.Middleware).Post("/signin", http.HandlerFunc(h.SignIn))
		r.With(signUpLimiter.Middleware).Post("/signup", http.HandlerFunc(h.SignUp))
		r.With(refreshLimiter.Middleware).Post("/refresh", http.HandlerFunc(h.Refresh))
		r.Post("/logout", http.HandlerFunc(h.Logout))
		r.With(authMiddleware).Get("/sessions", http.HandlerFunc(h.ListSessions))
		r.With(authMiddleware).Delete("/sessions/{sessionID}", http.HandlerFunc(h.RevokeSession))
	})
}
