# S3 Storage Backend

This package provides an implementation of the `storage.Backend` interface for AWS S3 and S3-compatible storage services like MinIO.

## Features

- Direct upload and download of objects
- Pre-signed URLs for client-side upload and download
- Server-side encryption support (AES256 and aws:kms)
- Support for S3-compatible services like MinIO
- Automatic bucket creation (optional)

## Configuration

The S3 backend can be configured with the following options:

```go
type Config struct {
    Region          string // AWS region
    Bucket          string // S3 bucket name
    AccessKeyID     string // AWS access key ID
    SecretAccessKey string // AWS secret access key
    Endpoint        string // Optional custom endpoint for S3-compatible services
    UseSSL          bool   // Use SSL for connections (default: true)
    UsePathStyle    bool   // Use path-style addressing (default: false)
    PresignDuration int    // Duration in seconds for presigned URLs (default: 3600)

    // Server-side encryption options
    EnableSSE    bool   // Enable server-side encryption
    SSEAlgorithm string // SSE algorithm (AES256 or aws:kms)
    SSEKMSKeyID  string // Optional KMS key ID for aws:kms algorithm

    // MinIO-specific options
    CreateBucketIfNotExist bool // Create bucket if it doesn't exist
}
```

## Usage

### Creating an S3 Backend

```go
// AWS S3 configuration
config := s3.Config{
    Region:          "us-east-1",
    Bucket:          "my-bucket",
    AccessKeyID:     "AKIAXXXXXXXXXXXXXXXX",
    SecretAccessKey: "XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
    EnableSSE:       true,
    SSEAlgorithm:    "AES256",
}

// Create the backend
backend, err := s3.NewS3Backend(config)
if err != nil {
    log.Fatalf("Failed to create S3 backend: %v", err)
}
```

### Using with MinIO

```go
// MinIO configuration
config := s3.Config{
    Region:                "us-east-1",
    Bucket:                "my-bucket",
    AccessKeyID:           "minioadmin",
    SecretAccessKey:       "minioadmin",
    Endpoint:              "http://localhost:9000",
    UseSSL:                false,
    UsePathStyle:          true,
    CreateBucketIfNotExist: true,
}

// Create the backend
backend, err := s3.NewS3Backend(config)
if err != nil {
    log.Fatalf("Failed to create MinIO backend: %v", err)
}
```

### Uploading an Object

```go
// Direct upload
file, err := os.Open("example.txt")
if err != nil {
    log.Fatalf("Failed to open file: %v", err)
}
defer file.Close()

err = backend.Upload(context.Background(), "path/to/object.txt", file)
if err != nil {
    log.Fatalf("Upload failed: %v", err)
}

// Get pre-signed URL for client-side upload
uploadURL, err := backend.GetUploadURL(context.Background(), "path/to/object.txt")
if err != nil {
    log.Fatalf("Failed to get upload URL: %v", err)
}
fmt.Printf("Upload URL: %s\n", uploadURL)
```

### Downloading an Object

```go
// Direct download
reader, err := backend.Download(context.Background(), "path/to/object.txt")
if err != nil {
    log.Fatalf("Download failed: %v", err)
}
defer reader.Close()

// Save to file
file, err := os.Create("downloaded.txt")
if err != nil {
    log.Fatalf("Failed to create file: %v", err)
}
defer file.Close()

_, err = io.Copy(file, reader)
if err != nil {
    log.Fatalf("Failed to write file: %v", err)
}

// Get pre-signed URL for client-side download
downloadURL, err := backend.GetDownloadURL(context.Background(), "path/to/object.txt")
if err != nil {
    log.Fatalf("Failed to get download URL: %v", err)
}
fmt.Printf("Download URL: %s\n", downloadURL)
```

### Deleting an Object

```go
err = backend.Delete(context.Background(), "path/to/object.txt")
if err != nil {
    log.Fatalf("Delete failed: %v", err)
}
```
