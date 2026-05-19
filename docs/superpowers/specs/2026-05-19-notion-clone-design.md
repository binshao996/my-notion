# Notion Clone — System Design

## Project Goal

1:1 clone of Notion Web, covering all features across 6 milestones.

## Confirmed Tech Stack

| Layer | Choice |
|-------|--------|
| Frontend framework | React 18 + TypeScript + Vite |
| Editor engine | Tiptap (ProseMirror) + custom block shell |
| State management | Zustand |
| Styling | Tailwind CSS |
| Backend language | Go |
| ORM | GORM |
| API style | REST (base: `/api/v1`) |
| Database | PostgreSQL + Redis |
| File storage | S3-compatible (MinIO dev / AWS prod) |
| Monorepo | pnpm workspaces |
| Search (M5) | OpenSearch |
| Collaboration (M4) | Yjs + WebSocket |
| AI (M6) | LLM Gateway + RAG (Hybrid search + Milvus) |
| Queues | Redis + worker processes |

## Architecture: Modular Monolith + Async Workers

Single Go API server. Worker binary reuses the same `internal/` packages for background tasks. One PostgreSQL database, shared across internal packages. Extract services only when scale demands it.

## Monorepo Structure

```
my-notion/
├── apps/
│   ├── web/                    # React frontend
│   │   ├── src/
│   │   │   ├── components/     # Shared UI components
│   │   │   ├── features/       # Feature modules
│   │   │   │   ├── editor/     # Block editor
│   │   │   │   ├── sidebar/    # Page tree
│   │   │   │   ├── database/   # Table/Board/Calendar
│   │   │   │   ├── search/     # Global search
│   │   │   │   └── ai/         # AI panel
│   │   │   ├── hooks/          # Shared hooks
│   │   │   ├── stores/         # Zustand stores
│   │   │   ├── lib/            # API client
│   │   │   └── types/          # Shared TS types
│   │   └── tailwind.config.ts
│   │
│   └── server/                 # Go backend
│       ├── cmd/
│       │   ├── api/            # API server entry
│       │   └── worker/         # Background worker entry
│       ├── internal/
│       │   ├── auth/
│       │   ├── page/
│       │   ├── block/
│       │   ├── database/
│       │   ├── search/
│       │   ├── file/
│       │   ├── permission/
│       │   ├── notification/
│       │   ├── collaboration/
│       │   └── ai/
│       ├── pkg/
│       │   ├── db/
│       │   ├── queue/
│       │   ├── storage/
│       │   └── middleware/
│       └── migrations/
│
├── packages/
│   └── shared-types/
│
├── docker/
│   ├── docker-compose.yml
│   └── Dockerfile
│
└── pnpm-workspace.yaml
```

### Rules

- `internal/` packages are independent — no cross-imports between siblings
- Worker binary reuses all `internal/` packages, different `cmd/` entry
- Frontend `features/` map 1:1 to backend `internal/` packages
- Each package owns its own DB migrations

## Milestone Roadmap

| Phase | Scope | Est. |
|-------|-------|------|
| M0 | Project init, CI/CD, DB schema, auth, workspace | 2-4w |
| M1 | Block editor, page tree, sidebar, / command, drag-drop, file upload | 4-8w |
| M2 | Database (Table/Board/Calendar), properties, views, filter/sort, record pages | 6-12w |
| M2b | Relation, rollup, formula (deferred from M2 to keep it shippable) | 4-6w |
| M3 | Permissions, sharing, comments, notifications | 6-12w |
| M4 | Real-time collaboration (Yjs CRDT), presence | 8-16w |
| M5 | Full-text search (OpenSearch), permission-aware indexing, performance | 6-12w |
| M6 | AI writing, RAG Q&A, database autofill, LLMOps/evals | 6-12w |

## Core Data Models

### M0: Workspaces & Users

```sql
users: id, email, name, avatar_url, password_hash, created_at
workspaces: id, name, icon, created_at
workspace_members: workspace_id, user_id, role(owner|member), joined_at
```

### M1: Pages & Block Tree

```sql
pages: id, workspace_id, parent_page_id, title, icon, cover,
       created_by, created_at, updated_at, archived

blocks: id, page_id, parent_block_id, type(enum), position(text),
        props(jsonb), created_at, updated_at
```

Block types (MVP): paragraph, heading1-3, bulleted_list_item, numbered_list_item, todo, toggle, divider, callout, quote, code, image, file, bookmark, equation, table_of_contents, columns.

`position` uses fractional indexing for O(1) insert between siblings without reordering. `props` is JSONB — each block type stores its specific fields (text content, checked state, URL, language, etc.).

### M2: Database Engine

```sql
databases: id, page_id(container), name, created_at
properties: id, database_id, name, type(enum), config(jsonb), position
records: id, database_id, page_id(the record's page), position
property_values: record_id, property_id, value(jsonb)
views: id, database_id, name, type(enum), config(jsonb), position
```

Property types: title, text, number, select, multi_select, status, date, person, files, checkbox, url, email, phone, relation, rollup, formula, created_time, last_edited_time, created_by, last_edited_by.

View types: table, board, timeline, calendar, list, gallery.

View config (JSONB): `{ filters, sorts, groups, hidden_properties, layout }`.

Records are pages (`records.page_id` references `pages.id`). Record page has properties at top, blocks below.

### M3: Permissions & Sharing

```sql
page_permissions: page_id, subject_type(user|group|link), subject_id, role
share_tokens: token, page_id, role, expires_at, created_at, created_by
comments: id, page_id, block_id, author_id, content(jsonb), resolved, parent_id
notifications: id, user_id, type, actor_id, target_page_id, read, created_at
```

### Design principles

- All JSONB columns are flexible for evolution; can add typed columns when patterns stabilize
- Fractional-indexed positions avoid mass updates on insert/move
- Views are pure presentation — changing view config never changes data
- Permissions start simple (page-level), designed for future row-level extension

## API Design

### M0 — Auth & Workspace

```
POST /api/v1/auth/register        email, password → JWT
POST /api/v1/auth/login           email, password → JWT
GET  /api/v1/workspaces/:id        workspace info
POST /api/v1/workspaces            create workspace
```

### M1 — Pages & Blocks

```
GET    /api/v1/pages/:id               page meta + first N blocks
POST   /api/v1/pages                   create page
PATCH  /api/v1/pages/:id               rename, move, archive
GET    /api/v1/pages/:id/children      lazy-load subtree
GET    /api/v1/blocks/:pageId          query by page ?offset=&limit=
PUT    /api/v1/blocks/:pageId          batch save (MVP: whole page, 500ms debounce)
POST   /api/v1/blocks/:pageId/ops      incremental operations (for collaboration later)
POST   /api/v1/files/upload-url        get pre-signed S3 URL
```

### M2 — Databases

```
POST   /api/v1/databases                        create database
GET    /api/v1/databases/:id                    schema + views + first page records
POST   /api/v1/databases/:id/properties         add property
PATCH  /api/v1/properties/:id                   rename / reconfigure
DELETE /api/v1/properties/:id                   delete (cascades)
POST   /api/v1/databases/:id/records            create record
PATCH  /api/v1/records/:id                      update property values
GET    /api/v1/databases/:id/views/:viewId/records   query with filter/sort/group & pagination
POST   /api/v1/databases/:id/views              create view
PATCH  /api/v1/views/:id                        update view config
```

### Key decisions

- MVP uses whole-page save (`PUT /blocks/:pageId`) with 500ms client-side debounce
- Migration path to incremental ops (`POST /blocks/:pageId/ops`) built into the schema from day 1
- View config translates to parameterized SQL at query time — no raw string building

## Block Editor Architecture

Two-layer design:

**BlockShell (custom)** — manages block tree, nesting, drag-drop, menu. Each block has:
- Drag handle (hover affordance, ⋮⋮ icon)
- Block menu (turn into, duplicate, delete, copy link, comment)
- Recursive children rendering

**TiptapEditor (ProseMirror)** — one instance per block, handles:
- Inline marks: bold, italic, underline, strikethrough, code, link, color
- Mentions: @page, @person, @date
- Inline code, equation

### Why one Tiptap instance per block

Avoids selection/tree entanglement. BlockShell manages the tree; Tiptap manages inline text within each block. Arrow key navigation between blocks handled by custom keyboard handler on BlockShell.

### Keyboard shortcuts

| Key | Action |
|-----|--------|
| Enter | New block of same type below |
| Shift+Enter | Soft break within text |
| Backspace (empty) | Delete block, focus previous |
| Tab / Shift+Tab | Indent / outdent |
| `/` | Command palette |
| Ctrl+Shift+ArrowUp/Down | Move block |
| `@` | Mention picker (page/person/date) |

### Command palette (`/` menu)

Categories: Basic blocks, Media, Database, Advanced. Type-to-filter, Enter to insert, Esc to dismiss.

### Selection model

- Text selection: ProseMirror native within block
- Block multi-select: custom Shift+Click or drag-select handles
- Cursor position: `{ blockId, offset }` for paste, AI insert, drag-drop
- IME: compositionstart → mark composing → skip certain mutations → compositionend → flush

## Database View Engine

### Principle

Views only change presentation. Backend translates view config (filters/sorts/groups) into parameterized SQL. Frontend renders 6 layouts from same record stream.

### View types

- **Table**: `react-virtual` for row+column virtualization, inline cell editors per property type
- **Board**: grouped by select/status property, drag cards between columns = update property value
- **Calendar**: date-grid layout, drag to move dates, resize for date range
- **Timeline**: horizontal Gantt with zoom levels (week/month/quarter)
- **Gallery**: card grid with cover image previews
- **List**: minimal row layout optimized for reading

### Filter engine

```
filter config:
  { property, operator, value }
  compound: { and: [...] } | { or: [...] }

SQL translation:
  property → column lookup → parameterized WHERE clause
  GROUP BY → appended to query with aggregate functions
```

No raw SQL string building. All values parameterized. Filter tree parsed and translated safely.

### Property types (M2 core)

title, text, number, select, multi_select, status, date, person, files, checkbox, url, email, phone.

### Deferred to M2b

relation, rollup, formula — complex, don't block M2 release.

## M3-M6 Design Summary

### M3: Permissions & Sharing

Page-level permission model with parent inheritance. Walk up `parent_page_id` chain, first permission hit wins. Redis cache (TTL 60s) invalidated on permission change. Share tokens for public link access. Comment threads: threaded, resolvable, @mention triggers notification.

### M4: Real-time Collaboration

Yjs CRDT for inline text + custom Yjs type for block structure. WebSocket channels: `/ws/page/:id` for edits, `/ws/presence` for cursors. Server: Yrs (Rust Yjs port) for sync protocol. Periodic snapshots + incremental op persistence. Reconnect sends missing ops from last known version.

### M5: Search

OpenSearch: index page titles + block text + database property values. ACL fields embedded per document. Async indexing via change events → queue → worker. Query-time permission filter. Incremental updates on block save.

### M6: AI

AI Gateway (Go middleware): rate limiting, model routing, prompt injection guard, cost logging.

- Writing/editing: context from selection/block/page → structured block output
- RAG Q&A: chunk by block → Milvus vectors + OpenSearch keyword → hybrid retrieval → permission filter → LLM → cited answer
- Database autofill: queue per batch (Redis streams or Asynq) → worker reads record → LLM writes property → progress poll

## Testing Strategy

- Unit: Go table-driven tests per package, Vitest for frontend
- Integration: API endpoint tests with test DB, editor interaction tests (Playwright)
- E2E: Playwright for critical paths (create page → edit → share)
- Performance: benchmark 1000-block pages, 10k-record database tables
- AI evals (M6): offline eval harness for hallucination rate, citation accuracy, answer correctness

## DevOps

- `docker-compose up` for local dev (Postgres + Redis + MinIO)
- CI: lint + test + build on every PR
- Deploy: single Go binary + static frontend on CDN
- Staging environment for integration testing before M1+ releases
