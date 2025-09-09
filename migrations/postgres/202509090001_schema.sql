-- +goose Up
-- Create dedicated schema for content-related tables (customize as needed)
CREATE SCHEMA IF NOT EXISTS content;

-- +goose Down
DROP SCHEMA IF EXISTS content CASCADE;

