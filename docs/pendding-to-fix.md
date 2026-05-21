# Pending Gaps to Fix

Gaps found comparing current implementation against `docs/superpowers/specs/2026-05-19-notion-clone-design.md`.

## Infrastructure

### Gap 1: Dockerfile missing
- **Spec:** `docker/Dockerfile` for production Go binary + static frontend
- **Current:** Only `docker/docker-compose.yml` exists
- **Fix:** Create `docker/Dockerfile` with multi-stage build (Go builder + final image with binary + static files)

### Gap 2: packages/shared-types/ missing
- **Spec:** Monorepo structure includes `packages/shared-types/` for shared TS types
- **Current:** Directory doesn't exist. Types live inline in `apps/web/src/types/`
- **Fix:** Create `packages/shared-types/` with `package.json`, extract shared type definitions (database, search, AI types)

### Gap 3: No e2e tests
- **Spec:** "Playwright for critical paths (create page -> edit -> share)"
- **Current:** Zero e2e tests
- **Fix:** Add Playwright config + basic e2e test covering: login -> create page -> edit block -> share page

### Gap 4: No benchmarks
- **Spec:** "benchmark 1000-block pages, 10k-record database tables"
- **Current:** Zero benchmark tests
- **Fix:** Add Go benchmark tests for block loading (1000 blocks), database record loading (10k records)

## Features

### Gap 5: Files property editor
- **Spec:** `files` is an editable property type (M2 property types list)
- **Current:** `CellEditor.tsx:385` marks `files` as read-only with comment "files: read-only for now"
- **Fix:** Implement `FilesCellEditor` component — upload button, file list display, remove file. Similar pattern to person editor. Use existing file upload API.

### Gap 6: Block drag-drop reorder
- **Spec:** Drag handle on blocks, drag to reorder (M1 block editor)
- **Current:** `BlockShell.tsx` has drag handles and `onDragStart` but no `onDrop`/`onDragEnd` that calls a reorder API. Visual placeholder only.
- **Fix:** Add `onDrop` handler in BlockShell that computes target position, calls `moveBlock` action in useEditorStore, updates block positions with fractional indexing.

### Gap 7: Cover image UI
- **Spec:** `pages` table has `cover` column
- **Current:** `db.Page.Cover` field exists in DB model. No UI to set/change page cover image.
- **Fix:** Add cover image area at top of PageView — hover shows "Add cover" button, click opens file picker, uploads image, saves cover URL to page.

### Gap 8: Milvus vector DB for RAG
- **Spec:** "RAG (Hybrid search + Milvus)" for AI RAG service
- **Current:** `embedding.go` uses simple TF-IDF keyword vectorizer with cosine similarity. No vector database.
- **Fix:** Add Milvus to docker-compose, add Go Milvus client, modify RAG service to use Milvus for vector search + OpenSearch for keyword search (hybrid retrieval).

### Gap 9: AI features uncommitted
- **Current:** 20 untracked AI files + 3 modified files (main.go, App.tsx, PageView.tsx) from M3/M6 implementation. All tests pass, builds clean.
- **Fix:** Stage and commit all AI feature files with appropriate commit message.
