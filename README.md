# my-notion

1:1 Notion clone. Go backend (REST API) + React frontend (Tiptap editor).

## Prerequisites

- **Go** 1.23+
- **Node** 18+ + **pnpm** 9+
- **Docker** + docker-compose

## Quick Start

### 1. Infrastructure

```bash
cd docker
docker-compose up -d
```

Starts:
| Service | Port | Purpose |
|----------|------|---------|
| PostgreSQL 16 | 5432 | Primary database |
| Redis 7 | 6379 | Permission cache + search queue |
| OpenSearch 2.18 | 9200 | Full-text search |
| MinIO | 9000, 9001 | S3-compatible file storage |

### 2. Backend

```bash
cd apps/server

# Install Go dependencies (if first time)
go mod tidy

# Run API server
go run ./cmd/api
```

Server starts on **http://localhost:8080**.

```bash
# (Optional) Run background worker for async search indexing
go run ./cmd/worker
```

**Environment variables** (all have defaults for local dev):

| Variable | Default |
|----------|---------|
| `DATABASE_URL` | `postgres://notion:notion_dev@localhost:5432/my_notion?sslmode=disable` |
| `REDIS_URL` | `redis://localhost:6379` |
| `PORT` | `8080` |
| `OPENSEARCH_URL` | `http://localhost:9200` |
| `S3_ENDPOINT` | `localhost:9000` |
| `S3_ACCESS_KEY` | `minioadmin` |
| `S3_SECRET_KEY` | `minioadmin` |
| `S3_BUCKET` | `my-notion` |

### 3. Frontend

```bash
cd apps/web

# Install dependencies (if first time)
pnpm install

# Start dev server
pnpm dev
```

App at **http://localhost:5173**. API calls proxied to `localhost:8080` via Vite.

### 4. OpenSearch index setup (one-time)

After first start, create search indices:

```bash
curl -X POST http://localhost:8080/api/v1/search/reindex \
  -H "Authorization: Bearer <your-jwt-token>"
```

## Useful Commands

```bash
# Backend
cd apps/server
go build ./cmd/api          # build API binary
go vet ./...                # static analysis
go test ./...               # run tests

# Frontend
cd apps/web
pnpm build                  # production build
pnpm test                   # run vitest
pnpm exec tsc --noEmit      # type-check

# Infrastructure
cd docker
docker-compose down         # stop all services
docker-compose down -v      # stop + delete data volumes
```

## Architecture

```
apps/
├── web/          React 18 + TypeScript + Vite + Tailwind
└── server/       Go REST API (chi router, GORM + PostgreSQL JSONB)

apps/server/
├── cmd/api/      API server entry
├── cmd/worker/   Background worker (search indexing)
├── internal/     Service packages (auth, page, block, database, search, etc.)
└── pkg/          Shared packages (db, redis, queue, middleware)

apps/web/src/
├── features/     Feature modules (editor, database, sidebar, search, collaboration, etc.)
├── stores/       Zustand stores
└── lib/          API client
```

## Milestones

| Phase | Scope | Status |
|-------|-------|--------|
| M0 | Auth, workspace, DB schema | Done |
| M1 | Block editor, sidebar, file upload | Done |
| M2 | Database engine (table/board/calendar) | Done |
| M2b | Relation, rollup, formula, timeline | Done |
| M3 | Permissions, sharing, comments, notifications | Done |
| M4 | Real-time collaboration (Yjs + WebSocket) | Done |
| M5 | Full-text search (OpenSearch) | Done |
| M6 | AI writing, RAG Q&A | Future |
