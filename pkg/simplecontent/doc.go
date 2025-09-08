// Package simplecontent provides a reusable library for content and object
// management with pluggable repository and blob storage backends.
//
// It exposes a single Service interface that orchestrates creation of content
// and objects, object upload/download, metadata management, and optional
// event/preview integrations. Implementations of repositories (e.g., memory,
// Postgres) and blob stores (e.g., memory, filesystem, S3) are provided under
// subpackages.
//
// Metadata Strategy
//
// First-class fields represent authoritative, common attributes on domain
// models (e.g., Content.Name, Content.Description, Object.ObjectType).
// Extensible attributes are stored in JSON maps (ContentMetadata.Metadata,
// ObjectMetadata.Metadata). Avoid duplicating authoritative values in the JSON
// maps; if mirroring is needed for compatibility, treat the first-class fields
// as the source of truth.
package simplecontent

