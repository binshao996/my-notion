# M0: Infrastructure, Auth & Workspace — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Scaffold monorepo with Go backend + React frontend, local dev with Docker Compose, user registration/login with JWT, and workspace CRUD.

**Architecture:** pnpm workspaces monorepo. Go REST API at `apps/server/` (modular monolith, GORM + PostgreSQL). React SPA at `apps/web/` (Vite + Tailwind + Zustand). Docker Compose for local PostgreSQL + Redis + MinIO.

**Tech Stack:** Go 1.22+, GORM, chi router, JWT (golang-jwt), React 18, TypeScript, Vite, Tailwind CSS, Zustand, PostgreSQL 16, Redis 7

---

## File Structure

```
my-notion/
├── pnpm-workspace.yaml              # [create] defines apps/* and packages/*
├── package.json                      # [create] root package.json (scripts only)
├── .gitignore                        # [modify] add .superpowers/, dist/, tmp/
├── .github/workflows/ci.yml          # [create] lint + test + build
│
├── docker/
│   └── docker-compose.yml            # [create] Postgres 16 + Redis 7 + MinIO
│
├── apps/
│   ├── server/
│   │   ├── go.mod                    # [create] module github.com/user/my-notion
│   │   ├── cmd/
│   │   │   ├── api/main.go           # [create] API server entrypoint
│   │   │   └── worker/main.go        # [create] worker entrypoint (noop for M0)
│   │   ├── internal/
│   │   │   ├── auth/
│   │   │   │   ├── handler.go        # [create] register/login handlers
│   │   │   │   ├── service.go        # [create] auth business logic
│   │   │   │   └── middleware.go     # [create] JWT auth middleware
│   │   │   ├── workspace/
│   │   │   │   ├── handler.go        # [create] workspace CRUD handlers
│   │   │   │   └── service.go        # [create] workspace business logic
│   │   │   └── user/
│   │   │       └── service.go        # [create] user queries
│   │   ├── pkg/
│   │   │   ├── db/
│   │   │   │   └── db.go             # [create] GORM connection + auto-migrate
│   │   │   └── middleware/
│   │   │       └── cors.go           # [create] CORS middleware
│   │   └── migrations/
│   │       └── 001_init.sql          # [create] initial schema
│   │
│   └── web/
│       ├── package.json              # [create] Vite + React + deps
│       ├── vite.config.ts            # [create] Vite config with proxy
│       ├── tailwind.config.ts        # [create] Tailwind config
│       ├── postcss.config.js         # [create] PostCSS config
│       ├── tsconfig.json             # [create] TypeScript config
│       ├── index.html                # [create] HTML entry
│       └── src/
│           ├── main.tsx              # [create] React entry
│           ├── App.tsx               # [create] App with router
│           ├── lib/
│           │   └── api.ts            # [create] API client (fetch wrapper)
│           ├── stores/
│           │   └── auth.ts           # [create] auth Zustand store
│           ├── pages/
│           │   ├── Login.tsx         # [create] login page
│           │   ├── Register.tsx      # [create] register page
│           │   └── Workspace.tsx     # [create] workspace home (empty state)
│           └── components/
│               └── ProtectedRoute.tsx # [create] auth guard wrapper
│
└── packages/
    └── shared-types/                 # [create] stub for future shared types
        └── package.json
```

---

### Task 1: Scaffold monorepo root

**Files:**
- Create: `pnpm-workspace.yaml`
- Create: `package.json`
- Modify: `.gitignore`

- [ ] **Step 1: Create pnpm-workspace.yaml**

```bash
cat > pnpm-workspace.yaml << 'EOF'
packages:
  - "apps/*"
  - "packages/*"
EOF
```

- [ ] **Step 2: Create root package.json**

```bash
cat > package.json << 'EOF'
{
  "name": "my-notion",
  "private": true,
  "scripts": {
    "dev": "pnpm --filter web dev",
    "dev:server": "cd apps/server && go run ./cmd/api",
    "build": "pnpm --filter web build",
    "lint": "pnpm --filter web lint",
    "test": "pnpm --filter web test"
  }
}
EOF
```

- [ ] **Step 3: Add .superpowers/ to .gitignore**

Append to `.gitignore`:
```
.superpowers/
dist/
tmp/
```

- [ ] **Step 4: Install pnpm and verify**

```bash
pnpm --version
```

Expected: pnpm version >= 8

- [ ] **Step 5: Commit**

```bash
git add pnpm-workspace.yaml package.json .gitignore
git commit -m "chore: scaffold monorepo root with pnpm workspaces"
```

---

### Task 2: Docker Compose for local dev

**Files:**
- Create: `docker/docker-compose.yml`

- [ ] **Step 1: Create docker-compose.yml**

```yaml
version: "3.8"

services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: notion
      POSTGRES_PASSWORD: notion_dev
      POSTGRES_DB: my_notion
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  minio:
    image: minio/minio
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    ports:
      - "9000:9000"
      - "9001:9001"
    volumes:
      - miniodata:/data

volumes:
  pgdata:
  miniodata:
```

- [ ] **Step 2: Start services and verify**

```bash
docker compose -f docker/docker-compose.yml up -d
docker compose -f docker/docker-compose.yml ps
```

Expected: postgres, redis, minio all "Up"

- [ ] **Step 3: Commit**

```bash
git add docker/docker-compose.yml
git commit -m "chore: add Docker Compose for local dev (PostgreSQL + Redis + MinIO)"
```

---

### Task 3: Scaffold Go backend module

**Files:**
- Create: `apps/server/go.mod`

- [ ] **Step 1: Create Go module directory and init**

```bash
mkdir -p apps/server/cmd/api apps/server/cmd/worker
mkdir -p apps/server/internal/auth apps/server/internal/workspace apps/server/internal/user
mkdir -p apps/server/pkg/db apps/server/pkg/middleware
mkdir -p apps/server/migrations
cd apps/server && go mod init github.com/bin-ke/my-notion
```

- [ ] **Step 2: Create cmd/api/main.go (skeleton)**

```go
package main

import (
	"log"
	"net/http"
	"os"

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

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("API server starting on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("server failed: %v", err)
	}

	_ = database // will be used by handlers
}
```

- [ ] **Step 3: Create cmd/worker/main.go (noop skeleton)**

```go
package main

import "log"

func main() {
	log.Println("Worker started (noop for M0)")
	select {}
}
```

- [ ] **Step 4: Add Go dependencies, verify build**

```bash
cd apps/server
go get github.com/go-chi/chi/v5 github.com/go-chi/cors
go mod tidy
go build ./cmd/api
go build ./cmd/worker
```

Expected: both binaries compile without errors.

- [ ] **Step 5: Commit**

```bash
cd /path/to/root && git add apps/server/
git commit -m "feat(server): scaffold Go backend with chi router skeleton"
```

---

### Task 4: Database connection + migration

**Files:**
- Create: `apps/server/pkg/db/db.go`
- Create: `apps/server/migrations/001_init.sql`

- [ ] **Step 1: Create models and DB connection**

`apps/server/pkg/db/db.go`:

```go
package db

import (
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Models
type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Email        string    `gorm:"uniqueIndex;not null" json:"email"`
	Name         string    `gorm:"not null" json:"name"`
	AvatarURL    string    `json:"avatar_url"`
	PasswordHash string    `gorm:"not null" json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Workspace struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	Icon      string    `json:"icon"`
	CreatedAt time.Time `json:"created_at"`
}

type WorkspaceMember struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	WorkspaceID uint      `gorm:"not null;index" json:"workspace_id"`
	UserID      uint      `gorm:"not null;index" json:"user_id"`
	Role        string    `gorm:"not null;default:member" json:"role"` // owner | member
	JoinedAt    time.Time `json:"joined_at"`
	Workspace   Workspace `gorm:"foreignKey:WorkspaceID" json:"-"`
	User        User      `gorm:"foreignKey:UserID" json:"-"`
}

func Connect(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&User{}, &Workspace{}, &WorkspaceMember{}); err != nil {
		return nil, err
	}

	return db, nil
}
```

- [ ] **Step 2: Create initial SQL migration (reference)**

`apps/server/migrations/001_init.sql`:

```sql
-- M0: Initial schema (users, workspaces, workspace_members)
-- Note: GORM AutoMigrate creates these tables. This file is for reference and manual migrations.

CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    avatar_url TEXT DEFAULT '',
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS workspaces (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    icon VARCHAR(255) DEFAULT '',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS workspace_members (
    id BIGSERIAL PRIMARY KEY,
    workspace_id BIGINT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL DEFAULT 'member',
    joined_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(workspace_id, user_id)
);

CREATE INDEX idx_workspace_members_user ON workspace_members(user_id);
CREATE INDEX idx_workspace_members_workspace ON workspace_members(workspace_id);
```

- [ ] **Step 3: Install GORM + postgres driver, verify build**

```bash
cd apps/server
go get gorm.io/gorm gorm.io/driver/postgres
go mod tidy
go build ./cmd/api
```

Expected: compiles without errors.

- [ ] **Step 4: Test DB connection**

```bash
# With docker-compose running:
cd apps/server && DATABASE_URL="postgres://notion:notion_dev@localhost:5432/my_notion?sslmode=disable" go run ./cmd/api
# Check health endpoint: curl http://localhost:8080/health
```

Expected: "ok" response, no crash. Tables created in PostgreSQL.

- [ ] **Step 5: Commit**

```bash
git add apps/server/pkg/db/ apps/server/migrations/ apps/server/go.mod apps/server/go.sum
git commit -m "feat(server): add GORM models, DB connection, and initial migration"
```

---

### Task 5: User service (password hashing + queries)

**Files:**
- Create: `apps/server/internal/user/service.go`

- [ ] **Step 1: Create user service**

`apps/server/internal/user/service.go`:

```go
package user

import (
	"errors"

	"github.com/bin-ke/my-notion/pkg/db"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Service struct {
	DB *gorm.DB
}

func NewService(database *gorm.DB) *Service {
	return &Service{DB: database}
}

func (s *Service) Create(email, name, password string) (*db.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &db.User{
		Email:        email,
		Name:         name,
		PasswordHash: string(hash),
	}

	if err := s.DB.Create(user).Error; err != nil {
		return nil, errors.New("email already registered")
	}

	return user, nil
}

func (s *Service) FindByEmail(email string) (*db.User, error) {
	var user db.User
	if err := s.DB.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, errors.New("user not found")
	}
	return &user, nil
}

func (s *Service) FindByID(id uint) (*db.User, error) {
	var user db.User
	if err := s.DB.First(&user, id).Error; err != nil {
		return nil, errors.New("user not found")
	}
	return &user, nil
}

func (s *Service) CheckPassword(user *db.User, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	return err == nil
}
```

- [ ] **Step 2: Add bcrypt dependency, verify build**

```bash
cd apps/server
go get golang.org/x/crypto/bcrypt
go mod tidy
go build ./...
```

Expected: compiles without errors.

- [ ] **Step 3: Commit**

```bash
git add apps/server/internal/user/ apps/server/go.mod apps/server/go.sum
git commit -m "feat(server): add user service with bcrypt password hashing"
```

---

### Task 6: Auth service (JWT + register/login)

**Files:**
- Create: `apps/server/internal/auth/service.go`
- Create: `apps/server/internal/auth/middleware.go`

- [ ] **Step 1: Create auth service**

`apps/server/internal/auth/service.go`:

```go
package auth

import (
	"errors"
	"os"
	"time"

	"github.com/bin-ke/my-notion/internal/user"
	"github.com/bin-ke/my-notion/pkg/db"
	"github.com/golang-jwt/jwt/v5"
)

type Service struct {
	UserService *user.Service
}

func NewService(database *gorm.DB) *Service {
	return &Service{
		UserService: user.NewService(database),
	}
}

type Claims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func (s *Service) Register(email, name, password string) (*db.User, error) {
	return s.UserService.Create(email, name, password)
}

func (s *Service) Login(email, password string) (string, *db.User, error) {
	u, err := s.UserService.FindByEmail(email)
	if err != nil {
		return "", nil, errors.New("invalid email or password")
	}

	if !s.UserService.CheckPassword(u, password) {
		return "", nil, errors.New("invalid email or password")
	}

	token, err := s.generateToken(u)
	if err != nil {
		return "", nil, err
	}

	return token, u, nil
}

func (s *Service) generateToken(u *db.User) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-secret-change-in-production"
	}

	claims := &Claims{
		UserID: u.ID,
		Email:  u.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(72 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func (s *Service) ValidateToken(tokenString string) (*Claims, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-secret-change-in-production"
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
```

- [ ] **Step 2: Create JWT auth middleware**

`apps/server/internal/auth/middleware.go`:

```go
package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const UserContextKey contextKey = "user"

func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, `{"error":"invalid authorization format"}`, http.StatusUnauthorized)
			return
		}

		claims, err := s.ValidateToken(parts[1])
		if err != nil {
			http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetUserFromContext(ctx context.Context) *Claims {
	claims, ok := ctx.Value(UserContextKey).(*Claims)
	if !ok {
		return nil
	}
	return claims
}
```

- [ ] **Step 3: Add JWT dependency, verify build**

```bash
cd apps/server
go get github.com/golang-jwt/jwt/v5
go mod tidy
go build ./...
```

Expected: compiles without errors.

- [ ] **Step 4: Commit**

```bash
git add apps/server/internal/auth/ apps/server/go.mod apps/server/go.sum
git commit -m "feat(server): add auth service with JWT token generation and middleware"
```

---

### Task 7: Auth HTTP handlers

**Files:**
- Create: `apps/server/internal/auth/handler.go`

- [ ] **Step 1: Create auth handler**

`apps/server/internal/auth/handler.go`:

```go
package auth

import (
	"encoding/json"
	"net/http"
)

type Handler struct {
	Service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{Service: service}
}

type registerRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string `json:"token"`
	User  struct {
		ID        uint   `json:"id"`
		Email     string `json:"email"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
	} `json:"user"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	if req.Email == "" || req.Password == "" || req.Name == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "email, name, and password are required"})
		return
	}

	if len(req.Password) < 8 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "password must be at least 8 characters"})
		return
	}

	user, err := h.Service.Register(req.Email, req.Name, req.Password)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	token, err := h.Service.Login(req.Email, req.Password)
	_ = token // login returns token but we already have user
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "registration succeeded but login failed"})
		return
	}

	resp := authResponse{Token: token}
	resp.User.ID = user.ID
	resp.User.Email = user.Email
	resp.User.Name = user.Name
	resp.User.AvatarURL = user.AvatarURL

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	if req.Email == "" || req.Password == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "email and password are required"})
		return
	}

	token, user, err := h.Service.Login(req.Email, req.Password)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	resp := authResponse{Token: token}
	resp.User.ID = user.ID
	resp.User.Email = user.Email
	resp.User.Name = user.Name
	resp.User.AvatarURL = user.AvatarURL

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	claims := GetUserFromContext(r.Context())
	if claims == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "not authenticated"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id": claims.UserID,
		"email":   claims.Email,
	})
}
```

- [ ] **Step 2: Verify build**

```bash
cd apps/server && go build ./...
```

Expected: compiles without errors.

- [ ] **Step 3: Commit**

```bash
git add apps/server/internal/auth/handler.go
git commit -m "feat(server): add auth HTTP handlers for register, login, and me endpoints"
```

---

### Task 8: Workspace service + handlers

**Files:**
- Create: `apps/server/internal/workspace/service.go`
- Create: `apps/server/internal/workspace/handler.go`

- [ ] **Step 1: Create workspace service**

`apps/server/internal/workspace/service.go`:

```go
package workspace

import (
	"errors"

	"github.com/bin-ke/my-notion/pkg/db"
	"gorm.io/gorm"
)

type Service struct {
	DB *gorm.DB
}

func NewService(database *gorm.DB) *Service {
	return &Service{DB: database}
}

func (s *Service) Create(name string, ownerID uint) (*db.Workspace, error) {
	ws := &db.Workspace{Name: name}

	tx := s.DB.Begin()

	if err := tx.Create(ws).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	member := &db.WorkspaceMember{
		WorkspaceID: ws.ID,
		UserID:      ownerID,
		Role:        "owner",
	}
	if err := tx.Create(member).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	tx.Commit()
	return ws, nil
}

func (s *Service) GetByID(id uint) (*db.Workspace, error) {
	var ws db.Workspace
	if err := s.DB.First(&ws, id).Error; err != nil {
		return nil, errors.New("workspace not found")
	}
	return &ws, nil
}

func (s *Service) ListByUser(userID uint) ([]db.Workspace, error) {
	var members []db.WorkspaceMember
	if err := s.DB.Preload("Workspace").Where("user_id = ?", userID).Find(&members).Error; err != nil {
		return nil, err
	}

	workspaces := make([]db.Workspace, len(members))
	for i, m := range members {
		workspaces[i] = m.Workspace
	}
	return workspaces, nil
}
```

- [ ] **Step 2: Create workspace handler**

`apps/server/internal/workspace/handler.go`:

```go
package workspace

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/bin-ke/my-notion/internal/auth"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	Service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{Service: service}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "not authenticated"})
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "name is required"})
		return
	}

	ws, err := h.Service.Create(req.Name, claims.UserID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to create workspace"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ws)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid workspace id"})
		return
	}

	ws, err := h.Service.GetByID(uint(id))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ws)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "not authenticated"})
		return
	}

	workspaces, err := h.Service.ListByUser(claims.UserID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to list workspaces"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workspaces)
}
```

- [ ] **Step 3: Fix import in handler.go — add db import**

Add to handler.go imports:
```go
import (
	// ...
	"github.com/bin-ke/my-notion/pkg/db"
)
```

Wait, `db.Workspace` is used in the `List` method. Check the current import actually references db package. Actually the `db` package is used via the service returning `db.Workspace`, but the handler doesn't directly reference `db` types — the `[]db.Workspace` is used for the empty slice fallback `workspaces = []db.Workspace{}`. Let me check...

Actually the handler code uses `[]db.Workspace{}` but the db import isn't in the imports above. Let me fix the handler to not need the import directly, or add it:

Actually `h.Service.ListByUser` returns `[]db.Workspace`, so we're fine — we just need to handle the nil case differently:

Replace `workspaces = []db.Workspace{}` with `workspaces = make([]interface{}, 0)` and return that... No, that's worse.

Better approach: just add the import. But the handler shouldn't need to know about db types. Let me instead just return the raw nil — the JSON encoder will marshal nil as `null`, which is fine. Or use a different pattern:

Actually the simplest fix: `workspaces = []struct{}{}` won't type-match. Let me just remove the nil guard and let it return `null` — it's semantically correct. The frontend can handle null for an empty list.

No wait, returning `null` vs `[]` is a real API design question. Best practice is return empty array. Let me just add the import properly.

Actually simplest approach: use `json.NewEncoder(w).Encode(workspaces)` as-is. `json.Marshal(nil)` returns `[]byte("null")`, which is fine for an empty list. Frontend check: `Array.isArray(data) ? data : []`.

Let me just leave the handler as-is but fix the nil guard to not need the db import:

Replace:
```go
if workspaces == nil {
    workspaces = []db.Workspace{}
}
```

With a different approach. Actually, I can make the service return an empty slice instead of nil. That's cleaner:

Let me update the service to do that.

OK I'm overcomplicating this. Let me just remove the nil check from the handler and update the service to return empty slice. Two edits, clean.

Let me fix the handler and service.

- [ ] **Step 3: Verify build**

```bash
cd apps/server && go build ./...
```

Expected: compiles without errors.

- [ ] **Step 4: Commit**

```bash
git add apps/server/internal/workspace/
git commit -m "feat(server): add workspace service and HTTP handlers"
```

---

### Task 9: Wire everything together in main.go

**Files:**
- Modify: `apps/server/cmd/api/main.go`

- [ ] **Step 1: Update main.go to register all routes**

```go
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
```

- [ ] **Step 2: Verify build + test endpoints**

```bash
cd apps/server && go build ./cmd/api
```

Start server:
```bash
cd apps/server && DATABASE_URL="postgres://notion:notion_dev@localhost:5432/my_notion?sslmode=disable" go run ./cmd/api &
```

Test:
```bash
# Register
curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","name":"Test User","password":"password123"}' | jq .

# Login
curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}' | jq .

# Create workspace (replace TOKEN from login response)
curl -s -X POST http://localhost:8080/api/v1/workspaces \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer TOKEN" \
  -d '{"name":"My Workspace"}' | jq .
```

Expected: All return JSON responses with 201/200 status codes.

- [ ] **Step 3: Commit**

```bash
git add apps/server/cmd/api/main.go
git commit -m "feat(server): wire auth and workspace routes into main.go"
```

---

### Task 10: Scaffold React frontend

**Files:**
- Create: `apps/web/package.json`
- Create: `apps/web/vite.config.ts`
- Create: `apps/web/tsconfig.json`
- Create: `apps/web/tsconfig.node.json`
- Create: `apps/web/tailwind.config.ts`
- Create: `apps/web/postcss.config.js`
- Create: `apps/web/index.html`
- Create: `apps/web/src/main.tsx`
- Create: `apps/web/src/App.tsx`
- Create: `apps/web/src/index.css`

- [ ] **Step 1: Create web package.json**

```bash
mkdir -p apps/web/src/lib apps/web/src/stores apps/web/src/pages apps/web/src/components
cd apps/web
```

`apps/web/package.json`:
```json
{
  "name": "web",
  "private": true,
  "version": "0.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc && vite build",
    "preview": "vite preview",
    "lint": "eslint . --ext ts,tsx"
  },
  "dependencies": {
    "react": "^18.3.1",
    "react-dom": "^18.3.1",
    "react-router-dom": "^6.23.1",
    "zustand": "^4.5.2"
  },
  "devDependencies": {
    "@types/react": "^18.3.3",
    "@types/react-dom": "^18.3.0",
    "@vitejs/plugin-react": "^4.3.0",
    "autoprefixer": "^10.4.19",
    "postcss": "^8.4.38",
    "tailwindcss": "^3.4.3",
    "typescript": "^5.4.5",
    "vite": "^5.2.12"
  }
}
```

- [ ] **Step 2: Create Vite config**

`apps/web/vite.config.ts`:
```typescript
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
    },
  },
});
```

- [ ] **Step 3: Create Tailwind + PostCSS config**

`apps/web/tailwind.config.ts`:
```typescript
import type { Config } from "tailwindcss";

export default {
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  theme: {
    extend: {},
  },
  plugins: [],
} satisfies Config;
```

`apps/web/postcss.config.js`:
```javascript
export default {
  plugins: {
    tailwindcss: {},
    autoprefixer: {},
  },
};
```

- [ ] **Step 4: Create tsconfig.json**

`apps/web/tsconfig.json`:
```json
{
  "compilerOptions": {
    "target": "ES2020",
    "useDefineForClassFields": true,
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "resolveJsonModule": true,
    "isolatedModules": true,
    "noEmit": true,
    "jsx": "react-jsx",
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true
  },
  "include": ["src"],
  "references": [{ "path": "./tsconfig.node.json" }]
}
```

`apps/web/tsconfig.node.json`:
```json
{
  "compilerOptions": {
    "composite": true,
    "skipLibCheck": true,
    "module": "ESNext",
    "moduleResolution": "bundler",
    "allowSyntheticDefaultImports": true
  },
  "include": ["vite.config.ts"]
}
```

- [ ] **Step 5: Create index.html**

`apps/web/index.html`:
```html
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>My Notion</title>
  </head>
  <body class="bg-white text-gray-900 antialiased">
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
```

- [ ] **Step 6: Create entry files**

`apps/web/src/index.css`:
```css
@tailwind base;
@tailwind components;
@tailwind utilities;
```

`apps/web/src/main.tsx`:
```tsx
import React from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import App from "./App";
import "./index.css";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <BrowserRouter>
      <App />
    </BrowserRouter>
  </React.StrictMode>
);
```

`apps/web/src/App.tsx`:
```tsx
import { Routes, Route, Navigate } from "react-router-dom";
import Login from "./pages/Login";
import Register from "./pages/Register";
import Workspace from "./pages/Workspace";
import ProtectedRoute from "./components/ProtectedRoute";

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route path="/register" element={<Register />} />
      <Route
        path="/workspace/:id"
        element={
          <ProtectedRoute>
            <Workspace />
          </ProtectedRoute>
        }
      />
      <Route path="*" element={<Navigate to="/login" replace />} />
    </Routes>
  );
}
```

- [ ] **Step 7: Install dependencies and verify dev server**

```bash
cd apps/web && pnpm install
pnpm dev
# Open http://localhost:5173 in browser — should see blank page (no content yet)
```

Expected: dev server starts without errors, page loads (shows nothing — routes go to Login page which doesn't exist yet).

- [ ] **Step 8: Commit**

```bash
git add apps/web/
git commit -m "feat(web): scaffold React frontend with Vite, Tailwind, and React Router"
```

---

### Task 11: API client + auth store

**Files:**
- Create: `apps/web/src/lib/api.ts`
- Create: `apps/web/src/stores/auth.ts`

- [ ] **Step 1: Create API client**

`apps/web/src/lib/api.ts`:

```typescript
const BASE_URL = "/api/v1";

class ApiError extends Error {
  status: number;
  constructor(message: string, status: number) {
    super(message);
    this.status = status;
  }
}

async function request<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<T> {
  const token = localStorage.getItem("token");
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options.headers as Record<string, string>),
  };

  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(`${BASE_URL}${endpoint}`, {
    ...options,
    headers,
  });

  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new ApiError(body.error || "request failed", res.status);
  }

  return res.json();
}

export const api = {
  get: <T>(endpoint: string) => request<T>(endpoint),
  post: <T>(endpoint: string, body: unknown) =>
    request<T>(endpoint, {
      method: "POST",
      body: JSON.stringify(body),
    }),
  patch: <T>(endpoint: string, body: unknown) =>
    request<T>(endpoint, {
      method: "PATCH",
      body: JSON.stringify(body),
    }),
  put: <T>(endpoint: string, body: unknown) =>
    request<T>(endpoint, {
      method: "PUT",
      body: JSON.stringify(body),
    }),
  delete: <T>(endpoint: string) =>
    request<T>(endpoint, { method: "DELETE" }),
};

export { ApiError };
```

- [ ] **Step 2: Create auth store**

`apps/web/src/stores/auth.ts`:

```typescript
import { create } from "zustand";
import { api } from "../lib/api";

interface User {
  id: number;
  email: string;
  name: string;
  avatar_url: string;
}

interface AuthState {
  user: User | null;
  token: string | null;
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, name: string, password: string) => Promise<void>;
  logout: () => void;
  loadFromStorage: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  token: null,
  loading: false,

  login: async (email: string, password: string) => {
    set({ loading: true });
    try {
      const data = await api.post<{ token: string; user: User }>(
        "/auth/login",
        { email, password }
      );
      localStorage.setItem("token", data.token);
      set({ user: data.user, token: data.token, loading: false });
    } catch (e) {
      set({ loading: false });
      throw e;
    }
  },

  register: async (email: string, name: string, password: string) => {
    set({ loading: true });
    try {
      const data = await api.post<{ token: string; user: User }>(
        "/auth/register",
        { email, name, password }
      );
      localStorage.setItem("token", data.token);
      set({ user: data.user, token: data.token, loading: false });
    } catch (e) {
      set({ loading: false });
      throw e;
    }
  },

  logout: () => {
    localStorage.removeItem("token");
    set({ user: null, token: null });
  },

  loadFromStorage: () => {
    const token = localStorage.getItem("token");
    if (token) {
      set({ token });
    }
  },
}));
```

- [ ] **Step 3: Verify build**

```bash
cd apps/web && pnpm build
```

Expected: builds without errors (warnings OK).

- [ ] **Step 4: Commit**

```bash
git add apps/web/src/lib/ apps/web/src/stores/
git commit -m "feat(web): add API client and auth Zustand store"
```

---

### Task 12: Login + Register pages

**Files:**
- Create: `apps/web/src/pages/Login.tsx`
- Create: `apps/web/src/pages/Register.tsx`
- Create: `apps/web/src/components/ProtectedRoute.tsx`

- [ ] **Step 1: Create Login page**

`apps/web/src/pages/Login.tsx`:

```tsx
import { useState, FormEvent } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useAuthStore } from "../stores/auth";

export default function Login() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const { login, loading } = useAuthStore();
  const navigate = useNavigate();

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError("");
    try {
      await login(email, password);
      navigate("/workspace/1"); // TODO: redirect to first workspace
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Login failed");
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50">
      <div className="w-full max-w-sm rounded-lg bg-white p-8 shadow-md">
        <h1 className="mb-6 text-center text-2xl font-semibold text-gray-900">
          Sign in to My Notion
        </h1>

        {error && (
          <div className="mb-4 rounded bg-red-50 p-3 text-sm text-red-600">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label htmlFor="email" className="block text-sm font-medium text-gray-700">
              Email
            </label>
            <input
              id="email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              className="mt-1 block w-full rounded border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          </div>

          <div>
            <label htmlFor="password" className="block text-sm font-medium text-gray-700">
              Password
            </label>
            <input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              className="mt-1 block w-full rounded border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
          >
            {loading ? "Signing in..." : "Sign in"}
          </button>
        </form>

        <p className="mt-4 text-center text-sm text-gray-600">
          Don't have an account?{" "}
          <Link to="/register" className="text-blue-600 hover:underline">
            Sign up
          </Link>
        </p>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Create Register page**

`apps/web/src/pages/Register.tsx`:

```tsx
import { useState, FormEvent } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useAuthStore } from "../stores/auth";

export default function Register() {
  const [email, setEmail] = useState("");
  const [name, setName] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const { register, loading } = useAuthStore();
  const navigate = useNavigate();

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError("");
    if (password.length < 8) {
      setError("Password must be at least 8 characters");
      return;
    }
    try {
      await register(email, name, password);
      navigate("/workspace/1");
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Registration failed");
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50">
      <div className="w-full max-w-sm rounded-lg bg-white p-8 shadow-md">
        <h1 className="mb-6 text-center text-2xl font-semibold text-gray-900">
          Create your account
        </h1>

        {error && (
          <div className="mb-4 rounded bg-red-50 p-3 text-sm text-red-600">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label htmlFor="name" className="block text-sm font-medium text-gray-700">
              Name
            </label>
            <input
              id="name"
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              className="mt-1 block w-full rounded border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          </div>

          <div>
            <label htmlFor="email" className="block text-sm font-medium text-gray-700">
              Email
            </label>
            <input
              id="email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              className="mt-1 block w-full rounded border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          </div>

          <div>
            <label htmlFor="password" className="block text-sm font-medium text-gray-700">
              Password
            </label>
            <input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              className="mt-1 block w-full rounded border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
          >
            {loading ? "Creating account..." : "Create account"}
          </button>
        </form>

        <p className="mt-4 text-center text-sm text-gray-600">
          Already have an account?{" "}
          <Link to="/login" className="text-blue-600 hover:underline">
            Sign in
          </Link>
        </p>
      </div>
    </div>
  );
}
```

- [ ] **Step 3: Create ProtectedRoute component**

`apps/web/src/components/ProtectedRoute.tsx`:

```tsx
import { Navigate } from "react-router-dom";
import { useAuthStore } from "../stores/auth";

export default function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const token = useAuthStore((s) => s.token) || localStorage.getItem("token");

  if (!token) {
    return <Navigate to="/login" replace />;
  }

  return <>{children}</>;
}
```

- [ ] **Step 4: Create Workspace placeholder page**

`apps/web/src/pages/Workspace.tsx`:

```tsx
import { useAuthStore } from "../stores/auth";

export default function Workspace() {
  const user = useAuthStore((s) => s.user);
  const logout = useAuthStore((s) => s.logout);

  return (
    <div className="flex min-h-screen bg-white">
      {/* Sidebar placeholder */}
      <aside className="flex w-60 flex-col border-r border-gray-200 bg-gray-50 p-4">
        <div className="mb-4 text-sm font-semibold text-gray-900">
          {user?.name || "User"}'s Notion
        </div>
        <div className="flex-1" />
        <button
          onClick={logout}
          className="text-left text-sm text-gray-500 hover:text-gray-700"
        >
          Sign out
        </button>
      </aside>

      {/* Main content — empty state */}
      <main className="flex flex-1 items-center justify-center">
        <div className="text-center">
          <h2 className="text-xl font-medium text-gray-700">Welcome to My Notion</h2>
          <p className="mt-2 text-sm text-gray-500">
            M0 complete. Editor and pages coming in M1.
          </p>
        </div>
      </main>
    </div>
  );
}
```

- [ ] **Step 5: Verify build**

```bash
cd apps/web && pnpm build
```

Expected: builds without errors.

- [ ] **Step 6: Commit**

```bash
git add apps/web/src/pages/ apps/web/src/components/
git commit -m "feat(web): add login, register pages, protected route, and workspace shell"
```

---

### Task 13: CI/CD (GitHub Actions)

**Files:**
- Create: `.github/workflows/ci.yml`

- [ ] **Step 1: Create CI workflow**

`.github/workflows/ci.yml`:

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  backend:
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:16-alpine
        env:
          POSTGRES_USER: notion
          POSTGRES_PASSWORD: notion_ci
          POSTGRES_DB: my_notion_test
        ports:
          - 5432:5432
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: "1.22"

      - name: Build
        working-directory: apps/server
        run: go build ./...

      - name: Test
        working-directory: apps/server
        run: go vet ./...

  frontend:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4

      - uses: pnpm/action-setup@v3
        with:
          version: 8

      - uses: actions/setup-node@v4
        with:
          node-version: 20
          cache: "pnpm"

      - name: Install dependencies
        run: pnpm install --frozen-lockfile

      - name: Build
        working-directory: apps/web
        run: pnpm build
```

- [ ] **Step 2: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: add GitHub Actions workflow for backend and frontend"
```

---

### Task 14: E2E smoke test

**Files:**
- Stub: test stub (full E2E comes in M1)

- [ ] **Step 1: Verify full flow manually**

```bash
# Start all services
docker compose -f docker/docker-compose.yml up -d

# Start backend
cd apps/server && DATABASE_URL="postgres://notion:notion_dev@localhost:5432/my_notion?sslmode=disable" go run ./cmd/api &

# Start frontend
cd apps/web && pnpm dev &

# Smoke test
# 1. Open http://localhost:5173 → should redirect to /login
# 2. Click "Sign up" → fill form → submit
# 3. Should redirect to workspace page with sidebar and "Sign out" button
# 4. Click "Sign out" → should redirect back to login
# 5. Sign in with created account → should work
```

Expected: Full register → login → workspace → logout flow works.

- [ ] **Step 2: Commit any remaining files**

```bash
git status
git add -A
git commit -m "chore: M0 complete — infrastructure, auth, workspace"
```

---

## Verification Checklist

After all tasks complete, verify:

- [ ] `docker compose -f docker/docker-compose.yml up -d` starts all services
- [ ] `cd apps/server && go run ./cmd/api` starts API on :8080
- [ ] `cd apps/web && pnpm dev` starts frontend on :5173
- [ ] `POST /api/v1/auth/register` creates user and returns JWT
- [ ] `POST /api/v1/auth/login` authenticates and returns JWT
- [ ] `GET /api/v1/auth/me` returns current user when authenticated
- [ ] `POST /api/v1/workspaces` creates workspace (authenticated)
- [ ] `GET /api/v1/workspaces` lists user's workspaces
- [ ] `GET /api/v1/workspaces/:id` returns workspace by ID
- [ ] Frontend: register → auto-login → workspace page → logout → login works
- [ ] Frontend: unauthenticated access redirects to /login
- [ ] CI passes: `go build ./...`, `go vet ./...`, `pnpm build`
