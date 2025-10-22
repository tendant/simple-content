# Middleware Example

A complete example demonstrating the Simple Content middleware system for HTTP request/response processing.

## Features Demonstrated

- ✅ Request ID tracking
- ✅ Logging middleware
- ✅ Recovery from panics
- ✅ CORS handling
- ✅ Authentication
- ✅ Rate limiting
- ✅ Request size limits
- ✅ Metrics collection
- ✅ Request timeouts
- ✅ Middleware chaining
- ✅ Per-route middleware

## Running the Example

```bash
cd examples/middleware
go run main.go
```

Output:
```
🔧 Simple Content - Middleware Example
========================================

🚀 Server starting on http://localhost:8080

Endpoints:
  GET  /health              - Health check (no auth)
  GET  /api/v1/contents     - List contents (requires auth)
  POST /api/v1/contents     - Create content (requires auth)
```

## Example Requests

### 1. Health Check (Public, No Auth)

```bash
curl http://localhost:8080/health
```

Response:
```json
{
  "status": "ok",
  "time": "2025-01-22T10:30:00Z"
}
```

**Middleware applied:**
- Request ID
- Logging
- Recovery
- CORS

### 2. List Contents (Protected, Auth Required)

```bash
curl -H 'Authorization: Bearer demo-token' \
  http://localhost:8080/api/v1/contents
```

Response:
```json
{
  "contents": [],
  "count": 0
}
```

**Middleware applied:**
- Request ID
- Logging
- Recovery
- Rate limiting (60/min)
- Request size limit (10MB)
- CORS
- Authentication
- Metrics
- Timeout (30s)

### 3. Create Content (Protected)

```bash
curl -X POST http://localhost:8080/api/v1/contents \
  -H 'Authorization: Bearer demo-token' \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "test.txt",
    "document_type": "text"
  }'
```

Response:
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "owner_id": "00000000-0000-0000-0000-000000000001",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "name": "test.txt",
  "document_type": "text",
  "status": "created",
  "created_at": "2025-01-22T10:30:00Z"
}
```

### 4. View Metrics (Public)

```bash
curl http://localhost:8080/metrics
```

Response:
```json
{
  "total_requests": 15,
  "average_duration_ms": 12,
  "status_counts": {
    "200": 10,
    "201": 3,
    "401": 2
  },
  "recent_requests": [
    {
      "method": "GET",
      "path": "/api/v1/contents",
      "status_code": 200,
      "duration": 8000000,
      "size": 156,
      "timestamp": "2025-01-22T10:30:00Z"
    }
  ]
}
```

### 5. Missing Authentication

```bash
curl http://localhost:8080/api/v1/contents
```

Response (401):
```json
{
  "error": {
    "code": "unauthorized",
    "message": "Authentication required"
  }
}
```

### 6. Rate Limit Exceeded

After 60 requests in a minute:

Response (429):
```json
{
  "error": {
    "code": "rate_limit_exceeded",
    "message": "Rate limit exceeded. Maximum 60 requests per minute."
  }
}
```

Headers:
```
Retry-After: 60
```

## Server Logs

The middleware logging shows detailed request information:

```
[API] 2025/01/22 10:30:00 [abc-123] → GET /health
[API] 2025/01/22 10:30:00 [abc-123] ← 200 GET /health (2ms, 45 bytes)
[API] 2025/01/22 10:30:01 [def-456] → GET /api/v1/contents
[API] 2025/01/22 10:30:01 [def-456] ← 200 GET /api/v1/contents (8ms, 156 bytes)
[API] 2025/01/22 10:30:02 [ghi-789] → POST /api/v1/contents
[API] 2025/01/22 10:30:02 [ghi-789] ← 201 POST /api/v1/contents (15ms, 234 bytes)
```

## Middleware Architecture

### Public Routes
```
Request
  ↓
RequestIDMiddleware        (add request ID)
  ↓
RecoveryMiddleware         (catch panics)
  ↓
LoggingMiddleware          (log request/response)
  ↓
CORSMiddleware             (add CORS headers)
  ↓
Handler                    (process request)
```

### Protected Routes
```
Request
  ↓
RequestIDMiddleware        (add request ID)
  ↓
RecoveryMiddleware         (catch panics)
  ↓
LoggingMiddleware          (log request/response)
  ↓
RateLimitMiddleware        (60 req/min)
  ↓
RequestSizeLimitMiddleware (10MB max)
  ↓
CORSMiddleware             (add CORS headers)
  ↓
AuthenticationMiddleware   (validate token)
  ↓
MetricsMiddleware          (track metrics)
  ↓
TimeoutMiddleware          (30s timeout)
  ↓
Handler                    (process request)
```

## Key Concepts

### 1. Middleware Chaining

```go
chain := api.NewMiddlewareChain(
    api.RequestIDMiddleware,
    api.RecoveryMiddleware,
    api.LoggingMiddleware(logger),
)

handler := chain.Wrap(mux)
```

### 2. Per-Route Middleware

Different routes can have different middleware:

```go
// Public routes (minimal middleware)
r.Group(func(r chi.Router) {
    r.Use(publicChainWrapper)
    r.Get("/health", healthHandler)
})

// Protected routes (full middleware stack)
r.Group(func(r chi.Router) {
    r.Use(protectedChainWrapper)
    r.Route("/api/v1", apiRoutes)
})
```

### 3. Context Values

Middleware can add values to the request context:

```go
// In middleware
ctx := context.WithValue(r.Context(), api.UserIDKey, userID)

// In handler
userID, ok := r.Context().Value(api.UserIDKey).(uuid.UUID)
```

### 4. Response Wrapping

Middleware can inspect and modify responses:

```go
rw := newResponseWriter(w)
next.ServeHTTP(rw, r)
log.Printf("Status: %d, Size: %d", rw.statusCode, rw.bytesWritten)
```

## Testing Middleware

### Test Request ID

```bash
# Send request with custom ID
curl -H 'X-Request-ID: my-custom-id' \
  http://localhost:8080/health

# Check logs - should see "my-custom-id"
```

### Test Rate Limiting

```bash
# Send 70 requests quickly (exceeds 60/min limit)
for i in {1..70}; do
  curl -H 'Authorization: Bearer demo-token' \
    http://localhost:8080/api/v1/contents &
done
wait

# Last 10 should fail with 429
```

### Test Request Size Limit

```bash
# Send large request (>10MB)
dd if=/dev/zero bs=1M count=11 | \
  curl -X POST http://localhost:8080/api/v1/contents \
    -H 'Authorization: Bearer demo-token' \
    -H 'Content-Type: application/json' \
    --data-binary @-

# Should fail with 413
```

### Test Timeout

In the code, temporarily set timeout to 1ms to test:

```go
api.TimeoutMiddleware(1 * time.Millisecond)
```

Then make a request - should timeout with 504.

## Production Considerations

For production deployments:

### 1. Use Real Authentication

Replace the demo auth function with JWT validation:

```go
import "github.com/golang-jwt/jwt/v5"

func jwtAuthFunc(r *http.Request) (userID, tenantID uuid.UUID, err error) {
    tokenString := r.Header.Get("Authorization")
    // Parse and validate JWT
    token, err := jwt.Parse(tokenString, keyFunc)
    // Extract user/tenant from claims
    return userID, tenantID, nil
}
```

### 2. Configure Rate Limiting

Adjust limits based on your needs:

```go
// Different limits for different routes
publicLimiter := api.NewRateLimiter(100)   // 100/min
apiLimiter := api.NewRateLimiter(1000)     // 1000/min
adminLimiter := api.NewRateLimiter(10000)  // 10000/min
```

### 3. Add Real Metrics

Integrate with Prometheus or similar:

```go
import "github.com/prometheus/client_golang/prometheus"

type PrometheusMetrics struct {
    // Define metrics
}

func (m *PrometheusMetrics) RecordRequest(...) {
    // Record to Prometheus
}
```

### 4. Configure CORS

Set specific allowed origins:

```go
api.CORSMiddleware(
    []string{
        "https://app.example.com",
        "https://admin.example.com",
    },
    []string{"GET", "POST", "PUT", "DELETE"},
    []string{"Content-Type", "Authorization"},
)
```

### 5. Add Security Headers

```go
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("Strict-Transport-Security", "max-age=31536000")
        next.ServeHTTP(w, r)
    })
}
```

## Learn More

- **[Middleware Guide](../../MIDDLEWARE_GUIDE.md)** - Comprehensive middleware documentation
- **[Hooks Guide](../../HOOKS_GUIDE.md)** - Service-level extensibility
- **[Quickstart](../../QUICKSTART.md)** - Getting started guide

---

**Ready to build production APIs with middleware?** Check out the [Middleware Guide](../../MIDDLEWARE_GUIDE.md)! 🚀
