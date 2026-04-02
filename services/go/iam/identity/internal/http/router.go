package httpapi

import (
	stdHttp "net/http"
	"strings"

	libHttp "github.com/tea0112/omnitat/libs/go/http"
)

// APIRouter combines multiple domain routers under a common prefix
type APIRouter struct {
	prefix  string
	mux     *stdHttp.ServeMux
	routers []libHttp.Router
}

// NewAPIRouter creates a new API router with the specified prefix
func NewAPIRouter(prefix string) *APIRouter {
	return &APIRouter{
		prefix: prefix,
		mux:    stdHttp.NewServeMux(),
	}
}

// Register adds a domain router to the API router
func (r *APIRouter) Register(router libHttp.Router) {
	r.routers = append(r.routers, router)
}

// Routes implements the http.Router interface
func (r *APIRouter) Routes() map[string]stdHttp.Handler {
	routes := make(map[string]stdHttp.Handler)
	for _, router := range r.routers {
		for pattern, handler := range router.Routes() {
			fullPattern := r.prefix + pattern
			routes[fullPattern] = handler
		}
	}
	return routes
}

// Handler returns the http.Handler with all routes registered under prefix
func (r *APIRouter) Handler() stdHttp.Handler {
	for _, router := range r.routers {
		for pattern, handler := range router.Routes() {
			r.mux.Handle(pattern, handler)
		}
	}

	return stdHttp.StripPrefix(strings.TrimSuffix(r.prefix, "/"), r.mux)
}

// RegisterRoutes is a convenience function to register routes directly
func RegisterRoutes(mux *stdHttp.ServeMux, router libHttp.Router) {
	for pattern, handler := range router.Routes() {
		mux.Handle(pattern, handler)
	}
}
