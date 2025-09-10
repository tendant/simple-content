-- +goose Up
SET search_path TO content;

-- Add a dedicated 'variant' column to derived_content for clarity. Keep existing
-- 'derivation_type' for smooth migration and backward compatibility.
ALTER TABLE derived_content
    ADD COLUMN IF NOT EXISTS variant VARCHAR(100);

-- Backfill variant from derivation_type and normalize to lowercase.
UPDATE derived_content
SET variant = lower(derivation_type)
WHERE variant IS NULL OR variant = '';

-- Optional: you may enforce NOT NULL in a controlled rollout later.
-- ALTER TABLE derived_content ALTER COLUMN variant SET NOT NULL;

-- +goose Down
SET search_path TO content;
ALTER TABLE derived_content
    DROP COLUMN IF EXISTS variant;

