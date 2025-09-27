# Object Key Generation

This package provides pluggable object key generators for optimal storage performance and organization in the simple-content library.

## Overview

Object keys determine where and how files are stored in the underlying storage backends. The system uses a pluggable architecture that allows different key generation strategies optimized for various deployment scenarios.

## Problem Solved

**Before**: Flat key structure like `C/{contentID}/{objectID}` could create performance issues with too many files in a single directory.

**After**: Git-like sharded structure like `originals/objects/ab/cd1234...` limits directory size and provides better filesystem performance.

## Available Generators

### GitLikeGenerator (Recommended)
Git-style sharded storage for optimal filesystem performance.

**Structure:**
- Original: `originals/objects/{shard}/{objectId}_{filename}`
- Derived: `derived/{type}/{variant}/objects/{shard}/{objectId}_{filename}`

**Benefits:**
- Limits directory size to ~256 entries
- Clear content hierarchy
- Better I/O performance

### TenantAwareGitLikeGenerator
Multi-tenant organization with Git-like sharding.

**Structure:**
- `tenants/{tenant}/originals/objects/{shard}/{objectId}_{filename}`
- `tenants/{tenant}/derived/{type}/{variant}/objects/{shard}/{objectId}_{filename}`

**Use case:** Multi-tenant SaaS applications requiring data isolation

### LegacyGenerator
Backwards compatibility with existing flat structure.

**Structure:** `C/{contentId}/{objectId}/{filename}`

**Use case:** Migration scenarios or legacy compatibility

### HashedGitLikeGenerator
Content-based hashing for deterministic sharding.

**Benefits:**
- Consistent keys for the same content
- Useful for deduplication scenarios

### CustomFuncGenerator
User-defined key generation logic.

**Use case:** Specialized requirements or complex organizational needs

## Quick Start

### Basic Usage

```go
import "github.com/tendant/simple-content/pkg/simplecontent/objectkey"

// Use recommended Git-like generator
generator := objectkey.NewGitLikeGenerator()

// Generate key for original content
key := generator.GenerateKey(contentID, objectID, &objectkey.KeyMetadata{
    FileName:   "document.pdf",
    IsOriginal: true,
})
// Result: originals/objects/ab/cd1234ef5678_document.pdf

// Generate key for derived content
thumbKey := generator.GenerateKey(contentID, objectID, &objectkey.KeyMetadata{
    FileName:        "thumb.jpg",
    IsOriginal:      false,
    DerivationType:  "thumbnail",
    Variant:         "256x256",
    ParentContentID: parentID,
})
// Result: derived/thumbnail/256x256/objects/ab/cd1234ef5678_thumb.jpg
```

### Service Integration

```go
import (
    "github.com/tendant/simple-content/pkg/simplecontent"
    "github.com/tendant/simple-content/pkg/simplecontent/objectkey"
)

// Configure service with custom key generator
service, err := simplecontent.New(
    simplecontent.WithRepository(repo),
    simplecontent.WithBlobStore("fs", fsBackend),
    simplecontent.WithObjectKeyGenerator(objectkey.NewGitLikeGenerator()),
)
```

### Environment Configuration

```bash
# Git-like sharding (recommended, default)
OBJECT_KEY_GENERATOR=git-like

# Multi-tenant aware
OBJECT_KEY_GENERATOR=tenant-aware

# High-performance (3-char sharding)
OBJECT_KEY_GENERATOR=high-performance

# Legacy compatibility
OBJECT_KEY_GENERATOR=legacy
```

## Custom Generator Example

```go
type TimestampGenerator struct {
    Prefix string
}

func (g *TimestampGenerator) GenerateKey(contentID, objectID uuid.UUID, metadata *objectkey.KeyMetadata) string {
    timestamp := time.Now().Format("2006/01/02")
    if metadata != nil && !metadata.IsOriginal {
        return fmt.Sprintf("%s/%s/derived/%s/%s_%s",
            g.Prefix, timestamp, metadata.DerivationType,
            objectID.String()[:8], metadata.FileName)
    }
    return fmt.Sprintf("%s/%s/original/%s_%s",
        g.Prefix, timestamp, objectID.String()[:8], metadata.FileName)
}

// Usage
generator := &TimestampGenerator{Prefix: "archive"}
```

## Performance Benefits

- **Sharding**: Limits directory size to ~256 entries for optimal filesystem performance
- **Organization**: Clear separation between originals and derived content
- **Scalability**: Handles millions of objects efficiently
- **Flexibility**: Easy to customize for specific deployment needs

## Migration

The system supports gradual migration:
1. New objects use the configured generator
2. Existing objects retain their current keys
3. No disruption to existing functionality
4. Optional bulk migration tools can be implemented

## Testing

Run the comprehensive test suite:

```bash
go test ./pkg/simplecontent/objectkey/... -v
```

View example usage:

```bash
go run ./examples/objectkey
```

## Examples

See `examples/objectkey/main.go` for comprehensive examples of all generator types and their output formats.