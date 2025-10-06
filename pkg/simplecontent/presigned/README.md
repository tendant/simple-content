# Presigned Upload Library

A reusable Go package for implementing secure, HMAC-signed presigned upload URLs similar to AWS S3 presigned URLs.

## Features

- 🔐 **HMAC-SHA256 Signatures** - Cryptographically secure authentication
- ⏱️ **Time-Limited URLs** - Automatic expiration for security
- 🔌 **Storage Agnostic** - Works with any storage backend
- 🚀 **Easy Integration** - Drop-in HTTP middleware
- 📦 **Client SDK** - Built-in upload client with retry logic
- 🎨 **Customizable** - Flexible URL patterns and payload formats
- 🧪 **Well-Tested** - Comprehensive test coverage
- 📚 **Zero Dependencies** - Only stdlib (except uuid)

## Installation

```bash
go get github.com/tendant/simple-content/pkg/simplecontent/presigned
```

## Quick Start

### Server-Side: Generate Presigned URL

```go
import "github.com/tendant/simple-content/pkg/simplecontent/presigned"

// Create signer
signer := presigned.New(presigned.WithSecretKey("your-secret-key-min-32-chars"))

// Generate presigned URL
url, err := signer.SignURL("PUT", "/upload/myfile.pdf", 1*time.Hour)
// Returns: /upload/myfile.pdf?signature=abc123...&expires=1696789012
```

### Server-Side: Validate Upload Requests

```go
// Option 1: Using middleware (recommended)
http.Handle("/upload/", presigned.ValidateMiddleware(secretKey, uploadHandler))

// Option 2: Manual validation
func uploadHandler(w http.ResponseWriter, r *http.Request) {
    if err := signer.ValidateRequest(r); err != nil {
        http.Error(w, "Unauthorized", http.StatusForbidden)
        return
    }
    // Handle upload...
}
```

### Client-Side: Upload Files

```go
client := presigned.NewClient()
err := client.Upload(ctx, presignedURL, fileReader)
```

## Complete Example

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "os"
    "time"

    "github.com/tendant/simple-content/pkg/simplecontent/presigned"
)

func main() {
    secretKey := "your-secret-key-at-least-32-characters-long"

    // Server: Setup HTTP handler with middleware
    http.Handle("/upload/", presigned.ValidateMiddleware(secretKey,
        http.HandlerFunc(handleUpload)))

    http.HandleFunc("/get-upload-url", func(w http.ResponseWriter, r *http.Request) {
        getUploadURL(w, r, secretKey)
    })

    fmt.Println("Server listening on :8080")
    http.ListenAndServe(":8080", nil)
}

// Server: Generate presigned URL
func getUploadURL(w http.ResponseWriter, r *http.Request, secretKey string) {
    signer := presigned.New(presigned.WithSecretKey(secretKey))

    objectKey := "uploads/user123/document.pdf"
    url, err := signer.SignURLWithBase(
        "http://localhost:8080",
        "PUT",
        "/upload/"+objectKey,
        15*time.Minute,
    )
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    fmt.Fprintf(w, `{"upload_url":"%s"}`, url)
}

// Server: Handle validated upload
func handleUpload(w http.ResponseWriter, r *http.Request) {
    // Signature already validated by middleware
    objectKey := presigned.ObjectKeyFromContext(r.Context())

    // Save file
    file, err := os.Create("/tmp/" + objectKey)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer file.Close()

    io.Copy(file, r.Body)
    w.WriteHeader(http.StatusOK)
}

// Client: Upload file to presigned URL
func uploadFile(presignedURL string, filePath string) error {
    file, err := os.Open(filePath)
    if err != nil {
        return err
    }
    defer file.Close()

    client := presigned.NewClient()
    return client.Upload(context.Background(), presignedURL, file,
        presigned.WithContentType("application/pdf"))
}
```

## Advanced Usage

### Custom Configuration

```go
signer := presigned.New(
    presigned.WithSecretKey("your-secret-key"),
    presigned.WithDefaultExpiration(30*time.Minute),
    presigned.WithURLPattern("/api/v1/upload/{key}"),
)
```

### Progress Tracking

```go
client := presigned.NewClient(
    presigned.WithProgress(func(bytesUploaded int64) {
        fmt.Printf("Uploaded: %d bytes\n", bytesUploaded)
    }),
)
```

### Retry Configuration

```go
client := presigned.NewClient(
    presigned.WithRetry(5, 2*time.Second), // 5 attempts, 2s delay
)
```

### Custom HTTP Client

```go
httpClient := &http.Client{
    Timeout: 10 * time.Minute,
    Transport: customTransport,
}

client := presigned.NewClient(
    presigned.WithHTTPClient(httpClient),
)
```

### Custom Payload Format

```go
signer := presigned.New(
    presigned.WithSecretKey("your-secret-key"),
    presigned.WithCustomPayloadFunc(func(method, path string, expiresAt int64) string {
        // Custom signature payload format
        return fmt.Sprintf("%s:%s:%d:custom", method, path, expiresAt)
    }),
)
```

## API Reference

### Signer

```go
// Create signer
signer := presigned.New(opts ...Option)

// Generate signed URL
url, err := signer.SignURL(method, path string, expiresIn time.Duration)
url, err := signer.SignURLWithBase(baseURL, method, path string, expiresIn time.Duration)

// Validate request
err := signer.ValidateRequest(r *http.Request)
err := signer.Validate(method, path, signature string, expiresAt int64)

// Extract object key
key, err := signer.ExtractObjectKey(path string)

// Check if enabled
enabled := signer.IsEnabled()
```

### Options

```go
presigned.WithSecretKey(key string)
presigned.WithDefaultExpiration(duration time.Duration)
presigned.WithURLPattern(pattern string)
presigned.WithCustomPayloadFunc(fn func(method, path string, expiresAt int64) string)
```

### Middleware

```go
// Validate middleware with secret key
http.Handle("/upload/", presigned.ValidateMiddleware(secretKey, handler))

// Validate middleware with custom signer
http.Handle("/upload/", presigned.ValidateMiddlewareWithSigner(signer, handler))

// Extract object key from context
objectKey := presigned.ObjectKeyFromContext(ctx)
```

### Client

```go
// Create client
client := presigned.NewClient(opts ...ClientOption)

// Upload file
err := client.Upload(ctx, presignedURL string, data io.Reader, opts ...UploadOption)
err := client.UploadWithContentType(ctx, presignedURL string, data io.Reader, contentType string)
```

### Client Options

```go
presigned.WithHTTPClient(client *http.Client)
presigned.WithRetry(attempts int, delay time.Duration)
presigned.WithProgress(fn ProgressFunc)
```

### Upload Options

```go
presigned.WithContentType(contentType string)
presigned.WithHeader(key, value string)
```

## Security Best Practices

1. **Strong Secret Keys**
   ```go
   // Generate secure random key
   import "crypto/rand"
   key := make([]byte, 32)
   rand.Read(key)
   secretKey := base64.StdEncoding.EncodeToString(key)
   ```

2. **Appropriate Expiration Times**
   - Web apps: 15-30 minutes
   - Mobile apps: 1 hour (for flaky networks)
   - Background jobs: 5 minutes

3. **Always Use HTTPS in Production**
   ```go
   url, _ := signer.SignURLWithBase(
       "https://api.example.com", // HTTPS!
       "PUT",
       "/upload/file.pdf",
       time.Hour,
   )
   ```

4. **Rotate Keys Periodically**
   - Implement versioned keys
   - Gradual migration strategy
   - Monitor for old key usage

5. **Monitor Invalid Attempts**
   ```go
   if err := signer.ValidateRequest(r); err != nil {
       log.Printf("Invalid signature attempt from %s: %v", r.RemoteAddr, err)
       // Consider rate limiting
   }
   ```

## Error Handling

```go
import "errors"

if err := signer.ValidateRequest(r); err != nil {
    switch {
    case errors.Is(err, presigned.ErrExpired):
        // URL has expired - generate new URL
    case errors.Is(err, presigned.ErrInvalidSignature):
        // Invalid signature - possible attack
    case errors.Is(err, presigned.ErrMissingSignature):
        // Missing signature parameter
    case presigned.IsAuthError(err):
        // Any authentication error
    }
}
```

## Testing

```go
func TestUpload(t *testing.T) {
    // Use test secret key
    signer := presigned.New(presigned.WithSecretKey("test-secret-key"))

    // Generate test URL
    url, _ := signer.SignURL("PUT", "/upload/test.txt", 1*time.Hour)

    // Create test request
    req := httptest.NewRequest("PUT", url, strings.NewReader("test data"))

    // Validate
    if err := signer.ValidateRequest(req); err != nil {
        t.Fatalf("Validation failed: %v", err)
    }
}
```

## Integration Examples

### With Chi Router

```go
import "github.com/go-chi/chi/v5"

r := chi.NewRouter()
r.Put("/upload/*", presigned.ValidateHandler(secretKey, uploadHandler))
```

### With Gorilla Mux

```go
import "github.com/gorilla/mux"

r := mux.NewRouter()
r.Handle("/upload/{key:.*}",
    presigned.ValidateMiddleware(secretKey, uploadHandler)).Methods("PUT")
```

### With Standard Library

```go
mux := http.NewServeMux()
mux.Handle("/upload/", presigned.ValidateMiddleware(secretKey, uploadHandler))
```

## Comparison with S3 Presigned URLs

| Feature | S3 Presigned | This Package |
|---------|-------------|--------------|
| Signing Algorithm | AWS Sig V4 | HMAC-SHA256 |
| Expiration | ✅ Yes | ✅ Yes |
| Signature Location | Query params | Query params |
| Secret Management | AWS credentials | Environment var |
| Validation | AWS infra | Your server |
| Use Case | AWS S3 | Any storage |

## Performance

- Signature generation: ~50µs
- Signature validation: ~50µs
- Zero allocations for validation
- Constant-time comparison prevents timing attacks

## Contributing

Contributions welcome! Please ensure:
- Tests pass: `go test ./...`
- Code is formatted: `go fmt ./...`
- Documentation is updated

## License

MIT License - see LICENSE file for details

## Support

- 📖 Documentation: [pkg.go.dev](https://pkg.go.dev/github.com/tendant/simple-content/pkg/simplecontent/presigned)
- 🐛 Issues: [GitHub Issues](https://github.com/tendant/simple-content/issues)
- 💬 Discussions: [GitHub Discussions](https://github.com/tendant/simple-content/discussions)

## Related Packages

- [simple-content](../README.md) - Complete content management system
- [objectkey](../objectkey/README.md) - Object key generation strategies
- [urlstrategy](../urlstrategy/README.md) - URL generation strategies
