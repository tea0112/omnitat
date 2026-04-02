package http

import (
	"net/http"

	libHttp "github.com/tea0112/omnitat/libs/go/http"
)

// Routes implements the http.Router interface
func (h *UserHandler) Routes() map[string]http.Handler {
	return map[string]http.Handler{
		"/users": http.HandlerFunc(h.CreateUser),
	}
}

// RegisterRoutes registers user routes to a mux (legacy function kept for compatibility)
// Deprecated: Use http.RegisterRoutes(mux, userHandler) instead
func RegisterRoutes(mux *http.ServeMux, handler *UserHandler) {
	libHttp.RegisterRoutes(mux, handler)
}
