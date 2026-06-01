-- NomadOS File Schema
CREATE TABLE files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    filename VARCHAR(1024) NOT NULL,
    size BIGINT DEFAULT 0,
    content_type VARCHAR(255),
    encrypted_path VARCHAR(1024),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_files_workspace_id ON files(workspace_id);