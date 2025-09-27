# Object Operations Review and Simplification Plan

## Current State Analysis

The current object upload/download operations have these issues:

### 1. Repetitive URL Generation Methods
```go
// Current: 3 separate methods with identical patterns
GetUploadURL(ctx context.Context, id uuid.UUID) (string, error)
GetDownloadURL(ctx context.Context, id uuid.UUID) (string, error)
GetPreviewURL(ctx context.Context, id uuid.UUID) (string, error)
```

**Problems:**
- Each method has identical error handling and object lookup logic
- No flexibility for URL options (expiration, permissions, etc.)
- Violates DRY principle

### 2. Upload Method Inconsistency
```go
UploadObject(ctx context.Context, req UploadObjectRequest) error
```

**Problems:**
- Internal branching logic based on MimeType presence
- Uses different backend methods (`Upload` vs `UploadWithParams`)
- Could be more flexible for future upload options

### 3. Download Method is Simple but Inflexible
```go
DownloadObject(ctx context.Context, id uuid.UUID) (io.ReadCloser, error)
```

**This one is actually good** - simple, focused, does one thing well.

## Proposed Simplifications

### Option 1: Functional Options Pattern (Recommended)

#### URL Operations - Consolidate to Single Method
```go
// Replace 3 methods with 1 flexible method
GetObjectURL(ctx context.Context, id uuid.UUID, options ...URLOption) (string, error)

// URL option functions
func WithUploadURL() URLOption
func WithDownloadURL() URLOption
func WithPreviewURL() URLOption
func WithExpiration(duration time.Duration) URLOption
func WithPermissions(perms string) URLOption
```

**Benefits:**
- Single implementation with shared error handling
- Extensible for future URL options (expiration, permissions, etc.)
- Maintains backward compatibility through convenience functions
- Follows same pattern as our successful ListDerivedContent simplification

#### Upload Operations - Simplify to Single Path
```go
// Keep existing but make backend interface consistent
UploadObject(ctx context.Context, req UploadObjectRequest) error

// Ensure UploadObjectRequest supports all options
type UploadObjectRequest struct {
    ObjectID uuid.UUID
    Reader   io.Reader
    MimeType string                // Optional
    Metadata map[string]interface{} // Optional - for future extensibility
}
```

**Benefits:**
- Single code path regardless of options
- Backend interface can be simplified to one method
- Request struct can grow without breaking changes

### Option 2: Keep Current + Add Convenience

Keep current methods but add convenience functions:

```go
// Keep existing methods
GetUploadURL(ctx context.Context, id uuid.UUID) (string, error)
GetDownloadURL(ctx context.Context, id uuid.UUID) (string, error)
GetPreviewURL(ctx context.Context, id uuid.UUID) (string, error)

// Add flexible method for advanced use cases
GetObjectURLWithOptions(ctx context.Context, id uuid.UUID, options URLOptions) (string, error)
```

**Benefits:**
- Zero breaking changes
- Simple cases stay simple
- Advanced cases become possible

**Drawbacks:**
- Still have code duplication
- Interface continues to grow

## Recommendation: Option 1 (Functional Options)

This follows the successful pattern we used for ListDerivedContent:

1. **Consolidate URL methods** into `GetObjectURL` with functional options
2. **Provide convenience functions** for backward compatibility
3. **Simplify upload implementation** to single code path
4. **Keep download method as-is** (it's already good)

### Implementation Plan

1. Add functional options for URL operations
2. Implement consolidated `GetObjectURL` method
3. Update backend interface to be consistent
4. Add convenience functions to maintain backward compatibility
5. Update server handlers to use new methods
6. Update tests and documentation

### Backward Compatibility

Maintain existing method signatures as convenience functions:

```go
// Convenience functions (maintain backward compatibility)
func (s *service) GetUploadURL(ctx context.Context, id uuid.UUID) (string, error) {
    return s.GetObjectURL(ctx, id, WithUploadURL())
}

func (s *service) GetDownloadURL(ctx context.Context, id uuid.UUID) (string, error) {
    return s.GetObjectURL(ctx, id, WithDownloadURL())
}

func (s *service) GetPreviewURL(ctx context.Context, id uuid.UUID) (string, error) {
    return s.GetObjectURL(ctx, id, WithPreviewURL())
}
```

## Expected Benefits

1. **Simplicity**: Reduce interface methods from 5 to 3 (UploadObject, DownloadObject, GetObjectURL)
2. **Flexibility**: Options pattern allows future extensibility without breaking changes
3. **Consistency**: Follows same successful pattern as ListDerivedContent
4. **Maintainability**: Single implementation for URL generation reduces code duplication
5. **Backward Compatibility**: Existing code continues to work