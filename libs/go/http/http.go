package http

import "net/http"

type Config struct {
	Port int
}

type Router interface {
	Routes() map[string]http.Handler
}

func RegisterRoutes(mux *http.ServeMux, r Router, middlewares ...Middleware) {
	for pattern, handler := range r.Routes() {
		mux.Handle(pattern, handler)
	}
}
