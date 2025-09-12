-- Simple Content Database Schema
-- This file contains the PostgreSQL schema for the simple-content library

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Content table: stores logical content entities
CREATE TABLE IF NOT EXISTS content (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    owner_id UUID NOT NULL,
    owner_type VARCHAR(50),
    name VARCHAR(500),
    description TEXT,
    document_type VARCHAR(100),
    status VARCHAR(50) NOT NULL DEFAULT 'created',
    derivation_type VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE NULL
);

-- Content metadata table: stores metadata for content
CREATE TABLE IF NOT EXISTS content_metadata (
    content_id UUID PRIMARY KEY REFERENCES content(id) ON DELETE CASCADE,
    tags TEXT[], -- PostgreSQL array of strings
    file_size BIGINT,
    file_name VARCHAR(500),
    mime_type VARCHAR(100),
    checksum VARCHAR(100),
    checksum_algorithm VARCHAR(50),
    metadata JSONB, -- PostgreSQL JSON for flexible metadata
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Object table: stores physical objects in storage backends
CREATE TABLE IF NOT EXISTS object (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    content_id UUID NOT NULL REFERENCES content(id) ON DELETE CASCADE,
    storage_backend_name VARCHAR(100) NOT NULL,
    storage_class VARCHAR(100),
    object_key VARCHAR(1000) NOT NULL,
    file_name VARCHAR(500),
    version INTEGER NOT NULL DEFAULT 1,
    object_type VARCHAR(100),
    status VARCHAR(50) NOT NULL DEFAULT 'created',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE NULL,
    
    -- Unique constraint on storage backend and object key
    UNIQUE(storage_backend_name, object_key)
);

-- Object metadata table: stores metadata about objects
CREATE TABLE IF NOT EXISTS object_metadata (
    object_id UUID PRIMARY KEY REFERENCES object(id) ON DELETE CASCADE,
    size_bytes BIGINT,
    mime_type VARCHAR(100),
    etag VARCHAR(100),
    metadata JSONB, -- PostgreSQL JSON for flexible metadata
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS content_derived (
    parent_id UUID NOT NULL REFERENCES content(id) ON DELETE CASCADE,
    content_id UUID NOT NULL REFERENCES content(id) ON DELETE CASCADE,
    variant VARCHAR(100) NOT NULL,
    derivation_params JSONB,
    processing_metadata JSONB,
    status VARCHAR(50) NOT NULL DEFAULT 'created',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE NULL,
    
    PRIMARY KEY (parent_id, content_id)
);


-- Indexes for better query performance

-- Content indexes
CREATE INDEX IF NOT EXISTS idx_content_owner_tenant ON content(owner_id, tenant_id);
CREATE INDEX IF NOT EXISTS idx_content_status ON content(status);
CREATE INDEX IF NOT EXISTS idx_content_created_at ON content(created_at);
CREATE INDEX IF NOT EXISTS idx_content_derivation_type ON content(derivation_type);

-- Object indexes
CREATE INDEX IF NOT EXISTS idx_object_content_id ON object(content_id);
CREATE INDEX IF NOT EXISTS idx_object_storage_backend ON object(storage_backend_name);
CREATE INDEX IF NOT EXISTS idx_object_status ON object(status);
CREATE INDEX IF NOT EXISTS idx_object_created_at ON object(created_at);

-- Derived content indexes
CREATE INDEX IF NOT EXISTS idx_content_derived_parent ON content_derived(parent_id);
CREATE INDEX IF NOT EXISTS idx_content_derived_variant ON content_derived(variant);


-- Functions for automatic timestamp updates
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers for automatic timestamp updates
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
