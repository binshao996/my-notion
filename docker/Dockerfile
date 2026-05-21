# .dockerignore patterns:
#   node_modules
#   .git
#   *.md
#   .env*
#   docker/
#   apps/server/.modcache
#   apps/web/node_modules
#   **/*_test.go
#   **/testdata

# ============================================================
# Stage 1: Frontend build
# ============================================================
FROM node:20-alpine AS frontend

RUN corepack enable && corepack prepare pnpm@latest --activate

WORKDIR /src

# pnpm workspace config + lockfile (needed for install resolution)
COPY pnpm-workspace.yaml pnpm-lock.yaml package.json ./
COPY apps/web/package.json apps/web/

RUN pnpm install --frozen-lockfile --filter web

COPY apps/web/ apps/web/

RUN pnpm --filter web build && rm -rf apps/web/node_modules

# ============================================================
# Stage 2: Go build
# ============================================================
FROM golang:1.23-alpine AS builder

WORKDIR /src

COPY apps/server/go.mod apps/server/go.sum ./
RUN go mod download

COPY apps/server/cmd ./cmd
COPY apps/server/internal ./internal
COPY apps/server/pkg ./pkg

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/api ./cmd/api
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/worker ./cmd/worker

# ============================================================
# Stage 3: Final image
# ============================================================
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /out/api /usr/local/bin/api
COPY --from=builder /out/worker /usr/local/bin/worker
COPY --from=frontend /src/apps/web/dist /var/www/html

ENV PORT=8080

EXPOSE 8080

CMD ["api"]
