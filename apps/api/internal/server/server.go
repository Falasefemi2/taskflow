package server

import (
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	db "github.con/falasefemi2/taskflow/api/db/generated"
	"github.con/falasefemi2/taskflow/api/internal/auth"
	"github.con/falasefemi2/taskflow/api/internal/config"
	mw "github.con/falasefemi2/taskflow/api/internal/middleware"
	"github.con/falasefemi2/taskflow/api/internal/workspace"
)

func New(pool *pgxpool.Pool, cfg *config.Config) http.Handler {
	r := chi.NewRouter()
	queries := db.New(pool)
	authHandler := auth.NewHandler(queries, cfg)
	workspaceHandler := workspace.NewHandler(queries, cfg)

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.Server.AllowedOrigin},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("ok"))
		if err != nil {
			http.Error(w, "failed to write response", http.StatusInternalServerError)
			return
		}
	})

	// public
	r.Route("/api/v1/auth", func(ar chi.Router) {
		ar.Post("/register", authHandler.Register)
		ar.Post("/login", authHandler.Login)
		ar.Post("/refresh", authHandler.Refresh)
		ar.Post("/logout", authHandler.Logout)
		ar.Post("/forgot-password", authHandler.ForgotPassword)
		ar.Post("/reset-password", authHandler.ResetPassword)
	})

	// protected
	r.Group(func(r chi.Router) {
		r.Use(mw.AuthMiddleware(cfg.Auth.JWTSecret))

		r.Get("/api/v1/auth/me", authHandler.Me)
		r.Post("/api/v1/workspaces", workspaceHandler.CreateWorkspace)
		r.Get("/api/v1/workspaces", workspaceHandler.ListWorkspaces)
		r.Get("/{id}", workspaceHandler.GetWorkspace)
		r.Put("/{id}", workspaceHandler.UpdateWorkspace)
		r.Delete("/{id}", workspaceHandler.DeleteWorkspace)
	})

	return r
}
