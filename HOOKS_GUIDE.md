# Hooks & Plugins Guide

Simple Content provides a powerful hook system that lets you extend functionality without modifying core code. Hooks allow you to inject custom logic at specific points in the content lifecycle.

## 📚 Table of Contents

- [Overview](#overview)
- [Available Hooks](#available-hooks)
- [Quick Start](#quick-start)
- [Common Use Cases](#common-use-cases)
- [Building Plugins](#building-plugins)
- [Best Practices](#best-practices)

## Overview

Hooks are functions called at specific lifecycle events. They allow you to:

- **Validate** content before operations
- **Transform** data during processing
- **Log** operations for audit trails
- **Track** metrics and analytics
- **Trigger** external systems (webhooks, notifications)
- **Enforce** business rules and policies

### Hook Types

**Before Hooks** - Run before an operation:
- Can validate and reject operations
- Can modify request data
- Can add context for after hooks

**After Hooks** - Run after an operation:
- Can trigger follow-up actions
- Can log results
- Can send notifications

**Event Hooks** - Run on specific events:
- Status changes
- Errors
- State transitions

## Available Hooks

### Content Lifecycle

```go
type Hooks struct {
    // Create content
    BeforeContentCreate  []BeforeContentCreateHook
    AfterContentCreate   []AfterContentCreateHook

    // Upload data
    BeforeContentUpload  []BeforeContentUploadHook
    AfterContentUpload   []AfterContentUploadHook

    // Download data
    BeforeContentDownload []BeforeContentDownloadHook
    AfterContentDownload []AfterContentDownloadHook

    // Delete content
    BeforeContentDelete  []BeforeContentDeleteHook
    AfterContentDelete   []AfterContentDeleteHook

    // Derived content
    BeforeDerivedCreate []BeforeDerivedCreateHook
    AfterDerivedCreate  []AfterDerivedCreateHook

    // Metadata
    BeforeMetadataSet []BeforeMetadataSetHook
    AfterMetadataSet  []AfterMetadataSetHook

    // Events
    OnStatusChange []StatusChangeHook
    OnError        []ErrorHook
}
```

## Quick Start

### Example 1: Logging Hook

```go
package main

import (
    "context"
    "log"

    "github.com/tendant/simple-content/pkg/simplecontent"
)

func main() {
    // Create hooks
    hooks := &simplecontent.Hooks{
        AfterContentCreate: []simplecontent.AfterContentCreateHook{
            func(hctx *simplecontent.HookContext, content *simplecontent.Content) error {
                log.Printf("✅ Content created: %s", content.ID)
                return nil
            },
        },
        AfterContentUpload: []simplecontent.AfterContentUploadHook{
            func(hctx *simplecontent.HookContext, contentID uuid.UUID, bytes int64) error {
                log.Printf("✅ Uploaded %d bytes to content %s", bytes, contentID)
                return nil
            },
        },
    }

    // Create service with hooks
    svc, err := simplecontent.New(
        simplecontent.WithRepository(repo),
        simplecontent.WithBlobStore("fs", backend),
        simplecontent.WithHooks(hooks),
    )

    // Now all operations will trigger hooks!
    content, _ := svc.UploadContent(ctx, req)
    // Output: ✅ Content created: 123e4567-...
    //         ✅ Uploaded 52428 bytes to content 123e4567-...
}
```

### Example 2: Validation Hook

```go
// Validate file size before upload
hooks := &simplecontent.Hooks{
    BeforeContentCreate: []simplecontent.BeforeContentCreateHook{
        func(hctx *simplecontent.HookContext, req *simplecontent.CreateContentRequest) error {
            // Enforce naming convention
            if !strings.HasPrefix(req.Name, "DOC-") {
                return fmt.Errorf("content name must start with 'DOC-'")
            }
            return nil
        },
    },
    BeforeContentUpload: []simplecontent.BeforeContentUploadHook{
        func(hctx *simplecontent.HookContext, contentID uuid.UUID, reader io.Reader) (io.Reader, error) {
            // Limit file size to 100MB
            maxSize := int64(100 * 1024 * 1024)
            limitedReader := io.LimitReader(reader, maxSize+1)

            // Count bytes
            counted := &countingReader{r: limitedReader}

            // Store in hook context for after hook
            hctx.Metadata["expected_size"] = maxSize

            return counted, nil
        },
    },
}
```

### Example 3: Automatic Thumbnail Generation

```go
// Automatically generate thumbnails after image upload
hooks := &simplecontent.Hooks{
    AfterContentUpload: []simplecontent.AfterContentUploadHook{
        func(hctx *simplecontent.HookContext, contentID uuid.UUID, bytes int64) error {
            // Get service from context (passed by application)
            svc := hctx.Metadata["service"].(simplecontent.Service)

            // Download the uploaded content
            reader, err := svc.DownloadContent(hctx.Context, contentID)
            if err != nil {
                return err
            }
            defer reader.Close()

            // Check if it's an image
            content, _ := svc.GetContent(hctx.Context, contentID)
            if !strings.HasPrefix(content.DocumentType, "image") {
                return nil // Skip non-images
            }

            // Generate thumbnail
            thumbnail := generateThumbnail(reader, 256, 256)

            // Upload as derived content
            _, err = svc.UploadDerivedContent(hctx.Context, simplecontent.UploadDerivedContentRequest{
                ParentID:       contentID,
                DerivationType: "thumbnail",
                Variant:        "thumbnail_256",
                Reader:         thumbnail,
            })

            return err
        },
    },
}
```

## Common Use Cases

### 1. Audit Logging

```go
type AuditLogger struct {
    db *sql.DB
}

func (a *AuditLogger) Hooks() *simplecontent.Hooks {
    return &simplecontent.Hooks{
        AfterContentCreate: []simplecontent.AfterContentCreateHook{
            func(hctx *simplecontent.HookContext, content *simplecontent.Content) error {
                return a.logAction(hctx.Context, "content.created", content.ID, content.OwnerID)
            },
        },
        AfterContentDelete: []simplecontent.AfterContentDeleteHook{
            func(hctx *simplecontent.HookContext, contentID uuid.UUID) error {
                return a.logAction(hctx.Context, "content.deleted", contentID, uuid.Nil)
            },
        },
        OnStatusChange: []simplecontent.StatusChangeHook{
            func(hctx *simplecontent.HookContext, id uuid.UUID, old, new simplecontent.ContentStatus) error {
                return a.logStatusChange(hctx.Context, id, old, new)
            },
        },
    }
}

func (a *AuditLogger) logAction(ctx context.Context, action string, contentID, userID uuid.UUID) error {
    _, err := a.db.ExecContext(ctx, `
        INSERT INTO audit_log (action, content_id, user_id, timestamp)
        VALUES ($1, $2, $3, NOW())
    `, action, contentID, userID)
    return err
}
```

### 2. Metrics & Analytics

```go
type MetricsCollector struct {
    prometheus *prometheus.Registry
    uploadCounter prometheus.Counter
    uploadSize    prometheus.Histogram
}

func (m *MetricsCollector) Hooks() *simplecontent.Hooks {
    return &simplecontent.Hooks{
        AfterContentUpload: []simplecontent.AfterContentUploadHook{
            func(hctx *simplecontent.HookContext, contentID uuid.UUID, bytes int64) error {
                m.uploadCounter.Inc()
                m.uploadSize.Observe(float64(bytes))
                return nil
            },
        },
        OnError: []simplecontent.ErrorHook{
            func(hctx *simplecontent.HookContext, operation string, err error) {
                // Track error rates
                m.errorCounter.WithLabelValues(operation).Inc()
            },
        },
    }
}
```

### 3. Webhook Notifications

```go
type WebhookNotifier struct {
    webhookURL string
    client     *http.Client
}

func (w *WebhookNotifier) Hooks() *simplecontent.Hooks {
    return &simplecontent.Hooks{
        AfterContentCreate: []simplecontent.AfterContentCreateHook{
            func(hctx *simplecontent.HookContext, content *simplecontent.Content) error {
                return w.notify("content.created", content)
            },
        },
        AfterDerivedCreate: []simplecontent.AfterDerivedCreateHook{
            func(hctx *simplecontent.HookContext, parent, derived *simplecontent.Content) error {
                return w.notify("derived.created", map[string]interface{}{
                    "parent":  parent.ID,
                    "derived": derived.ID,
                })
            },
        },
    }
}

func (w *WebhookNotifier) notify(event string, data interface{}) error {
    payload := map[string]interface{}{
        "event": event,
        "data":  data,
        "timestamp": time.Now(),
    }

    body, _ := json.Marshal(payload)
    _, err := w.client.Post(w.webhookURL, "application/json", bytes.NewReader(body))
    return err
}
```

### 4. Content Virus Scanning

```go
type VirusScanner struct {
    scannerAPI string
}

func (v *VirusScanner) Hooks() *simplecontent.Hooks {
    return &simplecontent.Hooks{
        BeforeContentUpload: []simplecontent.BeforeContentUploadHook{
            func(hctx *simplecontent.HookContext, contentID uuid.UUID, reader io.Reader) (io.Reader, error) {
                // Read content into buffer for scanning
                buf := new(bytes.Buffer)
                if _, err := io.Copy(buf, reader); err != nil {
                    return nil, err
                }

                // Scan for viruses
                if err := v.scanContent(buf.Bytes()); err != nil {
                    return nil, fmt.Errorf("virus detected: %w", err)
                }

                // Return clean content
                return bytes.NewReader(buf.Bytes()), nil
            },
        },
    }
}
```

### 5. Access Control

```go
type AccessControl struct {
    permissions PermissionService
}

func (ac *AccessControl) Hooks() *simplecontent.Hooks {
    return &simplecontent.Hooks{
        BeforeContentCreate: []simplecontent.BeforeContentCreateHook{
            func(hctx *simplecontent.HookContext, req *simplecontent.CreateContentRequest) error {
                userID := hctx.Metadata["user_id"].(uuid.UUID)
                if !ac.permissions.CanCreate(userID, req.TenantID) {
                    return fmt.Errorf("access denied: user %s cannot create content in tenant %s", userID, req.TenantID)
                }
                return nil
            },
        },
        BeforeContentDelete: []simplecontent.BeforeContentDeleteHook{
            func(hctx *simplecontent.HookContext, contentID uuid.UUID) error {
                userID := hctx.Metadata["user_id"].(uuid.UUID)
                if !ac.permissions.CanDelete(userID, contentID) {
                    return fmt.Errorf("access denied: user %s cannot delete content %s", userID, contentID)
                }
                return nil
            },
        },
    }
}
```

## Building Plugins

### Plugin Structure

```go
// Plugin interface
type Plugin interface {
    Name() string
    Version() string
    Hooks() *simplecontent.Hooks
    Initialize(config map[string]interface{}) error
}

// Example plugin
type ImageProcessingPlugin struct {
    config map[string]interface{}
}

func (p *ImageProcessingPlugin) Name() string {
    return "image-processing"
}

func (p *ImageProcessingPlugin) Version() string {
    return "1.0.0"
}

func (p *ImageProcessingPlugin) Initialize(config map[string]interface{}) error {
    p.config = config
    return nil
}

func (p *ImageProcessingPlugin) Hooks() *simplecontent.Hooks {
    return &simplecontent.Hooks{
        AfterContentUpload: []simplecontent.AfterContentUploadHook{
            p.processImage,
        },
    }
}

func (p *ImageProcessingPlugin) processImage(hctx *simplecontent.HookContext, contentID uuid.UUID, bytes int64) error {
    // Plugin implementation...
    return nil
}
```

### Plugin Registry

```go
type PluginRegistry struct {
    plugins []Plugin
    hooks   *simplecontent.Hooks
}

func NewPluginRegistry() *PluginRegistry {
    return &PluginRegistry{
        hooks: &simplecontent.Hooks{},
    }
}

func (r *PluginRegistry) Register(plugin Plugin) error {
    r.plugins = append(r.plugins, plugin)

    // Merge plugin hooks into registry hooks
    pluginHooks := plugin.Hooks()
    r.hooks.AfterContentCreate = append(r.hooks.AfterContentCreate, pluginHooks.AfterContentCreate...)
    r.hooks.AfterContentUpload = append(r.hooks.AfterContentUpload, pluginHooks.AfterContentUpload...)
    // ... merge other hooks

    return nil
}

func (r *PluginRegistry) Hooks() *simplecontent.Hooks {
    return r.hooks
}

// Usage
registry := NewPluginRegistry()
registry.Register(&ImageProcessingPlugin{})
registry.Register(&VirusScannerPlugin{})
registry.Register(&AuditLogPlugin{})

svc, _ := simplecontent.New(
    simplecontent.WithHooks(registry.Hooks()),
    // ... other options
)
```

## Best Practices

### 1. Hook Ordering

Hooks run in the order they're registered. Order matters!

```go
hooks := &simplecontent.Hooks{
    BeforeContentUpload: []simplecontent.BeforeContentUploadHook{
        virusScanHook,      // 1. Scan first
        compressionHook,    // 2. Then compress
        encryptionHook,     // 3. Finally encrypt
    },
}
```

### 2. Error Handling

Hooks should return meaningful errors:

```go
func validationHook(hctx *simplecontent.HookContext, req *simplecontent.CreateContentRequest) error {
    if req.Name == "" {
        return fmt.Errorf("content name is required")
    }
    return nil
}
```

### 3. Context Usage

Use `HookContext.Metadata` to pass data between hooks:

```go
beforeHook := func(hctx *simplecontent.HookContext, ...) error {
    hctx.Metadata["start_time"] = time.Now()
    return nil
}

afterHook := func(hctx *simplecontent.HookContext, ...) error {
    startTime := hctx.Metadata["start_time"].(time.Time)
    duration := time.Since(startTime)
    log.Printf("Operation took %v", duration)
    return nil
}
```

### 4. Async Operations

For slow operations, use goroutines:

```go
afterHook := func(hctx *simplecontent.HookContext, contentID uuid.UUID, bytes int64) error {
    go func() {
        // Run in background
        sendWebhook(contentID)
        generatePreview(contentID)
    }()
    return nil // Don't block
}
```

### 5. Stop Chain

Use `StopChain` to prevent subsequent hooks:

```go
func authCheckHook(hctx *simplecontent.HookContext, req *simplecontent.CreateContentRequest) error {
    if !isAuthenticated(hctx.Context) {
        hctx.StopChain = true
        return fmt.Errorf("authentication required")
    }
    return nil
}
```

## Next Steps

- 📖 See [examples/hooks/](../examples/hooks/) for complete examples
- 🔌 Check out [plugins/](../plugins/) for ready-to-use plugins
- 📚 Read [PLUGINS_CATALOG.md](./PLUGINS_CATALOG.md) for available plugins

---

**Ready to extend Simple Content? Start with the Quick Start examples above!** 🚀
