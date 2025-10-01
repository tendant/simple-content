# Content Scanner

The scanner package provides a generic pattern for processing contents in bulk, enabling backfill operations, batch processing, and content migrations.

## Overview

**What it does:**
- Queries contents with flexible filters (using admin service)
- Iterates through results in efficient batches
- Calls external processor for each content
- Tracks statistics and errors

**What it doesn't do:**
- No job creation logic (external apps decide)
- No event formatting (external apps format)
- No processing rules (external apps implement)

## Architecture

```
┌─────────────────────────────────────────────────┐
│ simple-content Scanner                          │
│                                                 │
│  Query contents → Iterate → Call processor     │
│                                                 │
│  Admin Service (queries) + ContentProcessor     │
└──────────────────┬──────────────────────────────┘
                   │
                   │ ContentProcessor interface
                   │
                   ▼
┌─────────────────────────────────────────────────┐
│ External App (Your Code)                        │
│                                                 │
│  Process(content) {                             │
│    - Apply tenant rules                         │
│    - Format events/jobs                         │
│    - Send to message queue                      │
│    - Update status                              │
│    - Generate reports                           │
│    - Anything you want!                         │
│  }                                              │
└─────────────────────────────────────────────────┘
```

## Quick Start

```go
import (
    "github.com/tendant/simple-content/pkg/simplecontent/admin"
    "github.com/tendant/simple-content/pkg/simplecontent/scan"
)

// 1. Create scanner
adminSvc := admin.New(repo)
scanner := scan.New(adminSvc)

// 2. Implement processor
type MyProcessor struct{}

func (p *MyProcessor) Process(ctx context.Context, content *simplecontent.Content) error {
    // Your logic here
    return nil
}

// 3. Scan and process
result, _ := scanner.Scan(ctx, scan.ScanOptions{
    Filters: admin.ContentFilters{
        Status: stringPtr("uploaded"),
    },
    Processor: &MyProcessor{},
})
```

## Use Cases

### 1. Job Backfill (Primary Use Case)

Create simple-process jobs for existing contents:

```go
type JobCreatorProcessor struct {
    natsConn    *nats.Conn
    rulesEngine *RulesEngine
    contentAPI  *ContentAPIClient
}

func (p *JobCreatorProcessor) Process(ctx context.Context, content *simplecontent.Content) error {
    // Get tenant-specific rules
    rules := p.rulesEngine.GetRulesForContent(content)

    for _, rule := range rules {
        // Get presigned URL
        downloadURL, _ := p.contentAPI.GetPresignedURL(content.ID, 1*time.Hour)

        // Create simple-process Job
        job := contracts.Job{
            JobID: uuid.New().String(),
            UoW:   rule.UoWType,  // "thumbnail", "transcode", etc.
            File: contracts.File{
                ID:   content.ID.String(),
                Blob: contracts.Blob{Location: downloadURL},
            },
            Hints: rule.Params,  // {"sizes": [128, 256, 512]}
        }

        // Submit to NATS for processing
        data, _ := json.Marshal(job)
        p.natsConn.Publish("jobs.processing", data)
    }

    return nil
}

// Backfill thumbnails for all uploaded images
result, _ := scanner.Scan(ctx, scan.ScanOptions{
    Filters: admin.ContentFilters{
        Status:       stringPtr("uploaded"),
        DocumentType: stringPtr("image/*"),
        TenantID:     &tenantID,
    },
    Processor: jobCreator,
})

fmt.Printf("Created jobs for %d images\n", result.TotalProcessed)
```

### 2. Event Emission

Emit events for external processing:

```go
type EventEmitterProcessor struct {
    natsConn *nats.Conn
}

func (p *EventEmitterProcessor) Process(ctx context.Context, content *simplecontent.Content) error {
    // Format event (external app decides format)
    event := map[string]interface{}{
        "content_id":    content.ID.String(),
        "tenant_id":     content.TenantID.String(),
        "document_type": content.DocumentType,
        "status":        content.Status,
    }

    data, _ := json.Marshal(event)
    return p.natsConn.Publish("content.ready", data)
}
```

### 3. Bulk Status Updates

```go
type StatusUpdaterProcessor struct {
    svc simplecontent.Service
}

func (p *StatusUpdaterProcessor) Process(ctx context.Context, content *simplecontent.Content) error {
    content.Status = "processing"
    return p.svc.UpdateContent(ctx, simplecontent.UpdateContentRequest{
        Content: content,
    })
}

// Mark all created content as processing
scanner.Scan(ctx, scan.ScanOptions{
    Filters: admin.ContentFilters{
        Status: stringPtr("created"),
    },
    Processor: statusUpdater,
})
```

### 4. Data Validation

```go
type ValidatorProcessor struct {
    svc simplecontent.StorageService
}

func (p *ValidatorProcessor) Process(ctx context.Context, content *simplecontent.Content) error {
    objects, err := p.svc.GetObjectsByContentID(ctx, content.ID)
    if err != nil {
        return err
    }

    if len(objects) == 0 {
        return fmt.Errorf("no objects found")
    }

    return nil
}

// Find invalid contents
result, _ := scanner.Scan(ctx, scan.ScanOptions{
    Filters:   filters,
    Processor: validator,
})

fmt.Printf("Found %d invalid: %v\n", result.TotalFailed, result.FailedIDs)
```

### 5. Report Generation

```go
type CSVReporterProcessor struct {
    writer *csv.Writer
}

func (p *CSVReporterProcessor) Process(ctx context.Context, content *simplecontent.Content) error {
    return p.writer.Write([]string{
        content.ID.String(),
        content.Name,
        content.Status,
        content.CreatedAt.Format(time.RFC3339),
    })
}
```

## Features

### Filtering

Use admin filters to select specific contents:

```go
scanner.Scan(ctx, scan.ScanOptions{
    Filters: admin.ContentFilters{
        Status:         stringPtr("uploaded"),
        TenantID:       &tenantID,
        DocumentType:   stringPtr("image/*"),
        CreatedAfter:   &startDate,
        DerivationType: stringPtr("thumbnail"),
    },
    Processor: processor,
})
```

### Dry-Run Mode

Test without processing:

```go
result, _ := scanner.Scan(ctx, scan.ScanOptions{
    Filters:   filters,
    DryRun:    true,  // Just report what would be processed
})
```

### Progress Tracking

Monitor long-running scans:

```go
scanner.Scan(ctx, scan.ScanOptions{
    Filters:   filters,
    Processor: processor,
    OnProgress: func(processed, total int64) {
        pct := float64(processed) / float64(total) * 100
        fmt.Printf("Progress: %.1f%%\n", pct)
    },
})
```

### Error Handling

Failed items are tracked but don't stop the scan:

```go
result, _ := scanner.Scan(ctx, scan.ScanOptions{
    Filters:   filters,
    Processor: processor,
})

if result.TotalFailed > 0 {
    fmt.Printf("Failed: %d\n", result.TotalFailed)
    fmt.Printf("IDs: %v\n", result.FailedIDs)
}
```

### Batch Processing

Configure batch size for efficiency:

```go
scanner.Scan(ctx, scan.ScanOptions{
    Filters:   filters,
    Processor: processor,
    BatchSize: 1000,  // Process 1000 at a time
})
```

## Advanced Patterns

### Chain Processors

Execute multiple processors in sequence:

```go
import "github.com/tendant/simple-content/examples/scan/processors"

chain := processors.NewChainProcessor(
    &ValidatorProcessor{},    // First validate
    &StatusUpdater{},         // Then update status
    &JobCreator{},            // Then create jobs
)

scanner.Scan(ctx, scan.ScanOptions{
    Filters:   filters,
    Processor: chain,
})
```

### Conditional Processing

Process only matching contents:

```go
import "github.com/tendant/simple-content/examples/scan/processors"

// Only process JPEGs
processor := processors.NewConditionalProcessor(
    func(c *simplecontent.Content) bool {
        return c.DocumentType == "image/jpeg"
    },
    &ThumbnailProcessor{},
)
```

Built-in conditions:
- `processors.OnlyImages`
- `processors.OnlyVideos`
- `processors.OnlyOriginals`
- `processors.OnlyDerived`
- `processors.OnlyStatus(status)`

### ForEach Convenience

Process with inline function:

```go
scanner.ForEach(ctx, filters,
    func(ctx context.Context, content *simplecontent.Content) error {
        // Process inline
        return nil
    })
```

## Integration with simple-process

Complete example for job backfill:

```go
// 1. Define tenant rules (external app)
type TenantRules struct {
    ImageSizes []int
    VideoFormats []string
}

func (r *RulesEngine) GetRulesForContent(content *Content) []ProcessingRule {
    // External app's logic
    if content.DocumentType == "image/jpeg" {
        return []ProcessingRule{
            {UoWType: "thumbnail", Params: {"sizes": [128, 256, 512]}},
            {UoWType: "metadata", Params: {}},
        }
    }
    return nil
}

// 2. Create job processor (external app)
type JobCreator struct {
    asyncRunner *runner.AsyncRunner
    rulesEngine *RulesEngine
    contentAPI  *ContentAPIClient
}

func (p *JobCreator) Process(ctx context.Context, content *Content) error {
    rules := p.rulesEngine.GetRulesForContent(content)

    for _, rule := range rules {
        downloadURL, _ := p.contentAPI.GetPresignedURL(content.ID)

        job := contracts.Job{
            JobID: uuid.New().String(),
            UoW:   rule.UoWType,
            File:  contracts.File{
                ID:   content.ID.String(),
                Blob: contracts.Blob{Location: downloadURL},
            },
            Hints: rule.Params,
        }

        p.asyncRunner.Run(ctx, nil, job)
    }
    return nil
}

// 3. Run backfill (external app)
scanner := scan.New(adminSvc)
result, _ := scanner.Scan(ctx, scan.ScanOptions{
    Filters: admin.ContentFilters{
        Status:       stringPtr("uploaded"),
        DocumentType: stringPtr("image/*"),
        TenantID:     &tenantID,
    },
    Processor: jobCreator,
})

fmt.Printf("Created jobs for %d images\n", result.TotalProcessed)
```

## Performance

- **Batch size**: 100-1000 for most cases
- **Memory**: Proportional to batch size
- **Database queries**: `(total / batch_size)` queries
- **Concurrent processing**: Implement in your processor if needed

## Examples

Complete working examples:
- `examples/scan/main.go` - Full examples
- `examples/scan/processors/` - Processor implementations

Run:
```bash
go run ./examples/scan
```

## API Reference

See detailed API documentation:
- `pkg/simplecontent/scan/README.md` - Complete API docs
- `pkg/simplecontent/scan/processor.go` - Interface definition
- `pkg/simplecontent/scan/scanner.go` - Scanner implementation

## Summary

**simple-content provides:**
- ✅ Content query with filters
- ✅ Batch iteration
- ✅ Generic processor interface
- ✅ Progress tracking
- ✅ Error handling

**External apps provide:**
- ✅ Processing logic
- ✅ Tenant rules
- ✅ Event formatting
- ✅ Job creation
- ✅ Any custom processing

**Benefits:**
- Clean separation of concerns
- Maximum flexibility
- Reusable for any use case
- Easy to test
- Efficient batch processing
