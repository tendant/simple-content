package objectkey

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Generator defines the interface for object key generation strategies
type Generator interface {
	// GenerateKey creates an object key for storage backends
	GenerateKey(contentID, objectID uuid.UUID, metadata *KeyMetadata) string
}

// KeyMetadata contains information that influences key generation
type KeyMetadata struct {
	FileName        string
	ContentType     string
	TenantID        string
	OwnerID         string

	// Content classification
	IsOriginal      bool
	DerivationType  string    // "thumbnail", "preview", "transcode", etc.
	Variant         string    // "256x256", "1080p", "small", etc.
	ParentContentID uuid.UUID // For derived content
}

// LegacyGenerator provides the original flat structure for backwards compatibility
type LegacyGenerator struct{}

func NewLegacyGenerator() *LegacyGenerator {
	return &LegacyGenerator{}
}

func (g *LegacyGenerator) GenerateKey(contentID, objectID uuid.UUID, metadata *KeyMetadata) string {
	if metadata != nil && metadata.FileName != "" {
		return fmt.Sprintf("C/%s/%s/%s", contentID, objectID, metadata.FileName)
	}
	return fmt.Sprintf("C/%s/%s", contentID, objectID)
}

// GitLikeGenerator provides Git-style sharded storage with original/derived separation
// Original: originals/objects/ab/cd1234ef5678_filename
// Derived:  derived/{type}/{variant}/objects/ab/cd1234ef5678_filename
type GitLikeGenerator struct {
	// ShardLength controls how many characters to use for sharding (default: 2)
	ShardLength int
}

func NewGitLikeGenerator() *GitLikeGenerator {
	return &GitLikeGenerator{
		ShardLength: 2,
	}
}

func (g *GitLikeGenerator) GenerateKey(contentID, objectID uuid.UUID, metadata *KeyMetadata) string {
	// Use objectID for sharding since it's unique and random
	objectIDStr := strings.ReplaceAll(objectID.String(), "-", "")

	// Ensure we have enough characters for sharding
	if len(objectIDStr) < g.ShardLength {
		g.ShardLength = len(objectIDStr)
	}

	// Git-style sharding
	shardDir := objectIDStr[:g.ShardLength]
	remaining := objectIDStr[g.ShardLength:]

	// Build filename
	filename := remaining
	if metadata != nil && metadata.FileName != "" {
		filename = fmt.Sprintf("%s_%s", remaining, sanitizeFilename(metadata.FileName))
	}

	// Determine path prefix based on content type
	var pathPrefix string
	if metadata != nil && !metadata.IsOriginal && metadata.DerivationType != "" {
		// Derived content: derived/{type}/{variant}/objects/{shard}/{filename}
		variant := "default"
		if metadata.Variant != "" {
			variant = sanitizePathComponent(metadata.Variant)
		}
		pathPrefix = fmt.Sprintf("derived/%s/%s/objects/%s",
			sanitizePathComponent(metadata.DerivationType), variant, shardDir)
	} else {
		// Original content: originals/objects/{shard}/{filename}
		pathPrefix = fmt.Sprintf("originals/objects/%s", shardDir)
	}

	return fmt.Sprintf("%s/%s", pathPrefix, filename)
}

// TenantAwareGitLikeGenerator adds tenant isolation to Git-like sharding
// Structure: tenants/{tenant}/originals/objects/ab/cd1234ef5678_filename
//           tenants/{tenant}/derived/{type}/{variant}/objects/ab/cd1234ef5678_filename
type TenantAwareGitLikeGenerator struct {
	BaseGenerator Generator
	DefaultTenant string
}

func NewTenantAwareGitLikeGenerator() *TenantAwareGitLikeGenerator {
	return &TenantAwareGitLikeGenerator{
		BaseGenerator: NewGitLikeGenerator(),
		DefaultTenant: "default",
	}
}

func (g *TenantAwareGitLikeGenerator) GenerateKey(contentID, objectID uuid.UUID, metadata *KeyMetadata) string {
	// Determine tenant
	tenant := g.DefaultTenant
	if metadata != nil && metadata.TenantID != "" {
		tenant = sanitizePathComponent(metadata.TenantID)
	}

	// Get base key from underlying generator
	baseKey := g.BaseGenerator.GenerateKey(contentID, objectID, metadata)

	// Prefix with tenant
	return fmt.Sprintf("tenants/%s/%s", tenant, baseKey)
}

// HashedGitLikeGenerator uses content-based hashing for deterministic sharding
// Useful for deduplication or when consistent keys are needed
type HashedGitLikeGenerator struct {
	ShardLength int
}

func NewHashedGitLikeGenerator() *HashedGitLikeGenerator {
	return &HashedGitLikeGenerator{
		ShardLength: 2,
	}
}

func (g *HashedGitLikeGenerator) GenerateKey(contentID, objectID uuid.UUID, metadata *KeyMetadata) string {
	// Create deterministic hash from contentID + objectID
	hash := sha256.Sum256([]byte(contentID.String() + objectID.String()))
	hashStr := fmt.Sprintf("%x", hash)

	// Git-style sharding
	shardDir := hashStr[:g.ShardLength]
	remaining := hashStr[g.ShardLength:16] // Use 16 chars total for filename

	// Build filename
	filename := remaining
	if metadata != nil && metadata.FileName != "" {
		filename = fmt.Sprintf("%s_%s", remaining, sanitizeFilename(metadata.FileName))
	}

	// Determine path prefix
	var pathPrefix string
	if metadata != nil && !metadata.IsOriginal && metadata.DerivationType != "" {
		variant := "default"
		if metadata.Variant != "" {
			variant = sanitizePathComponent(metadata.Variant)
		}
		pathPrefix = fmt.Sprintf("derived/%s/%s/objects/%s",
			sanitizePathComponent(metadata.DerivationType), variant, shardDir)
	} else {
		pathPrefix = fmt.Sprintf("originals/objects/%s", shardDir)
	}

	return fmt.Sprintf("%s/%s", pathPrefix, filename)
}

// CustomFuncGenerator allows users to provide their own key generation function
type CustomFuncGenerator struct {
	GenerateFunc func(contentID, objectID uuid.UUID, metadata *KeyMetadata) string
}

func NewCustomFuncGenerator(fn func(contentID, objectID uuid.UUID, metadata *KeyMetadata) string) *CustomFuncGenerator {
	return &CustomFuncGenerator{
		GenerateFunc: fn,
	}
}

func (g *CustomFuncGenerator) GenerateKey(contentID, objectID uuid.UUID, metadata *KeyMetadata) string {
	return g.GenerateFunc(contentID, objectID, metadata)
}

// Helper functions for path sanitization
func sanitizeFilename(filename string) string {
	// Replace problematic characters for filesystem compatibility
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
	)
	return replacer.Replace(filename)
}

func sanitizePathComponent(component string) string {
	// Similar to filename but allow some special chars that are OK in paths
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
	)
	return strings.ToLower(replacer.Replace(component))
}

// Predefined generators for common use cases

// NewRecommendedGenerator returns the recommended generator for new installations
func NewRecommendedGenerator() Generator {
	return NewGitLikeGenerator()
}

// NewMultiTenantGenerator returns a generator optimized for multi-tenant scenarios
func NewMultiTenantGenerator() Generator {
	return NewTenantAwareGitLikeGenerator()
}

// NewHighPerformanceGenerator returns a generator optimized for high-performance scenarios
func NewHighPerformanceGenerator() Generator {
	return &GitLikeGenerator{ShardLength: 3} // 3-char sharding for even better distribution
}