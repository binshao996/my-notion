-- M1: Pages and Blocks

CREATE TABLE IF NOT EXISTS pages (
    id BIGSERIAL PRIMARY KEY,
    workspace_id BIGINT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    parent_page_id BIGINT REFERENCES pages(id) ON DELETE SET NULL,
    title VARCHAR(500) NOT NULL DEFAULT '',
    icon VARCHAR(255) DEFAULT '',
    cover TEXT DEFAULT '',
    created_by BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    archived BOOLEAN DEFAULT FALSE
);

CREATE INDEX idx_pages_workspace ON pages(workspace_id);
CREATE INDEX idx_pages_parent ON pages(parent_page_id) WHERE parent_page_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS blocks (
    id BIGSERIAL PRIMARY KEY,
    page_id BIGINT NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    parent_block_id BIGINT REFERENCES blocks(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    position VARCHAR(50) NOT NULL,
    props JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_blocks_page ON blocks(page_id);
CREATE INDEX idx_blocks_parent ON blocks(parent_block_id) WHERE parent_block_id IS NOT NULL;
