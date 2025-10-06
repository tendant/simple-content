package presigned

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Signer generates and validates HMAC-signed presigned URLs
type Signer struct {
	secretKey          []byte
	defaultExpiration  time.Duration
	urlPattern         string // e.g., "/upload/{key}" or "/api/v1/upload/{key}"
	customPayloadFunc  func(method, path string, expiresAt int64) string
}

// New creates a new Signer with the given options
func New(opts ...Option) *Signer {
	s := &Signer{
		defaultExpiration: 1 * time.Hour,
		urlPattern:        "/upload/{key}",
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// SignURL generates a presigned URL for the given HTTP method and path
// Returns the complete URL with signature and expiration query parameters
//
// Example:
//   url, err := signer.SignURL("PUT", "/upload/myfile.pdf", 1*time.Hour)
//   // Returns: /upload/myfile.pdf?signature=abc123...&expires=1696789012
func (s *Signer) SignURL(method, path string, expiresIn time.Duration) (string, error) {
	if len(s.secretKey) == 0 {
		return "", ErrNoSecretKey
	}

	if expiresIn == 0 {
		expiresIn = s.defaultExpiration
	}

	// Calculate expiration timestamp
	expiresAt := time.Now().Add(expiresIn).Unix()

	// Create signature payload
	payload := s.createPayload(method, path, expiresAt)

	// Generate HMAC-SHA256 signature
	signature := s.generateSignature(payload)

	// Build signed URL
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	signedURL := fmt.Sprintf("%s%ssignature=%s&expires=%d",
		path, separator, signature, expiresAt)

	return signedURL, nil
}

// SignURLWithBase generates a presigned URL with a base URL prefix
//
// Example:
//   url, err := signer.SignURLWithBase("https://api.example.com", "PUT", "/upload/myfile.pdf", 1*time.Hour)
//   // Returns: https://api.example.com/upload/myfile.pdf?signature=abc123...&expires=1696789012
func (s *Signer) SignURLWithBase(baseURL, method, path string, expiresIn time.Duration) (string, error) {
	signedPath, err := s.SignURL(method, path, expiresIn)
	if err != nil {
		return "", err
	}
	return baseURL + signedPath, nil
}

// ValidateRequest validates the signature and expiration of an HTTP request
// Returns an error if the signature is invalid or the URL has expired
func (s *Signer) ValidateRequest(r *http.Request) error {
	if len(s.secretKey) == 0 {
		// No secret key configured - allow all requests (backward compatibility)
		return nil
	}

	// Extract signature and expiration from query parameters
	query := r.URL.Query()
	signature := query.Get("signature")
	expiresStr := query.Get("expires")

	if signature == "" {
		return ErrMissingSignature
	}
	if expiresStr == "" {
		return ErrMissingExpiration
	}

	expiresAt, err := strconv.ParseInt(expiresStr, 10, 64)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidExpiration, err)
	}

	// Extract path without query parameters
	path := r.URL.Path
	if r.URL.RawQuery != "" {
		// Preserve original query params (except signature and expires)
		cleanQuery := url.Values{}
		for k, v := range query {
			if k != "signature" && k != "expires" {
				cleanQuery[k] = v
			}
		}
		if len(cleanQuery) > 0 {
			path = path + "?" + cleanQuery.Encode()
		}
	}

	// Validate signature
	return s.Validate(r.Method, path, signature, expiresAt)
}

// Validate validates the signature and expiration for a given method, path, signature, and expiration timestamp
func (s *Signer) Validate(method, path, signature string, expiresAt int64) error {
	// Check expiration
	if time.Now().Unix() > expiresAt {
		return ErrExpired
	}

	// Recreate the payload that was signed
	payload := s.createPayload(method, path, expiresAt)

	// Generate expected signature
	expectedSignature := s.generateSignature(payload)

	// Compare signatures using constant-time comparison to prevent timing attacks
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return ErrInvalidSignature
	}

	return nil
}

// ExtractObjectKey extracts the object key from a URL path based on the configured URL pattern
//
// Example:
//   signer := New(WithURLPattern("/upload/{key}"))
//   key, err := signer.ExtractObjectKey("/upload/myfile.pdf")
//   // Returns: "myfile.pdf"
func (s *Signer) ExtractObjectKey(path string) (string, error) {
	// Parse the URL pattern to find where {key} is located
	pattern := s.urlPattern
	placeholder := "{key}"

	idx := strings.Index(pattern, placeholder)
	if idx == -1 {
		return "", fmt.Errorf("URL pattern does not contain {key} placeholder")
	}

	prefix := pattern[:idx]
	suffix := pattern[idx+len(placeholder):]

	// Remove prefix and suffix from path
	if !strings.HasPrefix(path, prefix) {
		return "", fmt.Errorf("path does not match URL pattern prefix")
	}

	key := strings.TrimPrefix(path, prefix)

	if suffix != "" && strings.HasSuffix(key, suffix) {
		key = strings.TrimSuffix(key, suffix)
	}

	return key, nil
}

// IsEnabled returns true if signature validation is enabled (secret key is set)
func (s *Signer) IsEnabled() bool {
	return len(s.secretKey) > 0
}

// createPayload creates the signature payload
// Default format: METHOD|PATH|EXPIRES
// Can be customized using WithCustomPayloadFunc
func (s *Signer) createPayload(method, path string, expiresAt int64) string {
	if s.customPayloadFunc != nil {
		return s.customPayloadFunc(method, path, expiresAt)
	}
	return fmt.Sprintf("%s|%s|%d", method, path, expiresAt)
}

// generateSignature generates HMAC-SHA256 signature for the given payload
func (s *Signer) generateSignature(payload string) string {
	h := hmac.New(sha256.New, s.secretKey)
	h.Write([]byte(payload))
	return hex.EncodeToString(h.Sum(nil))
}
