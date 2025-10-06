# Config Package for Library Users

If you're using `simple-content` as a **library** in your own application, you have two main approaches:

## TL;DR

- ✅ **Recommended**: Use `config.Load()` for convenience, ignore `Port` and `Environment`
- ✅ **Alternative**: Create service directly with `simplecontent.New()` for maximum control
- ❌ **Don't**: Use `config.LoadServerConfig()` - that's for `cmd/server-configured` only

## Two Configuration Types

### 1. ServiceConfig (For Library Users)

**Core service configuration** - database, storage, URL strategy, etc.

```go
// ServiceConfig contains ONLY service-level settings
type ServiceConfig struct {
    DatabaseType          string
    DatabaseURL           string
    DefaultStorageBackend string
    StorageBackends       []StorageBackendConfig
    URLStrategy           string
    // ... other service settings
}
```

### 2. ServerConfig (For cmd/server-configured)

**Extends ServiceConfig** with server-specific settings like Port and Environment.

```go
// ServerConfig = ServiceConfig + server infrastructure
type ServerConfig struct {
    ServiceConfig              // Embedded
    Port           string      // ← Only for cmd/server-configured
    Environment    string      // ← Only for cmd/server-configured
    EnableAdminAPI bool        // ← Only for cmd/server-configured
}
```

## Usage Patterns for Library Users

### Pattern 1: Direct Service Creation (Most Control)

Best when you want explicit control over all components:

```go
import (
    "github.com/tendant/simple-content/pkg/simplecontent"
    memoryrepo "github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
    memorystorage "github.com/tendant/simple-content/pkg/simplecontent/storage/memory"
)

func main() {
    // Create components
    repo := memoryrepo.New()
    store := memorystorage.New()

    // Build service
    svc, err := simplecontent.New(
        simplecontent.WithRepository(repo),
        simplecontent.WithBlobStore("memory", store),
    )

    // Use in YOUR application with YOUR port
    http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
        // Use svc.UploadContent(), etc.
    })
    http.ListenAndServe(":3000", nil) // YOUR port, not config.Port
}
```

### Pattern 2: Using config.Load() (Recommended)

Best for convenience while maintaining control:

```go
import "github.com/tendant/simple-content/pkg/simplecontent/config"

func main() {
    // Use config for convenience
    cfg, err := config.Load(
        // Service-level config (what you care about)
        config.WithDatabase("postgres", os.Getenv("DATABASE_URL")),
        config.WithFilesystemStorage("fs", "./data", "/api/v1", "secret"),
        config.WithDefaultStorage("fs"),
        config.WithStorageDelegatedURLs(),

        // Server-level config (IGNORED for library usage)
        // Don't set Port or Environment - you control your own server
    )

    // Build service from config
    svc, err := cfg.BuildService()

    // Use in YOUR application
    myApp := MyApp{contentService: svc}
    myApp.Start(":4000") // YOUR port
}
```

### Pattern 3: Environment Variables (Not Recommended for Libraries)

Avoid `config.LoadServerConfig()` in library usage:

```go
// ❌ Don't do this in library code
cfg, err := config.LoadServerConfig() // Loads Port, Environment, etc from env

// The Port field is irrelevant - you run your own server
// This creates confusion about which port is actually used
```

## Complete Example

```go
package main

import (
    "net/http"
    "os"

    "github.com/tendant/simple-content/pkg/simplecontent"
    "github.com/tendant/simple-content/pkg/simplecontent/config"
)

type MyApplication struct {
    contentService simplecontent.Service
    httpPort       string
}

func main() {
    // Build service using config (convenience)
    cfg, err := config.Load(
        config.WithDatabase("postgres", os.Getenv("DATABASE_URL")),
        config.WithFilesystemStorage("fs", "/var/data", "https://api.example.com", os.Getenv("FS_SECRET")),
        config.WithDefaultStorage("fs"),
        config.WithStorageDelegatedURLs(),
        config.WithObjectKeyGenerator("git-like"),
    )
    if err != nil {
        panic(err)
    }

    svc, err := cfg.BuildService()
    if err != nil {
        panic(err)
    }

    // Create YOUR application with YOUR port
    app := &MyApplication{
        contentService: svc,
        httpPort:       os.Getenv("MY_APP_PORT"), // YOUR env var, not cfg.Port
    }

    app.Start()
}

func (app *MyApplication) Start() {
    mux := http.NewServeMux()

    // YOUR handlers using the service
    mux.HandleFunc("/api/upload", app.handleUpload)
    mux.HandleFunc("/api/download", app.handleDownload)

    // YOUR server, YOUR port
    http.ListenAndServe(app.httpPort, mux)
}

func (app *MyApplication) handleUpload(w http.ResponseWriter, r *http.Request) {
    // Use app.contentService.UploadContent(), etc.
}

func (app *MyApplication) handleDownload(w http.ResponseWriter, r *http.Request) {
    // Use app.contentService.DownloadContent(), etc.
}
```

## Key Points

1. **ServiceConfig vs ServerConfig**
   - `ServiceConfig` = Core service settings (database, storage, URL strategy)
   - `ServerConfig` = ServiceConfig + infrastructure (Port, Environment)

2. **Library users should**:
   - Use `config.Load()` with service-level options
   - Ignore `WithPort()` and `WithEnvironment()` options
   - Control your own HTTP server and port
   - Call `cfg.BuildService()` to get the service

3. **Don't confuse**:
   - `config.Port` ≠ your application's port
   - `config.Environment` ≠ your application's environment
   - These are for `cmd/server-configured` only

4. **When to use each approach**:
   - Direct creation: Maximum control, testing, minimal dependencies
   - config.Load(): Convenience, complex storage setups, URL strategies
   - LoadServerConfig(): Only for `cmd/server-configured`

## See Also

- `/examples/library-usage` - Complete working example
- `/examples/basic` - Direct service creation example
- `/pkg/simplecontent/config/README.md` - Full config documentation
