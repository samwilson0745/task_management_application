package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"taskmanager/internal/handlers"
	"taskmanager/internal/middleware"
)

type Deps struct {
	JWTSecret         string
	AllowedOrigin     string
	AuthHandler       *handlers.AuthHandler
	TaskHandler       *handlers.TaskHandler
	AttachmentHandler *handlers.AttachmentHandler
}

func New(d Deps) http.Handler {
	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{d.AllowedOrigin},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/auth", func(r chi.Router) {
		r.Post("/signup", d.AuthHandler.Signup)
		r.Post("/login", d.AuthHandler.Login)

		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(d.JWTSecret))
			r.Get("/me", d.AuthHandler.Me)
		})
	})

	r.Route("/tasks", func(r chi.Router) {
		r.Use(middleware.Auth(d.JWTSecret))
		r.Get("/", d.TaskHandler.List)
		r.Post("/", d.TaskHandler.Create)
		r.Get("/stream", d.TaskHandler.Stream)
		r.Get("/{id}", d.TaskHandler.Get)
		r.Patch("/{id}", d.TaskHandler.Update)
		r.Delete("/{id}", d.TaskHandler.Delete)
		r.Get("/{id}/activity", d.TaskHandler.Activity)

		r.Get("/{id}/attachments", d.AttachmentHandler.List)
		r.Post("/{id}/attachments", d.AttachmentHandler.Upload)
		r.Get("/{id}/attachments/{attachmentID}", d.AttachmentHandler.Download)
		r.Delete("/{id}/attachments/{attachmentID}", d.AttachmentHandler.Delete)
	})

	return r
}
