-- +goose Up
-- Migrations run in whichever schema is selected by the connection search_path.

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Content table
CREATE TABLE IF NOT EXISTS content (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID,
    owner_id UUID,
    owner_type VARCHAR(32),
    name VARCHAR(255),
    description TEXT,
    document_type VARCHAR(100),
    status VARCHAR(32) NOT NULL DEFAULT 'created',
    derivation_type VARCHAR(32),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    deleted_at TIMESTAMP WITH TIME ZONE NULL
);

-- Content metadata table
CREATE TABLE IF NOT EXISTS content_metadata (
    content_id UUID PRIMARY KEY REFERENCES content(id) ON DELETE CASCADE,
    tags TEXT[],
    file_size BIGINT,
    file_name VARCHAR(255),
    mime_type VARCHAR(100),
    checksum VARCHAR(64),
    checksum_algorithm VARCHAR(32),
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc')
);

-- Object table
CREATE TABLE IF NOT EXISTS object (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    content_id UUID NOT NULL REFERENCES content(id) ON DELETE CASCADE,
    storage_backend_name VARCHAR(64) NOT NULL,
    storage_class VARCHAR(32),
    object_key VARCHAR(1024) NOT NULL,
    file_name VARCHAR(255),
    version INTEGER NOT NULL DEFAULT 1,
    object_type VARCHAR(32),
    status VARCHAR(32) NOT NULL DEFAULT 'created',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    deleted_at TIMESTAMP WITH TIME ZONE NULL,
    UNIQUE(storage_backend_name, object_key)
);

-- Object metadata table
CREATE TABLE IF NOT EXISTS object_metadata (
    object_id UUID PRIMARY KEY REFERENCES object(id) ON DELETE CASCADE,
    size_bytes BIGINT,
    mime_type VARCHAR(100),
    etag VARCHAR(128),
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc')
);

CREATE TABLE IF NOT EXISTS content_derived (
    parent_id UUID NOT NULL REFERENCES content(id) ON DELETE CASCADE,
    content_id UUID NOT NULL REFERENCES content(id) ON DELETE CASCADE,
    variant VARCHAR(32) NOT NULL,
    derivation_type VARCHAR(32) NOT NULL,
    derivation_params JSONB,
    processing_metadata JSONB,
    status VARCHAR(32) NOT NULL DEFAULT 'created',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc'),
    deleted_at TIMESTAMP WITH TIME ZONE NULL,
    PRIMARY KEY (parent_id, content_id)
);


-- Indexes
CREATE INDEX IF NOT EXISTS idx_content_owner_tenant ON content(owner_id, tenant_id);
CREATE INDEX IF NOT EXISTS idx_content_status ON content(status);
CREATE INDEX IF NOT EXISTS idx_content_created_at ON content(created_at);
CREATE INDEX IF NOT EXISTS idx_content_derivation_type ON content(derivation_type);

CREATE INDEX IF NOT EXISTS idx_object_content_id ON object(content_id);
CREATE INDEX IF NOT EXISTS idx_object_storage_backend ON object(storage_backend_name);
CREATE INDEX IF NOT EXISTS idx_object_status ON object(status);
CREATE INDEX IF NOT EXISTS idx_object_created_at ON object(created_at);

CREATE INDEX IF NOT EXISTS idx_content_derived_parent ON content_derived(parent_id);
CREATE INDEX IF NOT EXISTS idx_content_derived_variant ON content_derived(variant);


-- Triggers to maintain updated_at
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';
-- +goose StatementEnd

CREATE TRIGGER update_content_updated_at BEFORE UPDATE ON content
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_content_metadata_updated_at BEFORE UPDATE ON content_metadata
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_object_updated_at BEFORE UPDATE ON object
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_object_metadata_updated_at BEFORE UPDATE ON object_metadata
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_content_derived_updated_at BEFORE UPDATE ON content_derived
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- +goose Down
DROP TRIGGER IF EXISTS update_content_derived_updated_at ON content_derived;
DROP TRIGGER IF EXISTS update_object_metadata_updated_at ON object_metadata;
DROP TRIGGER IF EXISTS update_object_updated_at ON object;
DROP TRIGGER IF EXISTS update_content_metadata_updated_at ON content_metadata;
DROP TRIGGER IF EXISTS update_content_updated_at ON content;
DROP TABLE IF EXISTS content_derived;
DROP TABLE IF EXISTS object_metadata;
DROP TABLE IF EXISTS object;
DROP TABLE IF EXISTS content_metadata;
DROP TABLE IF EXISTS content;
