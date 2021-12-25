package controller

import (
	"github.com/go-chi/chi"
	"net/http"
)

func InitControllers() *chi.Mux {
	r := chi.NewRouter()

	InitHealthCheckEndpoints(r)
	return r
}

func InitHealthCheckEndpoints(r *chi.Mux) {
	r.Get("/healthcheck", func(res http.ResponseWriter, req *http.Request) {})
}
