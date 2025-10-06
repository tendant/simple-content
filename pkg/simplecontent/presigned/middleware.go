package presigned

import (
	"context"
	"log"
	"net/http"
)

type contextKey string

const (
	// ObjectKeyContextKey is the context key for storing the validated object key
	ObjectKeyContextKey contextKey = "presigned:object_key"
)

// ValidateMiddleware returns HTTP middleware that validates presigned URL signatures
// If validation fails, it returns an appropriate HTTP error response
// If validation succeeds, it calls the next handler with the validated object key in the context
//
// Example:
//   http.Handle("/upload/", presigned.ValidateMiddleware(secretKey, uploadHandler))
func ValidateMiddleware(secretKey string, next http.Handler) http.Handler {
	signer := New(WithSecretKey(secretKey))
	return ValidateMiddlewareWithSigner(signer, next)
}

// ValidateMiddlewareWithSigner returns HTTP middleware using a pre-configured Signer
// This allows more control over the signer configuration (expiration, URL pattern, etc.)
//
// Example:
//   signer := presigned.New(
//       presigned.WithSecretKey(secretKey),
//       presigned.WithDefaultExpiration(30*time.Minute),
//   )
//   http.Handle("/upload/", presigned.ValidateMiddlewareWithSigner(signer, uploadHandler))
func ValidateMiddlewareWithSigner(signer *Signer, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If signer is not enabled, pass through without validation
		if !signer.IsEnabled() {
			next.ServeHTTP(w, r)
			return
		}

		// Validate the request signature
		if err := signer.ValidateRequest(r); err != nil {
			handleValidationError(w, err)
			return
		}

		// Extract object key from URL path
		objectKey, err := signer.ExtractObjectKey(r.URL.Path)
		if err != nil {
			log.Printf("presigned: failed to extract object key: %v", err)
			http.Error(w, "Invalid upload URL", http.StatusBadRequest)
			return
		}

		// Add object key to context
		ctx := context.WithValue(r.Context(), ObjectKeyContextKey, objectKey)
		r = r.WithContext(ctx)

		// Call next handler
		next.ServeHTTP(w, r)
	})
}

// ObjectKeyFromContext extracts the validated object key from the request context
// Returns empty string if not found
//
// Example:
//   func uploadHandler(w http.ResponseWriter, r *http.Request) {
//       objectKey := presigned.ObjectKeyFromContext(r.Context())
//       // Use objectKey for storage
//   }
func ObjectKeyFromContext(ctx context.Context) string {
	if key, ok := ctx.Value(ObjectKeyContextKey).(string); ok {
		return key
	}
	return ""
}

// handleValidationError writes an appropriate HTTP error response based on the validation error
func handleValidationError(w http.ResponseWriter, err error) {
	switch {
	case err == ErrMissingSignature:
		http.Error(w, "Missing signature parameter", http.StatusUnauthorized)
	case err == ErrMissingExpiration:
		http.Error(w, "Missing expires parameter", http.StatusUnauthorized)
	case err == ErrInvalidExpiration:
		http.Error(w, "Invalid expires parameter", http.StatusBadRequest)
	case err == ErrExpired:
		http.Error(w, "Presigned URL has expired", http.StatusForbidden)
	case err == ErrInvalidSignature:
		http.Error(w, "Invalid signature", http.StatusForbidden)
	default:
		log.Printf("presigned: validation error: %v", err)
		http.Error(w, "Authentication failed", http.StatusForbidden)
	}
}

// ValidateHandler wraps an http.HandlerFunc with signature validation
// This is a convenience function for simple use cases
//
// Example:
//   http.HandleFunc("/upload/", presigned.ValidateHandler(secretKey, uploadHandler))
func ValidateHandler(secretKey string, handler http.HandlerFunc) http.HandlerFunc {
	middleware := ValidateMiddleware(secretKey, handler)
	return func(w http.ResponseWriter, r *http.Request) {
		middleware.ServeHTTP(w, r)
	}
}
