# Testing Preset Example

This example demonstrates the **Testing Preset** - the easiest way to write unit and integration tests.

## Features

- **One-line setup**: `NewTesting(t)` creates an isolated service instance
- **In-memory everything**: Database and storage (blazingly fast)
- **Automatic cleanup**: No manual cleanup needed via `t.Cleanup()`
- **Parallel execution**: Each test gets its own isolated service
- **No mocking required**: Real service implementation, real operations

## Quick Start

```bash
go test -v
```

## What This Example Shows

1. **Upload and Download** - Basic content operations in tests
2. **Derived Content** - Creating and querying thumbnails
3. **Metadata Operations** - Custom metadata in tests
4. **Parallel Execution** - Running tests concurrently
5. **Service Isolation** - Each service is completely isolated
6. **Convenience Function** - Using `TestService()` helper

## Code Walkthrough

### Basic Test Setup

```go
func TestMyFeature(t *testing.T) {
    // One line - creates service, registers cleanup
    svc := simplecontentpresets.NewTesting(t)

    // Use service normally
    content, err := svc.UploadContent(ctx, request)
    require.NoError(t, err)

    // No cleanup code needed - automatic!
}
```

### Parallel Tests

```go
func TestParallelExecution(t *testing.T) {
    t.Run("test1", func(t *testing.T) {
        t.Parallel() // Run concurrently
        svc := simplecontentpresets.NewTesting(t)
        // Each gets isolated service
    })

    t.Run("test2", func(t *testing.T) {
        t.Parallel()
        svc := simplecontentpresets.NewTesting(t)
        // Completely separate instance
    })
}
```

### Test Isolation

```go
func TestIsolation(t *testing.T) {
    svc1 := simplecontentpresets.NewTesting(t)
    svc2 := simplecontentpresets.NewTesting(t)

    // Upload to svc1
    content, _ := svc1.UploadContent(ctx, req)

    // Content does NOT exist in svc2
    _, err := svc2.GetContent(ctx, content.ID)
    assert.Error(t, err) // Isolated!
}
```

## Test Patterns

### Table-Driven Tests

```go
func TestUploadVariousTypes(t *testing.T) {
    tests := []struct {
        name     string
        fileType string
        data     string
    }{
        {"PDF", "application/pdf", "fake-pdf"},
        {"Image", "image/jpeg", "fake-jpg"},
        {"Video", "video/mp4", "fake-mp4"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            svc := simplecontentpresets.NewTesting(t)
            // Test with tt.fileType and tt.data
        })
    }
}
```

### Benchmarks

```go
func BenchmarkUpload(b *testing.B) {
    // Note: Use testing.B for benchmarks
    svc := simplecontentpresets.NewTesting(b)
    ctx := context.Background()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        svc.UploadContent(ctx, request)
    }
}
```

## Advantages Over Mocking

**With Mocks** (traditional approach):
```go
// Mock repository
mockRepo := &MockRepository{}
mockRepo.On("CreateContent", ...).Return(nil)
mockRepo.On("CreateObject", ...).Return(nil)
// ... many more mocks

// Mock storage
mockStorage := &MockBlobStore{}
mockStorage.On("Put", ...).Return(nil)

// Complex setup, brittle tests
```

**With Testing Preset** (simple approach):
```go
// One line - real implementation
svc := simplecontentpresets.NewTesting(t)

// Real operations, actual behavior
content, err := svc.UploadContent(ctx, req)
```

## Benefits

1. **Fast**: In-memory = no disk I/O
2. **Isolated**: Each test gets fresh state
3. **Real**: Tests actual service behavior
4. **Simple**: One line setup, no cleanup
5. **Parallel**: Safe concurrent execution
6. **Reliable**: No mock configuration drift

## When to Use

Use the **Testing Preset** when:

- Writing unit tests for your application
- Testing integration with Simple Content
- Running CI/CD test suites
- Benchmarking performance
- Testing error handling

## Customization

```go
// Add custom options (future extension point)
svc := simplecontentpresets.NewTesting(t,
    simplecontentpresets.WithTestFixtures(), // Load sample data
)
```

## Next Steps

- See [preset-development](../preset-development/) for local development
- See [QUICKSTART.md](../../QUICKSTART.md) for more usage examples
- See [HOOKS_GUIDE.md](../../HOOKS_GUIDE.md) for testing hooks
