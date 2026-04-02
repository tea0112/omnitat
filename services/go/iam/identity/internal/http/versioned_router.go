package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type VersionedRouter struct {
	mux *chi.Mux
}

func NewVersionedRouter() *VersionedRouter {
	return &VersionedRouter{
		mux: chi.NewRouter(),
	}
}

func (r *VersionedRouter) Use(middlewares ...func(http.Handler) http.Handler) {
	r.mux.Use(middlewares...)
}

func (r *VersionedRouter) V1() chi.Router {
	return r.mux.Route("/v1", func(r chi.Router) {})
}

func (r *VersionedRouter) V2() chi.Router {
	return r.mux.Route("/v2", func(r chi.Router) {})
}

func (r *VersionedRouter) Handler() http.Handler {
	return r.mux
}

type DomainRouter interface {
	RegisterV1(r chi.Router)
}

func RegisterDomain(r chi.Router, domain DomainRouter) {
	domain.RegisterV1(r)
}

func GetChiRouter() *chi.Mux {
	return chi.NewRouter()
}
