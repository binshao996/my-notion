package main

import (
	"log"
	"net/http"
	"os"

	"github.com/bin-ke/my-notion/internal/auth"
	"github.com/bin-ke/my-notion/internal/workspace"
	"github.com/bin-ke/my-notion/pkg/db"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://notion:notion_dev@localhost:5432/my_notion?sslmode=disable"
	}

	database, err := db.Connect(dsn)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	// Services
	authService := auth.NewService(database)
	authHandler := auth.NewHandler(authService)

	workspaceService := workspace.NewService(database)
	workspaceHandler := workspace.NewHandler(workspaceService)

	// Router
	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	// Auth routes (public)
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
	})

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(authService.Middleware)

		r.Get("/api/v1/auth/me", authHandler.Me)

		r.Route("/api/v1/workspaces", func(r chi.Router) {
			r.Post("/", workspaceHandler.Create)
			r.Get("/", workspaceHandler.List)
			r.Get("/{id}", workspaceHandler.Get)
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("API server starting on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
