package presigned

import "time"

// Option is a functional option for configuring a Signer
type Option func(*Signer)

// WithSecretKey sets the secret key used for HMAC signing
// The key should be at least 32 bytes for security
func WithSecretKey(key string) Option {
	return func(s *Signer) {
		s.secretKey = []byte(key)
	}
}

// WithDefaultExpiration sets the default expiration duration for signed URLs
// Default is 1 hour if not specified
func WithDefaultExpiration(duration time.Duration) Option {
	return func(s *Signer) {
		s.defaultExpiration = duration
	}
}

// WithURLPattern sets the URL pattern used for object key extraction
// The pattern must contain {key} placeholder
// Examples: "/upload/{key}", "/api/v1/upload/{key}", "/storage/{key}"
func WithURLPattern(pattern string) Option {
	return func(s *Signer) {
		s.urlPattern = pattern
	}
}

// WithCustomPayloadFunc allows customizing the signature payload format
// The function receives (method, path, expiresAt) and should return the payload string
// Default format is: METHOD|PATH|EXPIRES
func WithCustomPayloadFunc(fn func(method, path string, expiresAt int64) string) Option {
	return func(s *Signer) {
		s.customPayloadFunc = fn
	}
}
