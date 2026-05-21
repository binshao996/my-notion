package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/bin-ke/my-notion/internal/auth"
	"github.com/bin-ke/my-notion/internal/block"
	databasepkg "github.com/bin-ke/my-notion/internal/database"
	"github.com/bin-ke/my-notion/internal/file"
	"github.com/bin-ke/my-notion/internal/page"

	"github.com/bin-ke/my-notion/internal/collaboration"
	"github.com/bin-ke/my-notion/internal/comment"
	"github.com/bin-ke/my-notion/internal/notification"
	"github.com/bin-ke/my-notion/internal/permission"
	"github.com/bin-ke/my-notion/internal/share"

	"github.com/bin-ke/my-notion/internal/search"
	"github.com/bin-ke/my-notion/internal/workspace"
	"github.com/bin-ke/my-notion/pkg/db"
	myredis "github.com/bin-ke/my-notion/pkg/redis"
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

	// Redis connection (non-fatal if unavailable)
	redisAddr := os.Getenv("REDIS_URL")
	if redisAddr == "" {
		redisAddr = "redis://localhost:6379"
	}
	rdb, redisErr := myredis.Connect(redisAddr)
	if redisErr != nil {
		log.Printf("WARNING: redis not available (caching and async search disabled): %v", redisErr)
	}

	// File service (non-fatal if MinIO is down)
	fileService, fileErr := file.NewService()
	if fileErr != nil {
		log.Printf("WARNING: file service not available: %v", fileErr)
	}

	// Services
	authService := auth.NewService(database)
	authHandler := auth.NewHandler(authService)

	notifService := notification.NewService(database)

	workspaceService := workspace.NewService(database)
	workspaceHandler := workspace.NewHandler(workspaceService, notifService)

	pageService := page.NewService(database)
	pageHandler := page.NewHandler(pageService)

	blockService := block.NewService(database)
	blockHandler := block.NewHandler(blockService)

	// Database services
	databaseService := databasepkg.NewService(database)
	propertyService := databasepkg.NewPropertyService(database)
	recordService := databasepkg.NewRecordService(database)
	queryService := databasepkg.NewQueryService(database)
	viewService := databasepkg.NewViewService(database)

	databaseHandler := databasepkg.NewHandler(databaseService, propertyService, recordService, viewService)
	propertyHandler := databasepkg.NewPropertyHandler(propertyService)
	recordHandler := databasepkg.NewRecordHandler(recordService, queryService)
	viewHandler := databasepkg.NewViewHandler(viewService)

	// M3 services
	permissionService := permission.NewService(database)
	if rdb != nil {
		permissionService.SetRedisClient(rdb)
	}
	permissionHandler := permission.NewHandler(permissionService)

	shareService := share.NewService(database)
	shareHandler := share.NewHandler(shareService)
	shareHandler.NotificationService = notifService

	notifHandler := notification.NewHandler(notifService)

	commentService := comment.NewService(database, notifService)
	commentHandler := comment.NewHandler(commentService, permissionService)

	// M4 collaboration
	collabService := collaboration.NewService(database)
	collabDocStore := collaboration.NewDocStore()
	collabHub := collaboration.NewHub()
	collabHandler := collaboration.NewHandler(collabHub, collabDocStore, authService, authService.UserService)

	// Load existing snapshots from DB into memory
	if err := collabService.LoadAllSnapshots(collabDocStore); err != nil {
		log.Printf("WARNING: failed to load collaboration snapshots: %v", err)
	}

	// Wire hub -> docstore persistence
	collabHub.SetOnUpdate(func(pageID uint, data []byte) {
		collabDocStore.AppendUpdate(pageID, data)
	})
	collabHub.SetOnRoomEmpty(func(pageID uint) {
		collaboration.FlushOnEmpty(collabDocStore, collabService, pageID)
	})

	// Start periodic snapshot flush (every 30s)
	collaboration.StartFlushLoop(collabDocStore, collabService, collabHub, 30*time.Second)

	// Search service (non-fatal if OpenSearch is down)
	searchService, searchErr := search.NewService()
	if searchErr != nil {
		log.Printf("WARNING: search service not available: %v", searchErr)
	} else {
		// Wire Redis for async search indexing (if available)
		if rdb != nil {
			searchService.RedisClient = rdb
			log.Println("search: async indexing via Redis queue enabled")
		}
		pageService.SearchService = searchService
		blockService.SearchService = searchService
		recordService.SearchService = searchService
	}

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

	// Share token resolution (public, no auth required)
	r.Get("/api/v1/share/{token}", shareHandler.ResolveToken)

	// WebSocket collaboration (public route, JWT validated in handler)
	r.Get("/ws/page/{id}", collabHandler.HandleWebSocket)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(authService.Middleware)

		r.Get("/api/v1/auth/me", authHandler.Me)

		r.Route("/api/v1/workspaces", func(r chi.Router) {
			r.Post("/", workspaceHandler.Create)
			r.Get("/", workspaceHandler.List)
			r.Get("/{id}", workspaceHandler.Get)
			r.Get("/{id}/tree", pageHandler.GetTree)
			r.Patch("/{id}", workspaceHandler.Update)
			r.Delete("/{id}", workspaceHandler.Delete)
			r.Get("/{id}/members", workspaceHandler.ListMembers)
			r.Post("/{id}/members", workspaceHandler.AddMember)
			r.Delete("/{id}/members/{memberId}", workspaceHandler.RemoveMember)
		})

		r.Route("/api/v1/pages", func(r chi.Router) {
			// Create page -- workspace membership checked in handler
			r.Post("/", pageHandler.Create)

			// Viewer-level access for reads
			r.Group(func(r chi.Router) {
				r.Use(permission.PageAccess(permissionService, "viewer"))
				r.Get("/{id}", pageHandler.Get)
				r.Get("/{id}/children", pageHandler.GetChildren)
				r.Get("/{pageId}/blocks", blockHandler.GetByPage)
			})

			// Editor-level access for writes
			r.Group(func(r chi.Router) {
				r.Use(permission.PageAccess(permissionService, "editor"))
				r.Patch("/{id}", pageHandler.Update)
				r.Put("/{pageId}/blocks", blockHandler.BatchSave)
				r.Post("/{pageId}/blocks/ops", blockHandler.ApplyOps)
			})
		})

		// Permissions
		r.Route("/api/v1/pages/{id}/permissions", func(r chi.Router) {
			r.Use(permission.PageAccess(permissionService, "editor"))
			r.Get("/", permissionHandler.ListByPage)
			r.Post("/", permissionHandler.Set)
			r.Delete("/{permId}", permissionHandler.Remove)
		})

		// Share tokens
		r.Route("/api/v1/pages/{id}/share-tokens", func(r chi.Router) {
			r.Use(permission.PageAccess(permissionService, "editor"))
			r.Get("/", shareHandler.ListByPage)
			r.Post("/", shareHandler.Create)
			r.Delete("/{tokId}", shareHandler.Revoke)
		})

		// Comments
		r.Route("/api/v1/pages/{id}", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(permission.PageAccess(permissionService, "viewer"))
				r.Get("/comments", commentHandler.ListByPage)
			})
			r.Group(func(r chi.Router) {
				r.Use(permission.PageAccess(permissionService, "commenter"))
				r.Post("/comments", commentHandler.Create)
			})
		})

		r.Patch("/api/v1/comments/{id}", commentHandler.Update)
		r.Delete("/api/v1/comments/{id}", commentHandler.Delete)

		// Notifications
		r.Get("/api/v1/notifications", notifHandler.ListByUser)
		r.Patch("/api/v1/notifications/read-all", notifHandler.MarkAllRead)
		r.Patch("/api/v1/notifications/{id}/read", notifHandler.MarkRead)

		// Database routes
		r.Route("/api/v1/databases", func(r chi.Router) {
			r.Post("/", databaseHandler.Create)
			r.Get("/{id}", databaseHandler.Get)
			r.Patch("/{id}", databaseHandler.Update)
			r.Delete("/{id}", databaseHandler.Delete)

			r.Post("/{id}/properties", propertyHandler.Create)
			r.Get("/{id}/records", recordHandler.List)
			r.Post("/{id}/records", recordHandler.Create)

			r.Post("/{id}/views", viewHandler.Create)
			r.Get("/{id}/views/{viewId}/records", recordHandler.ListByView)
		})

		r.Patch("/api/v1/properties/{id}", propertyHandler.Update)
		r.Delete("/api/v1/properties/{id}", propertyHandler.Delete)

		r.Get("/api/v1/records/{id}", recordHandler.Get)
		r.Patch("/api/v1/records/{id}", recordHandler.Update)
		r.Delete("/api/v1/records/{id}", recordHandler.Delete)

		r.Patch("/api/v1/views/{id}", viewHandler.Update)
		r.Delete("/api/v1/views/{id}", viewHandler.Delete)

		// File upload
		if fileService != nil {
			fileHandler := file.NewHandler(fileService)
			r.Route("/api/v1/files", func(r chi.Router) {
				r.Post("/upload-url", fileHandler.GetUploadURL)
			})
		}

		// Search
		if searchService != nil {
			searchHandler := search.NewHandler(searchService, database, permissionService)
			r.Get("/api/v1/search", searchHandler.Search)
			r.Post("/api/v1/search/reindex", searchHandler.Reindex)
		}
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
