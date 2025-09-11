-- Set your target schema if needed
-- SET search_path TO content;

-- 1) Rename relationship table
ALTER TABLE content.derived_content RENAME TO content_derived;

-- 2) Rename relationship column to variant
ALTER TABLE content.content_derived RENAME COLUMN derivation_type TO variant;

-- 3) Add soft-delete columns if missing
ALTER TABLE content.content
  ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ NULL;

ALTER TABLE content.object
  ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ NULL;

ALTER TABLE content.content_derived
  ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ NULL;

ALTER TABLE content.object_preview
  ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ NULL;

-- 4) Helpful indexes
CREATE INDEX IF NOT EXISTS idx_content_derived_parent
  ON content.content_derived(parent_id);

CREATE INDEX IF NOT EXISTS idx_content_derived_variant
  ON content.content_derived(variant);

