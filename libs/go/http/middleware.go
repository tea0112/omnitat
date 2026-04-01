package http

import "net/http"

type Middleware func(http.Handler) http.Handler

// Chain creates a middleware chain. First arg (m) is outermost; variadic (ms)
// are innermost first. Request flows: m -> ms[0] -> ... -> handler
func Chain(m Middleware, ms ...Middleware) Middleware {
	return func(h http.Handler) http.Handler {
		for i := len(ms) - 1; i >= 0; i-- {
			h = ms[i](h)
		}

		return m(h)
	}
}
