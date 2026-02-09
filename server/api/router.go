package api

import (
	"net/http"

	"github.com/anamika/zenact-web/server/agent"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func NewRouter(ag *agent.Agent) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	h := &Handler{agent: ag}
	r.Route("/api", func(r chi.Router) {
		r.Post("/task", h.CreateTask)
		r.Get("/task/{id}", h.GetTask)
		r.Get("/task/{id}/ws", h.TaskWebSocket)
	})

	return r
}
