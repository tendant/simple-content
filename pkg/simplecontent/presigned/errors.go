package presigned

import "errors"

// Signature validation errors
var (
	// ErrNoSecretKey is returned when attempting to sign URLs without a configured secret key
	ErrNoSecretKey = errors.New("presigned: no secret key configured")

	// ErrMissingSignature is returned when the signature query parameter is missing
	ErrMissingSignature = errors.New("presigned: missing signature parameter")

	// ErrMissingExpiration is returned when the expires query parameter is missing
	ErrMissingExpiration = errors.New("presigned: missing expires parameter")

	// ErrInvalidExpiration is returned when the expires parameter cannot be parsed
	ErrInvalidExpiration = errors.New("presigned: invalid expires parameter")

	// ErrExpired is returned when the presigned URL has expired
	ErrExpired = errors.New("presigned: URL has expired")

	// ErrInvalidSignature is returned when the signature is invalid
	ErrInvalidSignature = errors.New("presigned: invalid signature")
)

// IsAuthError returns true if the error is a signature validation error
func IsAuthError(err error) bool {
	return errors.Is(err, ErrMissingSignature) ||
		errors.Is(err, ErrMissingExpiration) ||
		errors.Is(err, ErrInvalidExpiration) ||
		errors.Is(err, ErrExpired) ||
		errors.Is(err, ErrInvalidSignature)
}
