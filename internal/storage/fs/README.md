# File System Storage Backend

This package provides a file system implementation of the `storage.Backend` interface for the Simple Content Management System.

## Overview

The file system storage backend stores content directly on the local file system. It's a simple and efficient storage solution for single-server deployments or development environments.

## Configuration

The file system storage backend supports the following configuration options:

1. **Base Directory**: The directory where files will be stored
   - This can be an absolute path or a relative path
   - The directory will be created if it doesn't exist
   - Default: `./data/storage`

2. **URL Prefix**: An optional URL prefix for download/upload URLs
   - If provided, the backend will return URLs for upload/download operations
   - If not provided, direct upload/download will be required
   - Default: empty (direct access)

## Usage

### Initialization

```go
import (
    "github.com/tendant/simple-content/internal/storage/fs"
)

// Create a new file system backend
config := fs.Config{
    BaseDir:   "./data/storage",
    URLPrefix: "http://localhost:8080",
}
fsBackend, err := fs.NewFSBackend(config)
if err != nil {
    log.Fatalf("Failed to initialize file system storage: %v", err)
}
```

### Registration

Register the file system backend with the object service:

```go
// Register the file system backend with the object service
objectService.RegisterBackend("fs", fsBackend)

// Create a storage backend record in the database
storageBackendService.CreateStorageBackend(
    ctx,
    "fs-default",
    "fs",
    map[string]interface{}{
        "base_dir": config.BaseDir,
    },
)
```

### Storage Structure

Files are stored in the base directory with the object key as the path. For example, if the base directory is `./data/storage` and the object key is `content/123/file.txt`, the file will be stored at `./data/storage/content/123/file.txt`.

## Limitations

1. **Scalability**: Not suitable for distributed deployments
2. **Backup**: Requires external backup solutions
3. **URL Access**: Requires additional configuration for HTTP access to files
4. **Permissions**: May require careful management of file system permissions

## Benefits

1. **Simplicity**: Easy to set up and understand
2. **Performance**: Fast local access to files
3. **Debugging**: Easy to inspect stored files directly on the file system
4. **No External Dependencies**: Works without requiring external services
5. **Persistence**: Data persists across application restarts
