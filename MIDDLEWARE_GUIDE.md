# Middleware Guide

Simple Content provides a powerful middleware system for HTTP request/response processing. Middleware allows you to intercept, modify, and enhance HTTP operations without changing handler code.

## 📚 Table of Contents

- [Overview](#overview)
- [Built-in Middleware](#built-in-middleware)
- [Quick Start](#quick-start)
- [Common Use Cases](#common-use-cases)
- [Custom Middleware](#custom-middleware)
- [Best Practices](#best-practices)

## Overview

Middleware are functions that wrap HTTP handlers, allowing you to:

- **Log** requests and responses
- **Authenticate** and authorize users
- **Rate limit** API usage
- **Compress** responses
- **Add headers** (CORS, security, etc.)
- **Validate** request data
- **Track metrics** and performance
- **Handle errors** consistently

### Middleware Signature

```go
type Middleware func(http.Handler) http.Handler
```

Middleware wraps an existing handler and returns a new handler with added functionality.

## Built-in Middleware

Simple Content includes 14 production-ready middleware:

### 1. Request ID
Adds unique ID to each request for tracing:

```go
import "github.com/tendant/simple-content/pkg/simplecontent/api"

middleware := api.RequestIDMiddleware
```

**Features:**
- Generates UUID for each request
- Uses `X-Request-ID` header if provided
- Adds ID to response headers
- Available in context as `api.RequestIDKey`

### 2. Logging
Comprehensive request/response logging:

```go
logger := log.New(os.Stdout, "[API] ", log.LstdFlags)
middleware := api.LoggingMiddleware(logger)
```

**Logs:**
- Request method and path
- Response status code
- Request duration
- Response size
- Request ID (if available)

**Example output:**
```
[API] [abc-123] → GET /api/v1/contents
[API] [abc-123] ← 200 GET /api/v1/contents (45ms, 1024 bytes)
```

### 3. Recovery
Recovers from panics and returns 500 error:

```go
middleware := api.RecoveryMiddleware
```

**Features:**
- Catches panics in handlers
- Logs panic details
- Returns structured JSON error
- Prevents server crashes

### 4. CORS
Handles Cross-Origin Resource Sharing:

```go
middleware := api.CORSMiddleware(
    []string{"https://example.com", "https://app.example.com"}, // Allowed origins
    []string{"GET", "POST", "PUT", "DELETE"},                   // Allowed methods
    []string{"Content-Type", "Authorization"},                  // Allowed headers
)
```

**Features:**
- Configurable allowed origins
- Preflight request handling
- Credential support
- Wildcard origin support (`*`)

### 5. Rate Limiting
Token bucket rate limiting:

```go
limiter := api.NewRateLimiter(60) // 60 requests per minute
middleware := limiter.Middleware
```

**Features:**
- Per-IP rate limiting
- Automatic token refill
- 429 status on limit exceeded
- `Retry-After` header

### 6. Request Size Limit
Limits request body size:

```go
middleware := api.RequestSizeLimitMiddleware(10 * 1024 * 1024) // 10MB
```

**Features:**
- Prevents memory exhaustion
- Returns 413 on oversized requests
- Configurable limit

### 7. Authentication
Validates authentication tokens:

```go
authFunc := func(r *http.Request) (userID, tenantID uuid.UUID, err error) {
    token := r.Header.Get("Authorization")
    // Validate token and extract user/tenant IDs
    return userID, tenantID, nil
}

middleware := api.AuthenticationMiddleware(authFunc)
```

**Features:**
- Custom authentication logic
- Adds user/tenant to context
- Returns 401 on failure
- Flexible token validation

### 8. Compression
Gzip response compression:

```go
middleware := api.CompressionMiddleware
```

**Features:**
- Automatic gzip compression
- Client acceptance detection
- Reduces bandwidth usage

### 9. Metrics
Tracks request metrics:

```go
type MyMetrics struct{}

func (m *MyMetrics) RecordRequest(method, path string, statusCode int, duration time.Duration, size int64) {
    // Record to Prometheus, StatsD, etc.
}

collector := &MyMetrics{}
middleware := api.MetricsMiddleware(collector)
```

**Tracks:**
- Request count by method/path
- Response time
- Status codes
- Response sizes

### 10. Validation
Validates request data:

```go
type MyValidator struct{}

func (v *MyValidator) Validate(r *http.Request) error {
    // Validate request data
    return nil
}

validator := &MyValidator{}
middleware := api.ValidationMiddleware(validator)
```

**Features:**
- Schema validation
- Returns 400 on invalid data
- Custom validation logic

### 11. Cache Control
Adds cache headers:

```go
middleware := api.CacheMiddleware(3600) // 1 hour
```

**Features:**
- Automatic cache headers for GET
- Configurable max-age
- Public cache control

### 12. Timeout
Request timeout handling:

```go
middleware := api.TimeoutMiddleware(30 * time.Second)
```

**Features:**
- Configurable timeout
- Context cancellation
- 504 status on timeout

### 13. Body Logging
Logs request/response bodies (debugging):

```go
logger := log.New(os.Stdout, "[BODY] ", log.LstdFlags)
middleware := api.BodyLoggingMiddleware(logger)
```

**⚠️ Warning:** Only use in development. Logs sensitive data!

## Quick Start

### Example 1: Basic Middleware Chain

```go
package main

import (
    "log"
    "net/http"
    "os"

    "github.com/tendant/simple-content/pkg/simplecontent/api"
)

func main() {
    // Create handler
    mux := http.NewServeMux()
    mux.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    })

    // Create middleware chain
    chain := api.NewMiddlewareChain(
        api.RequestIDMiddleware,
        api.RecoveryMiddleware,
        api.LoggingMiddleware(log.New(os.Stdout, "[API] ", log.LstdFlags)),
    )

    // Wrap handler with middleware
    handler := chain.Wrap(mux)

    // Start server
    log.Println("Server starting on :8080")
    http.ListenAndServe(":8080", handler)
}
```

### Example 2: Production Middleware Stack

```go
// Create comprehensive middleware stack
logger := log.New(os.Stdout, "[API] ", log.LstdFlags)
limiter := api.NewRateLimiter(100) // 100 req/min

chain := api.NewMiddlewareChain(
    api.RequestIDMiddleware,                              // 1. Add request ID
    api.RecoveryMiddleware,                               // 2. Recover from panics
    api.LoggingMiddleware(logger),                        // 3. Log requests
    limiter.Middleware,                                   // 4. Rate limiting
    api.RequestSizeLimitMiddleware(10 * 1024 * 1024),    // 5. Limit body size (10MB)
    api.CORSMiddleware(                                   // 6. CORS headers
        []string{"*"},
        []string{"GET", "POST", "PUT", "DELETE"},
        []string{"Content-Type", "Authorization"},
    ),
    api.TimeoutMiddleware(30 * time.Second),             // 7. Request timeout
)

handler := chain.Wrap(mux)
```

### Example 3: Adding Middleware Dynamically

```go
chain := api.NewMiddlewareChain()

// Add middleware conditionally
if os.Getenv("ENVIRONMENT") == "development" {
    chain.Then(api.BodyLoggingMiddleware(logger))
}

if os.Getenv("ENABLE_AUTH") == "true" {
    chain.Then(api.AuthenticationMiddleware(authFunc))
}

chain.Then(api.RequestIDMiddleware)
chain.Then(api.RecoveryMiddleware)

handler := chain.Wrap(mux)
```

## Common Use Cases

### 1. API Gateway Setup

```go
// Full API gateway middleware stack
chain := api.NewMiddlewareChain(
    // Security
    api.RecoveryMiddleware,
    api.RequestSizeLimitMiddleware(5 * 1024 * 1024),

    // Observability
    api.RequestIDMiddleware,
    api.LoggingMiddleware(logger),
    api.MetricsMiddleware(metricsCollector),

    // Access control
    api.CORSMiddleware(allowedOrigins, methods, headers),
    api.AuthenticationMiddleware(jwtValidator),
    api.NewRateLimiter(1000).Middleware,

    // Performance
    api.CompressionMiddleware,
    api.CacheMiddleware(3600),
    api.TimeoutMiddleware(60 * time.Second),
)
```

### 2. Development vs Production

```go
func createMiddlewareChain(env string) *api.MiddlewareChain {
    chain := api.NewMiddlewareChain(
        api.RequestIDMiddleware,
        api.RecoveryMiddleware,
        api.LoggingMiddleware(logger),
    )

    if env == "development" {
        // Development-only middleware
        chain.Then(api.BodyLoggingMiddleware(logger))
        chain.Then(api.CORSMiddleware([]string{"*"}, nil, nil))
    } else {
        // Production middleware
        chain.Then(api.NewRateLimiter(100).Middleware)
        chain.Then(api.AuthenticationMiddleware(authFunc))
        chain.Then(api.CompressionMiddleware)
        chain.Then(api.CacheMiddleware(3600))
    }

    return chain
}
```

### 3. JWT Authentication

```go
import (
    "github.com/golang-jwt/jwt/v5"
)

func jwtAuthMiddleware(secretKey string) api.Middleware {
    authFunc := func(r *http.Request) (userID, tenantID uuid.UUID, err error) {
        tokenString := r.Header.Get("Authorization")
        if tokenString == "" {
            return uuid.Nil, uuid.Nil, fmt.Errorf("missing token")
        }

        // Parse JWT
        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            return []byte(secretKey), nil
        })
        if err != nil {
            return uuid.Nil, uuid.Nil, err
        }

        if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
            userID, _ = uuid.Parse(claims["user_id"].(string))
            tenantID, _ = uuid.Parse(claims["tenant_id"].(string))
            return userID, tenantID, nil
        }

        return uuid.Nil, uuid.Nil, fmt.Errorf("invalid token")
    }

    return api.AuthenticationMiddleware(authFunc)
}
```

### 4. Prometheus Metrics

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

type PrometheusMetrics struct {
    requestCount    *prometheus.CounterVec
    requestDuration *prometheus.HistogramVec
    requestSize     *prometheus.HistogramVec
}

func NewPrometheusMetrics() *PrometheusMetrics {
    return &PrometheusMetrics{
        requestCount: promauto.NewCounterVec(
            prometheus.CounterOpts{
                Name: "http_requests_total",
                Help: "Total HTTP requests",
            },
            []string{"method", "path", "status"},
        ),
        requestDuration: promauto.NewHistogramVec(
            prometheus.HistogramOpts{
                Name: "http_request_duration_seconds",
                Help: "HTTP request duration",
            },
            []string{"method", "path"},
        ),
        requestSize: promauto.NewHistogramVec(
            prometheus.HistogramOpts{
                Name: "http_response_size_bytes",
                Help: "HTTP response size",
            },
            []string{"method", "path"},
        ),
    }
}

func (m *PrometheusMetrics) RecordRequest(method, path string, statusCode int, duration time.Duration, size int64) {
    m.requestCount.WithLabelValues(method, path, fmt.Sprintf("%d", statusCode)).Inc()
    m.requestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
    m.requestSize.WithLabelValues(method, path).Observe(float64(size))
}

// Usage
metrics := NewPrometheusMetrics()
chain := api.NewMiddlewareChain(
    api.MetricsMiddleware(metrics),
)
```

### 5. Request Validation

```go
type ContentValidator struct{}

func (v *ContentValidator) Validate(r *http.Request) error {
    // Only validate POST/PUT requests
    if r.Method != "POST" && r.Method != "PUT" {
        return nil
    }

    // Check content type
    contentType := r.Header.Get("Content-Type")
    if contentType != "application/json" {
        return fmt.Errorf("content-type must be application/json")
    }

    // Parse and validate body
    var body map[string]interface{}
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
        return fmt.Errorf("invalid JSON: %w", err)
    }

    // Restore body for handler
    bodyBytes, _ := json.Marshal(body)
    r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

    // Validate required fields
    if _, ok := body["name"]; !ok {
        return fmt.Errorf("missing required field: name")
    }

    return nil
}

// Usage
validator := &ContentValidator{}
chain := api.NewMiddlewareChain(
    api.ValidationMiddleware(validator),
)
```

## Custom Middleware

### Creating Custom Middleware

```go
// Custom middleware template
func MyCustomMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Before request processing
        // - Inspect request
        // - Modify request
        // - Add to context

        // Process request
        next.ServeHTTP(w, r)

        // After request processing
        // - Inspect response
        // - Add headers
        // - Log results
    })
}
```

### Example: Tenant Isolation

```go
func TenantIsolationMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Extract tenant ID from subdomain
        host := r.Host
        parts := strings.Split(host, ".")
        if len(parts) < 2 {
            http.Error(w, "Invalid tenant", http.StatusBadRequest)
            return
        }

        tenantName := parts[0]
        tenantID, err := resolveTenantID(tenantName)
        if err != nil {
            http.Error(w, "Unknown tenant", http.StatusNotFound)
            return
        }

        // Add tenant to context
        ctx := context.WithValue(r.Context(), api.TenantIDKey, tenantID)

        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### Example: API Versioning

```go
func APIVersionMiddleware(requiredVersion string) api.Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            version := r.Header.Get("API-Version")
            if version == "" {
                version = "v1" // Default
            }

            if version != requiredVersion {
                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusBadRequest)
                json.NewEncoder(w).Encode(map[string]interface{}{
                    "error": map[string]interface{}{
                        "code":    "version_mismatch",
                        "message": fmt.Sprintf("API version %s required", requiredVersion),
                    },
                })
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

### Example: Security Headers

```go
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Add security headers
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        w.Header().Set("Content-Security-Policy", "default-src 'self'")

        next.ServeHTTP(w, r)
    })
}
```

## Best Practices

### 1. Middleware Order Matters

Apply middleware in the correct order:

```go
// ✅ CORRECT ORDER
chain := api.NewMiddlewareChain(
    api.RecoveryMiddleware,              // 1. Catch panics first
    api.RequestIDMiddleware,             // 2. Add tracing
    api.LoggingMiddleware(logger),       // 3. Log with request ID
    api.AuthenticationMiddleware(auth),  // 4. Auth before business logic
    api.ValidationMiddleware(validator), // 5. Validate before processing
)

// ❌ INCORRECT ORDER
chain := api.NewMiddlewareChain(
    api.LoggingMiddleware(logger),       // No request ID yet!
    api.RequestIDMiddleware,             // Too late
    api.ValidationMiddleware(validator), // Before auth - security issue!
    api.AuthenticationMiddleware(auth),  // Auth should be earlier
)
```

### 2. Performance Considerations

- Place expensive middleware last (after early exits)
- Use buffering carefully (memory usage)
- Avoid blocking operations in middleware
- Consider caching expensive validations

```go
// ✅ GOOD: Early exits before expensive operations
chain := api.NewMiddlewareChain(
    api.RecoveryMiddleware,         // Fast
    api.RequestIDMiddleware,        // Fast
    api.AuthenticationMiddleware,   // Fast (can exit early)
    api.RateLimiter,               // Fast (can exit early)
    api.CompressionMiddleware,     // Expensive (but only if request proceeds)
)
```

### 3. Context Usage

Use context for passing data between middleware:

```go
// Store data in context
ctx := context.WithValue(r.Context(), api.UserIDKey, userID)

// Retrieve in handler
userID, ok := r.Context().Value(api.UserIDKey).(uuid.UUID)
```

### 4. Error Handling

Return consistent error responses:

```go
func handleError(w http.ResponseWriter, code string, message string, status int) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "error": map[string]interface{}{
            "code":    code,
            "message": message,
        },
    })
}
```

### 5. Testing Middleware

Test middleware in isolation:

```go
func TestMyMiddleware(t *testing.T) {
    // Create test handler
    called := false
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        called = true
        w.WriteHeader(http.StatusOK)
    })

    // Wrap with middleware
    wrapped := MyMiddleware(handler)

    // Create test request
    req := httptest.NewRequest("GET", "/test", nil)
    rr := httptest.NewRecorder()

    // Execute
    wrapped.ServeHTTP(rr, req)

    // Assert
    assert.True(t, called)
    assert.Equal(t, http.StatusOK, rr.Code)
}
```

## Integration with Simple Content

### Using Middleware with Content Handlers

```go
import (
    "github.com/go-chi/chi/v5"
    "github.com/tendant/simple-content/pkg/simplecontent"
    "github.com/tendant/simple-content/pkg/simplecontent/api"
)

func setupServer(svc simplecontent.Service) http.Handler {
    // Create router
    r := chi.NewRouter()

    // Create content handler
    contentHandler := api.NewContentHandler(svc)

    // Create middleware chain
    logger := log.New(os.Stdout, "[API] ", log.LstdFlags)
    chain := api.NewMiddlewareChain(
        api.RequestIDMiddleware,
        api.RecoveryMiddleware,
        api.LoggingMiddleware(logger),
        api.CORSMiddleware([]string{"*"}, nil, nil),
    )

    // Apply middleware to routes
    r.Group(func(r chi.Router) {
        r.Use(chain.Wrap) // Apply to all routes in this group

        r.Route("/api/v1", func(r chi.Router) {
            r.Route("/contents", func(r chi.Router) {
                r.Post("/", contentHandler.CreateContent)
                r.Get("/{id}", contentHandler.GetContent)
                r.Put("/{id}", contentHandler.UpdateContent)
                r.Delete("/{id}", contentHandler.DeleteContent)
            })
        })
    })

    return r
}
```

### Per-Route Middleware

```go
// Different middleware for different routes
r.Route("/api/v1", func(r chi.Router) {
    // Public routes (no auth)
    r.Group(func(r chi.Router) {
        r.Use(api.RequestIDMiddleware)
        r.Use(api.LoggingMiddleware(logger))

        r.Get("/health", healthHandler)
        r.Get("/version", versionHandler)
    })

    // Authenticated routes
    r.Group(func(r chi.Router) {
        r.Use(api.RequestIDMiddleware)
        r.Use(api.AuthenticationMiddleware(authFunc))
        r.Use(api.LoggingMiddleware(logger))

        r.Route("/contents", contentRoutes)
    })

    // Admin routes (extra validation)
    r.Group(func(r chi.Router) {
        r.Use(api.RequestIDMiddleware)
        r.Use(api.AuthenticationMiddleware(authFunc))
        r.Use(adminAuthorizationMiddleware)
        r.Use(api.LoggingMiddleware(logger))

        r.Route("/admin", adminRoutes)
    })
})
```

## Next Steps

- Explore built-in middleware options
- Create custom middleware for your use cases
- Integrate with monitoring/logging systems
- Test middleware thoroughly
- Review [HOOKS_GUIDE.md](./HOOKS_GUIDE.md) for service-level extensibility

---

**Ready to enhance your API with middleware?** Start with the Quick Start examples above! 🚀
