package api

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestIDMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request ID is in context
		requestID, ok := r.Context().Value(RequestIDKey).(string)
		assert.True(t, ok)
		assert.NotEmpty(t, requestID)
		w.WriteHeader(http.StatusOK)
	})

	wrapped := RequestIDMiddleware(handler)

	t.Run("generates request ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.NotEmpty(t, rr.Header().Get("X-Request-ID"))
	})

	t.Run("uses provided request ID", func(t *testing.T) {
		customID := "my-custom-id"
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Request-ID", customID)
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, customID, rr.Header().Get("X-Request-ID"))
	})
}

func TestLoggingMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello"))
	})

	logger := log.Default()
	wrapped := LoggingMiddleware(logger)(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "Hello", rr.Body.String())
}

func TestRecoveryMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something went wrong")
	})

	wrapped := RecoveryMiddleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// Should not panic
	assert.NotPanics(t, func() {
		wrapped.ServeHTTP(rr, req)
	})

	assert.Equal(t, http.StatusInternalServerError, rr.Code)

	var response map[string]interface{}
	err := json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)

	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "internal_error", errorObj["code"])
}

func TestCORSMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := CORSMiddleware(
		[]string{"https://example.com"},
		[]string{"GET", "POST"},
		[]string{"Content-Type"},
	)(handler)

	t.Run("adds CORS headers for allowed origin", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "https://example.com", rr.Header().Get("Access-Control-Allow-Origin"))
		assert.Contains(t, rr.Header().Get("Access-Control-Allow-Methods"), "GET")
	})

	t.Run("handles preflight requests", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNoContent, rr.Code)
	})

	t.Run("wildcard origin", func(t *testing.T) {
		wildcardWrapped := CORSMiddleware(
			[]string{"*"},
			nil,
			nil,
		)(handler)

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "https://anywhere.com")
		rr := httptest.NewRecorder()

		wildcardWrapped.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.NotEmpty(t, rr.Header().Get("Access-Control-Allow-Origin"))
	})
}

func TestRateLimiterMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	limiter := NewRateLimiter(5) // 5 requests per minute
	wrapped := limiter.Middleware(handler)

	t.Run("allows requests within limit", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "127.0.0.1:1234"
			rr := httptest.NewRecorder()

			wrapped.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)
		}
	})

	t.Run("blocks requests over limit", func(t *testing.T) {
		// First exhaust the limit
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "127.0.0.1:5678"
			rr := httptest.NewRecorder()
			wrapped.ServeHTTP(rr, req)
		}

		// Next request should be blocked
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "127.0.0.1:5678"
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusTooManyRequests, rr.Code)
		assert.Equal(t, "60", rr.Header().Get("Retry-After"))

		var response map[string]interface{}
		err := json.NewDecoder(rr.Body).Decode(&response)
		require.NoError(t, err)

		errorObj := response["error"].(map[string]interface{})
		assert.Equal(t, "rate_limit_exceeded", errorObj["code"])
	})
}

func TestRequestSizeLimitMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to read body
		_, err := http.MaxBytesReader(w, r.Body, 100).Read(make([]byte, 200))
		if err != nil {
			http.Error(w, "Request too large", http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	wrapped := RequestSizeLimitMiddleware(100)(handler)

	t.Run("allows small requests", func(t *testing.T) {
		body := strings.NewReader("small body")
		req := httptest.NewRequest("POST", "/test", body)
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})
}

func TestAuthenticationMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify user/tenant in context
		userID, ok := r.Context().Value(UserIDKey).(uuid.UUID)
		assert.True(t, ok)
		assert.NotEqual(t, uuid.Nil, userID)

		tenantID, ok := r.Context().Value(TenantIDKey).(uuid.UUID)
		assert.True(t, ok)
		assert.NotEqual(t, uuid.Nil, tenantID)

		w.WriteHeader(http.StatusOK)
	})

	authFunc := func(r *http.Request) (userID, tenantID uuid.UUID, err error) {
		token := r.Header.Get("Authorization")
		if token == "" {
			return uuid.Nil, uuid.Nil, assert.AnError
		}
		return uuid.New(), uuid.New(), nil
	}

	wrapped := AuthenticationMiddleware(authFunc)(handler)

	t.Run("passes with valid token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("fails without token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)

		var response map[string]interface{}
		err := json.NewDecoder(rr.Body).Decode(&response)
		require.NoError(t, err)

		errorObj := response["error"].(map[string]interface{})
		assert.Equal(t, "unauthorized", errorObj["code"])
	})
}

func TestCacheMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := CacheMiddleware(3600)(handler)

	t.Run("adds cache header for GET", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Header().Get("Cache-Control"), "max-age=3600")
	})

	t.Run("no cache header for POST", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", nil)
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Empty(t, rr.Header().Get("Cache-Control"))
	})
}

func TestTimeoutMiddleware(t *testing.T) {
	t.Run("completes within timeout", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(10 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		})

		wrapped := TimeoutMiddleware(100 * time.Millisecond)(handler)

		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("times out on slow handler", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(200 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		})

		wrapped := TimeoutMiddleware(50 * time.Millisecond)(handler)

		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusGatewayTimeout, rr.Code)
	})
}

func TestMiddlewareChain(t *testing.T) {
	var executionOrder []string

	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executionOrder = append(executionOrder, "middleware1-before")
			next.ServeHTTP(w, r)
			executionOrder = append(executionOrder, "middleware1-after")
		})
	}

	middleware2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executionOrder = append(executionOrder, "middleware2-before")
			next.ServeHTTP(w, r)
			executionOrder = append(executionOrder, "middleware2-after")
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		executionOrder = append(executionOrder, "handler")
		w.WriteHeader(http.StatusOK)
	})

	chain := NewMiddlewareChain(middleware1, middleware2)
	wrapped := chain.Wrap(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	expected := []string{
		"middleware1-before",
		"middleware2-before",
		"handler",
		"middleware2-after",
		"middleware1-after",
	}
	assert.Equal(t, expected, executionOrder)
}

func TestMiddlewareChain_Then(t *testing.T) {
	chain := NewMiddlewareChain()

	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Middleware-1", "true")
			next.ServeHTTP(w, r)
		})
	}

	middleware2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Middleware-2", "true")
			next.ServeHTTP(w, r)
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	chain.Then(middleware1).Then(middleware2)
	wrapped := chain.Wrap(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "true", rr.Header().Get("X-Middleware-1"))
	assert.Equal(t, "true", rr.Header().Get("X-Middleware-2"))
}

func TestMetricsMiddleware(t *testing.T) {
	type recordedMetric struct {
		Method     string
		Path       string
		StatusCode int
		Duration   time.Duration
		Size       int64
	}

	var recorded recordedMetric

	collector := &mockMetricsCollector{
		recordFunc: func(method, path string, statusCode int, duration time.Duration, size int64) {
			recorded = recordedMetric{
				Method:     method,
				Path:       path,
				StatusCode: statusCode,
				Duration:   duration,
				Size:       size,
			}
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))
	})

	wrapped := MetricsMiddleware(collector)(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	assert.Equal(t, "GET", recorded.Method)
	assert.Equal(t, "/test", recorded.Path)
	assert.Equal(t, http.StatusOK, recorded.StatusCode)
	assert.Greater(t, recorded.Duration, time.Duration(0))
	assert.Equal(t, int64(13), recorded.Size) // "Hello, World!"
}

type mockMetricsCollector struct {
	recordFunc func(method, path string, statusCode int, duration time.Duration, size int64)
}

func (m *mockMetricsCollector) RecordRequest(method, path string, statusCode int, duration time.Duration, size int64) {
	if m.recordFunc != nil {
		m.recordFunc(method, path, statusCode, duration, size)
	}
}
