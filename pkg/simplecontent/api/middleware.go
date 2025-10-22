package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Middleware is a function that wraps an http.Handler
type Middleware func(http.Handler) http.Handler

// MiddlewareChain represents a chain of middleware functions
type MiddlewareChain struct {
	middlewares []Middleware
}

// NewMiddlewareChain creates a new middleware chain
func NewMiddlewareChain(middlewares ...Middleware) *MiddlewareChain {
	return &MiddlewareChain{
		middlewares: middlewares,
	}
}

// Then adds middleware to the chain
func (c *MiddlewareChain) Then(m Middleware) *MiddlewareChain {
	c.middlewares = append(c.middlewares, m)
	return c
}

// Wrap applies all middleware in the chain to the given handler
func (c *MiddlewareChain) Wrap(handler http.Handler) http.Handler {
	// Apply middleware in reverse order so the first middleware added is the outermost
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		handler = c.middlewares[i](handler)
	}
	return handler
}

// ResponseWriter wrapper that captures status code and response size
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
	body         []byte // Optionally capture response body
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK, // Default status
	}
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += int64(n)
	return n, err
}

// ResponseCapturingWriter captures the response body for inspection
type responseCapturer struct {
	*responseWriter
	captureBody bool
}

func newResponseCapturer(w http.ResponseWriter, capture bool) *responseCapturer {
	return &responseCapturer{
		responseWriter: newResponseWriter(w),
		captureBody:    capture,
	}
}

func (rc *responseCapturer) Write(b []byte) (int, error) {
	if rc.captureBody {
		rc.body = append(rc.body, b...)
	}
	return rc.responseWriter.Write(b)
}

// Context keys for middleware
type contextKey string

const (
	RequestIDKey  contextKey = "request_id"
	RequestTimeKey contextKey = "request_time"
	UserIDKey     contextKey = "user_id"
	TenantIDKey   contextKey = "tenant_id"
)

// Built-in Middleware Functions

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Add to context
		ctx := context.WithValue(r.Context(), RequestIDKey, requestID)

		// Add to response headers
		w.Header().Set("X-Request-ID", requestID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// LoggingMiddleware logs HTTP requests and responses
func LoggingMiddleware(logger *log.Logger) Middleware {
	if logger == nil {
		logger = log.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status code
			rw := newResponseWriter(w)

			// Add start time to context
			ctx := context.WithValue(r.Context(), RequestTimeKey, start)

			// Get request ID from context if available
			requestID := ""
			if id, ok := ctx.Value(RequestIDKey).(string); ok {
				requestID = id
			}

			// Log request
			logger.Printf("[%s] → %s %s", requestID, r.Method, r.URL.Path)

			// Process request
			next.ServeHTTP(rw, r.WithContext(ctx))

			// Log response
			duration := time.Since(start)
			logger.Printf("[%s] ← %d %s %s (%v, %d bytes)",
				requestID,
				rw.statusCode,
				r.Method,
				r.URL.Path,
				duration,
				rw.bytesWritten,
			)
		})
	}
}

// RecoveryMiddleware recovers from panics and returns 500 error
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				requestID := ""
				if id, ok := r.Context().Value(RequestIDKey).(string); ok {
					requestID = id
				}

				log.Printf("[%s] PANIC: %v", requestID, err)

				// Return 500 error
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": map[string]interface{}{
						"code":       "internal_error",
						"message":    "An internal server error occurred",
						"request_id": requestID,
					},
				})
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// CORSMiddleware handles CORS headers
func CORSMiddleware(allowedOrigins []string, allowedMethods []string, allowedHeaders []string) Middleware {
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"*"}
	}
	if len(allowedMethods) == 0 {
		allowedMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	}
	if len(allowedHeaders) == 0 {
		allowedHeaders = []string{"Content-Type", "Authorization", "X-Request-ID"}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			allowed := false
			for _, allowedOrigin := range allowedOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					allowed = true
					break
				}
			}

			if allowed {
				if origin == "" {
					origin = allowedOrigins[0]
				}
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", joinStrings(allowedMethods, ", "))
				w.Header().Set("Access-Control-Allow-Headers", joinStrings(allowedHeaders, ", "))
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitMiddleware implements simple rate limiting (token bucket)
type RateLimiter struct {
	requestsPerMinute int
	tokens            map[string]*tokenBucket
}

type tokenBucket struct {
	tokens     int
	lastRefill time.Time
}

func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	return &RateLimiter{
		requestsPerMinute: requestsPerMinute,
		tokens:            make(map[string]*tokenBucket),
	}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Use IP address as key (in production, you might want to use user ID)
		key := r.RemoteAddr

		// Get or create token bucket
		bucket, exists := rl.tokens[key]
		if !exists {
			bucket = &tokenBucket{
				tokens:     rl.requestsPerMinute,
				lastRefill: time.Now(),
			}
			rl.tokens[key] = bucket
		}

		// Refill tokens based on time elapsed
		now := time.Now()
		elapsed := now.Sub(bucket.lastRefill)
		tokensToAdd := int(elapsed.Minutes() * float64(rl.requestsPerMinute))
		if tokensToAdd > 0 {
			bucket.tokens = min(rl.requestsPerMinute, bucket.tokens+tokensToAdd)
			bucket.lastRefill = now
		}

		// Check if request is allowed
		if bucket.tokens <= 0 {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]interface{}{
					"code":    "rate_limit_exceeded",
					"message": fmt.Sprintf("Rate limit exceeded. Maximum %d requests per minute.", rl.requestsPerMinute),
				},
			})
			return
		}

		// Consume token
		bucket.tokens--

		next.ServeHTTP(w, r)
	})
}

// RequestSizeLimitMiddleware limits the size of request bodies
func RequestSizeLimitMiddleware(maxBytes int64) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Limit request body size
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

			next.ServeHTTP(w, r)
		})
	}
}

// AuthenticationMiddleware validates authentication tokens
type AuthenticationFunc func(r *http.Request) (userID, tenantID uuid.UUID, err error)

func AuthenticationMiddleware(authFunc AuthenticationFunc) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, tenantID, err := authFunc(r)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": map[string]interface{}{
						"code":    "unauthorized",
						"message": "Authentication required",
					},
				})
				return
			}

			// Add user and tenant to context
			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			ctx = context.WithValue(ctx, TenantIDKey, tenantID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CompressionMiddleware adds gzip compression for responses
func CompressionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if client accepts gzip
		if !containsString(r.Header.Values("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		// For now, we'll skip actual compression implementation
		// In production, you'd use something like github.com/klauspost/compress/gzip
		next.ServeHTTP(w, r)
	})
}

// MetricsMiddleware tracks request metrics
type MetricsCollector interface {
	RecordRequest(method, path string, statusCode int, duration time.Duration, size int64)
}

func MetricsMiddleware(collector MetricsCollector) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := newResponseWriter(w)

			next.ServeHTTP(rw, r)

			duration := time.Since(start)
			collector.RecordRequest(r.Method, r.URL.Path, rw.statusCode, duration, rw.bytesWritten)
		})
	}
}

// ValidationMiddleware validates request bodies against a schema
type RequestValidator interface {
	Validate(r *http.Request) error
}

func ValidationMiddleware(validator RequestValidator) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := validator.Validate(r); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": map[string]interface{}{
						"code":    "validation_error",
						"message": err.Error(),
					},
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CacheMiddleware adds cache control headers
func CacheMiddleware(maxAge int) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only cache GET requests
			if r.Method == http.MethodGet {
				w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge))
			}

			next.ServeHTTP(w, r)
		})
	}
}

// TimeoutMiddleware adds a timeout to request processing
func TimeoutMiddleware(timeout time.Duration) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			done := make(chan struct{})
			go func() {
				next.ServeHTTP(w, r.WithContext(ctx))
				close(done)
			}()

			select {
			case <-done:
				// Request completed successfully
			case <-ctx.Done():
				// Timeout occurred
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusGatewayTimeout)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": map[string]interface{}{
						"code":    "timeout",
						"message": "Request timeout",
					},
				})
			}
		})
	}
}

// BodyLoggingMiddleware logs request and response bodies (for debugging)
func BodyLoggingMiddleware(logger *log.Logger) Middleware {
	if logger == nil {
		logger = log.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := ""
			if id, ok := r.Context().Value(RequestIDKey).(string); ok {
				requestID = id
			}

			// Read and log request body
			if r.Body != nil {
				body, err := io.ReadAll(r.Body)
				if err == nil && len(body) > 0 {
					logger.Printf("[%s] Request body: %s", requestID, string(body))
					// Restore body for handler
					r.Body = io.NopCloser(bytes.NewReader(body))
				}
			}

			// Capture response body
			rc := newResponseCapturer(w, true)

			next.ServeHTTP(rc, r)

			// Log response body
			if len(rc.body) > 0 {
				logger.Printf("[%s] Response body: %s", requestID, string(rc.body))
			}
		})
	}
}

// Helper functions

func joinStrings(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}

func containsString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
