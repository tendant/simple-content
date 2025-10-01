# Content Scanner

The scanner package provides a generic pattern for processing contents in bulk. It queries contents with filters and calls an external processor for each content found.

## Core Concept

The scanner follows a simple pattern:
1. Query contents using filters (via admin service)
2. Iterate through results in batches
3. Call external processor for each content
4. Track statistics and errors

## Quick Start

```go
import (
    "github.com/tendant/simple-content/pkg/simplecontent/admin"
    "github.com/tendant/simple-content/pkg/simplecontent/scan"
)

// Create scanner
adminSvc := admin.New(repo)
scanner := scan.New(adminSvc)

// Create your processor
type MyProcessor struct{}

func (p *MyProcessor) Process(ctx context.Context, content *simplecontent.Content) error {
    // Your processing logic here
    fmt.Printf("Processing %s\n", content.ID)
    return nil
}

// Scan and process
result, err := scanner.Scan(ctx, scan.ScanOptions{
    Filters: admin.ContentFilters{
        Status: stringPtr("uploaded"),
    },
    Processor: &MyProcessor{},
})

fmt.Printf("Processed %d contents\n", result.TotalProcessed)
```

## ContentProcessor Interface

External apps implement this interface to define custom processing logic:

```go
type ContentProcessor interface {
    Process(ctx context.Context, content *simplecontent.Content) error
}
```

**Simple implementation:**
```go
type PrintProcessor struct{}

func (p *PrintProcessor) Process(ctx context.Context, content *simplecontent.Content) error {
    fmt.Printf("%s: %s\n", content.ID, content.Name)
    return nil
}
```

## Scanner API

### Scan Method

Process contents matching filters with a processor:

```go
result, err := scanner.Scan(ctx, scan.ScanOptions{
    Filters:   filters,      // admin.ContentFilters
    Processor: processor,    // ContentProcessor implementation
    BatchSize: 100,          // Optional (default: 100)
    Limit:     1000,         // Optional (0 = no limit)
    DryRun:    false,        // Optional (default: false)
    OnProgress: func(processed, total int64) {
        fmt.Printf("%d/%d\n", processed, total)
    },
})
```

**Options:**
- `Filters` - Query filters (status, tenant, document type, etc.)
- `Processor` - Your processor implementation (required unless DryRun)
- `BatchSize` - How many contents to query at once (default: 100, affects memory/performance)
- `Limit` - Maximum total to process (0 or negative = no limit, useful for testing/incremental backfill)
- `DryRun` - If true, shows what would be processed without calling processor
- `OnProgress` - Callback for progress tracking (optional)

**Result:**
```go
type ScanResult struct {
    TotalFound     int64      // Contents found matching filters
    TotalProcessed int64      // Successfully processed
    TotalFailed    int64      // Failed processing
    FailedIDs      []string   // IDs of failed contents
}
```

### ForEach Method

Convenience method for inline processing:

```go
result, err := scanner.ForEach(ctx, filters,
    func(ctx context.Context, content *simplecontent.Content) error {
        fmt.Printf("Processing %s\n", content.ID)
        return doSomething(content)
    })
```

## Use Cases

### 1. Event Emission (Job Backfill)

Create events/jobs for existing contents:

```go
type JobEmitterProcessor struct {
    natsConn *nats.Conn
    rulesEngine *RulesEngine
}

func (p *JobEmitterProcessor) Process(ctx context.Context, content *simplecontent.Content) error {
    // Get tenant-specific rules
    rules := p.rulesEngine.GetRulesForContent(content)

    // Emit event for each rule
    for _, rule := range rules {
        event := createEvent(content, rule)
        p.natsConn.Publish("content.ready", event)
    }
    return nil
}

// Usage
result, _ := scanner.Scan(ctx, scan.ScanOptions{
    Filters: admin.ContentFilters{
        Status: stringPtr("uploaded"),
        DocumentType: stringPtr("image/*"),
    },
    Processor: jobEmitter,
})
```

### 2. Status Updates

Bulk update content status:

```go
type StatusUpdaterProcessor struct {
    svc simplecontent.Service
    newStatus string
}

func (p *StatusUpdaterProcessor) Process(ctx context.Context, content *simplecontent.Content) error {
    return p.svc.UpdateContent(ctx, simplecontent.UpdateContentRequest{
        ID:     content.ID,
        Status: &p.newStatus,
    })
}

// Mark all created content as processing
scanner.Scan(ctx, scan.ScanOptions{
    Filters: admin.ContentFilters{
        Status: stringPtr("created"),
    },
    Processor: &StatusUpdaterProcessor{
        svc: svc,
        newStatus: "processing",
    },
})
```

### 3. Data Validation

Validate content integrity:

```go
type ValidatorProcessor struct {
    svc simplecontent.StorageService
}

func (p *ValidatorProcessor) Process(ctx context.Context, content *simplecontent.Content) error {
    // Check if content has objects
    objects, err := p.svc.GetObjectsByContentID(ctx, content.ID)
    if err != nil {
        return err
    }

    if len(objects) == 0 {
        return fmt.Errorf("content has no objects")
    }

    // Validate object metadata
    for _, obj := range objects {
        if obj.Status != "uploaded" {
            return fmt.Errorf("object %s not uploaded", obj.ID)
        }
    }

    return nil
}

// Find invalid contents
result, _ := scanner.Scan(ctx, scan.ScanOptions{
    Filters: admin.ContentFilters{
        Status: stringPtr("uploaded"),
    },
    Processor: validator,
})

fmt.Printf("Found %d invalid contents: %v\n",
    result.TotalFailed, result.FailedIDs)
```

### 4. Report Generation

Generate CSV reports:

```go
type CSVReporterProcessor struct {
    writer *csv.Writer
}

func (p *CSVReporterProcessor) Process(ctx context.Context, content *simplecontent.Content) error {
    return p.writer.Write([]string{
        content.ID.String(),
        content.TenantID.String(),
        content.Name,
        content.DocumentType,
        content.Status,
        content.CreatedAt.Format(time.RFC3339),
    })
}

// Generate report
file, _ := os.Create("report.csv")
defer file.Close()

writer := csv.NewWriter(file)
defer writer.Flush()

scanner.Scan(ctx, scan.ScanOptions{
    Filters: admin.ContentFilters{
        CreatedAfter: &startDate,
    },
    Processor: &CSVReporterProcessor{writer: writer},
})
```

### 5. Data Migration

Migrate content to new system:

```go
type MigrationProcessor struct {
    targetAPI *NewSystemAPI
}

func (p *MigrationProcessor) Process(ctx context.Context, content *simplecontent.Content) error {
    // Download content
    reader, _ := svc.DownloadContent(ctx, content.ID)
    defer reader.Close()

    // Upload to new system
    return p.targetAPI.Upload(content, reader)
}
```

## Advanced Patterns

### BatchSize vs Limit

Understanding the difference:

**BatchSize** - Query efficiency:
- How many contents to query from database at once
- Affects memory usage and query performance
- Default: 100
- Example: `BatchSize: 1000` queries 1000 items per database call

**Limit** - Total processing cap:
- Maximum total contents to process (across all batches)
- Useful for testing and incremental backfill
- Default: 0 (no limit)
- Example: `Limit: 10` stops after processing 10 items total

**Use cases:**

```go
// Process everything efficiently
scanner.Scan(ctx, scan.ScanOptions{
    Filters:   filters,
    Processor: processor,
    BatchSize: 1000,  // Large batches for efficiency
    Limit:     0,     // No limit, process all
})

// Test with small sample
scanner.Scan(ctx, scan.ScanOptions{
    Filters:   filters,
    Processor: processor,
    Limit:     10,    // Only process 10 items (for testing)
})

// Incremental backfill (process in chunks)
scanner.Scan(ctx, scan.ScanOptions{
    Filters:   filters,
    Processor: processor,
    BatchSize: 100,   // Query 100 at a time
    Limit:     1000,  // Process 1000 total, then stop
})
// Run again later to continue from where you left off
```

### Chain Multiple Processors

Process contents through multiple steps:

```go
import "github.com/tendant/simple-content/examples/scan/processors"

chain := processors.NewChainProcessor(
    &ValidatorProcessor{},      // First validate
    &StatusUpdaterProcessor{},  // Then update status
    &JobCreatorProcessor{},     // Then create jobs
)

scanner.Scan(ctx, scan.ScanOptions{
    Filters:   filters,
    Processor: chain,
})
```

### Conditional Processing

Process only contents matching a condition:

```go
import "github.com/tendant/simple-content/examples/scan/processors"

// Only process JPEGs
processor := processors.NewConditionalProcessor(
    func(c *simplecontent.Content) bool {
        return c.DocumentType == "image/jpeg"
    },
    &ThumbnailGeneratorProcessor{},
)

scanner.Scan(ctx, scan.ScanOptions{
    Filters:   filters,
    Processor: processor,
})
```

Built-in condition helpers:
- `processors.OnlyImages` - Image content types
- `processors.OnlyVideos` - Video content types
- `processors.OnlyStatus(status)` - Specific status
- `processors.OnlyOriginals` - Non-derived content
- `processors.OnlyDerived` - Derived content

### Progress Tracking

Track scan progress:

```go
total := int64(0)
scanner.Scan(ctx, scan.ScanOptions{
    Filters: filters,
    Processor: processor,
    OnProgress: func(processed, found int64) {
        total = found
        pct := float64(processed) / float64(found) * 100
        fmt.Printf("Progress: %.1f%% (%d/%d)\n", pct, processed, found)
    },
})
```

### Error Handling

Failed contents are tracked but don't stop the scan:

```go
result, err := scanner.Scan(ctx, scan.ScanOptions{
    Filters:   filters,
    Processor: processor,
})

if err != nil {
    log.Fatal("Scan error:", err)
}

if result.TotalFailed > 0 {
    fmt.Printf("Failed to process %d contents:\n", result.TotalFailed)
    for _, id := range result.FailedIDs {
        fmt.Printf("  - %s\n", id)
    }
}
```

## Testing

### Dry-Run Mode

Test what would be processed without actually processing:

```go
result, _ := scanner.Scan(ctx, scan.ScanOptions{
    Filters:   filters,
    DryRun:    true,  // No processing, just report
})

fmt.Printf("Would process %d contents\n", result.TotalFound)
```

### Mock Processor

Create a mock for testing:

```go
type MockProcessor struct {
    ProcessFunc func(context.Context, *simplecontent.Content) error
}

func (m *MockProcessor) Process(ctx context.Context, content *simplecontent.Content) error {
    if m.ProcessFunc != nil {
        return m.ProcessFunc(ctx, content)
    }
    return nil
}

// Test
mock := &MockProcessor{
    ProcessFunc: func(ctx context.Context, c *simplecontent.Content) error {
        // Verify content
        assert.Equal(t, "uploaded", c.Status)
        return nil
    },
}

scanner.Scan(ctx, scan.ScanOptions{
    Filters:   filters,
    Processor: mock,
})
```

## Performance Considerations

1. **Batch Size**: Adjust based on memory and query performance
   - Small batches (10-50): Less memory, more queries
   - Large batches (100-1000): More memory, fewer queries

2. **Concurrent Processing**: Implement concurrent processor if needed
   ```go
   type ConcurrentProcessor struct {
       processor scan.ContentProcessor
       workers   int
   }
   ```

3. **Progress Tracking**: Use `OnProgress` for long-running scans

4. **Error Handling**: Failed items don't stop the scan

## Integration with simple-process

Example: Create simple-process jobs during scan:

```go
type JobCreatorProcessor struct {
    asyncRunner *runner.AsyncRunner
    contentAPI  *ContentAPIClient
    rulesEngine *RulesEngine
}

func (p *JobCreatorProcessor) Process(ctx context.Context, content *simplecontent.Content) error {
    // Get tenant rules
    rules := p.rulesEngine.GetRulesForContent(content)

    for _, rule := range rules {
        // Get presigned download URL
        downloadURL, _ := p.contentAPI.GetPresignedURL(content.ID, 1*time.Hour)

        // Create simple-process Job
        job := contracts.Job{
            JobID: uuid.New().String(),
            UoW:   rule.UoWType,  // "thumbnail", "transcode", etc.
            File: contracts.File{
                ID:   content.ID.String(),
                Blob: contracts.Blob{
                    Location: downloadURL,
                },
            },
            Hints: rule.Params,
        }

        // Submit to simple-process
        if err := p.asyncRunner.Run(ctx, nil, job); err != nil {
            return err
        }
    }

    return nil
}

// Backfill jobs for uploaded images
scanner.Scan(ctx, scan.ScanOptions{
    Filters: admin.ContentFilters{
        Status:       stringPtr("uploaded"),
        DocumentType: stringPtr("image/*"),
    },
    Processor: jobCreator,
})
```

## Examples

See working examples in:
- `examples/scan/main.go` - Complete examples
- `examples/scan/processors/` - Processor implementations

Run the example:
```bash
go run ./examples/scan
```
