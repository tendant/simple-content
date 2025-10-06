// Package presigned provides HMAC-based authentication for presigned upload URLs.
//
// This package enables secure, time-limited upload URLs similar to AWS S3 presigned URLs,
// suitable for any storage backend (filesystem, S3-compatible, custom implementations).
//
// # Key Features
//
//   - HMAC-SHA256 signature-based authentication
//   - Time-limited URL expiration
//   - Storage backend agnostic
//   - HTTP middleware for easy integration
//   - Client SDK for upload workflows
//   - Customizable URL patterns
//
// # Basic Usage
//
// Server-side: Generate presigned URL
//
//	signer := presigned.New(presigned.WithSecretKey("your-secret-key"))
//	url, err := signer.SignURL("PUT", "/upload/myfile.pdf", 1*time.Hour)
//
// Server-side: Validate upload request
//
//	err := signer.ValidateRequest(r)
//	if err != nil {
//	    // Invalid signature or expired URL
//	}
//
// Client-side: Upload to presigned URL
//
//	client := presigned.NewClient()
//	err := client.Upload(ctx, presignedURL, fileReader)
//
// # HTTP Middleware
//
// Add validation middleware to your HTTP router:
//
//	mux := http.NewServeMux()
//	uploadHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	    // Handle upload - signature already validated
//	    objectKey := presigned.ObjectKeyFromContext(r.Context())
//	})
//	mux.Handle("/upload/", presigned.ValidateMiddleware(secretKey, uploadHandler))
//
// # Configuration Options
//
//	signer := presigned.New(
//	    presigned.WithSecretKey("your-secret-key"),
//	    presigned.WithDefaultExpiration(30*time.Minute),
//	    presigned.WithURLPattern("/api/v1/upload/{key}"),
//	)
//
// # Security Best Practices
//
//   - Use strong secret keys (minimum 32 bytes, use crypto/rand)
//   - Set appropriate expiration times (15min - 1hr recommended)
//   - Always use HTTPS in production
//   - Rotate keys periodically
//   - Monitor for invalid signature attempts
//   - Consider rate limiting upload endpoints
//
// # Example: Complete Upload Workflow
//
//	// Server: Generate presigned URL
//	func getUploadURL(w http.ResponseWriter, r *http.Request) {
//	    objectKey := "uploads/user123/document.pdf"
//	    url, _ := signer.SignURL("PUT", "/upload/"+objectKey, 1*time.Hour)
//	    json.NewEncoder(w).Encode(map[string]string{"upload_url": url})
//	}
//
//	// Server: Handle upload with validation
//	func handleUpload(w http.ResponseWriter, r *http.Request) {
//	    objectKey := presigned.ObjectKeyFromContext(r.Context())
//	    // Store uploaded file using objectKey
//	    io.Copy(storage, r.Body)
//	}
//
//	// Client: Upload file
//	func uploadFile(url string, file io.Reader) error {
//	    client := presigned.NewClient()
//	    return client.Upload(context.Background(), url, file)
//	}
//
// For complete examples, see the examples/ directory.
package presigned
