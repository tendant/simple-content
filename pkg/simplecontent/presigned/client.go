package presigned

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client provides methods for uploading files to presigned URLs
type Client struct {
	httpClient      *http.Client
	retryAttempts   int
	retryDelay      time.Duration
	progressFunc    ProgressFunc
}

// ProgressFunc is called during upload to report progress
// It receives the number of bytes uploaded so far
type ProgressFunc func(bytesUploaded int64)

// ClientOption is a functional option for configuring a Client
type ClientOption func(*Client)

// NewClient creates a new presigned upload client
func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Minute, // Long timeout for large uploads
		},
		retryAttempts: 3,
		retryDelay:    1 * time.Second,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = client
	}
}

// WithRetry configures retry behavior
func WithRetry(attempts int, delay time.Duration) ClientOption {
	return func(c *Client) {
		c.retryAttempts = attempts
		c.retryDelay = delay
	}
}

// WithProgress sets a progress callback function
func WithProgress(fn ProgressFunc) ClientOption {
	return func(c *Client) {
		c.progressFunc = fn
	}
}

// Upload uploads data to a presigned URL
//
// Example:
//   client := presigned.NewClient()
//   err := client.Upload(ctx, presignedURL, fileReader)
func (c *Client) Upload(ctx context.Context, presignedURL string, data io.Reader, opts ...UploadOption) error {
	uploadOpts := &uploadOptions{
		contentType: "application/octet-stream",
	}
	for _, opt := range opts {
		opt(uploadOpts)
	}

	// Wrap reader with progress tracking if enabled
	reader := data
	if c.progressFunc != nil {
		reader = &progressReader{
			reader:   data,
			callback: c.progressFunc,
		}
	}

	var lastErr error
	for attempt := 0; attempt < c.retryAttempts; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.retryDelay * time.Duration(attempt)):
			}
		}

		// Create upload request
		req, err := http.NewRequestWithContext(ctx, "PUT", presignedURL, reader)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		// Set headers
		req.Header.Set("Content-Type", uploadOpts.contentType)
		for k, v := range uploadOpts.headers {
			req.Header.Set(k, v)
		}

		// Perform upload
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("upload failed: %w", err)
			continue
		}

		// Check response
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil // Success
		}

		lastErr = fmt.Errorf("upload failed with status: %s", resp.Status)

		// Don't retry on client errors (4xx)
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return lastErr
		}
	}

	return fmt.Errorf("upload failed after %d attempts: %w", c.retryAttempts, lastErr)
}

// UploadWithContentType is a convenience method for uploading with a specific content type
//
// Example:
//   err := client.UploadWithContentType(ctx, url, file, "image/png")
func (c *Client) UploadWithContentType(ctx context.Context, presignedURL string, data io.Reader, contentType string) error {
	return c.Upload(ctx, presignedURL, data, WithContentType(contentType))
}

// uploadOptions contains upload configuration
type uploadOptions struct {
	contentType string
	headers     map[string]string
}

// UploadOption is a functional option for Upload method
type UploadOption func(*uploadOptions)

// WithContentType sets the Content-Type header for the upload
func WithContentType(contentType string) UploadOption {
	return func(o *uploadOptions) {
		o.contentType = contentType
	}
}

// WithHeader adds a custom header to the upload request
func WithHeader(key, value string) UploadOption {
	return func(o *uploadOptions) {
		if o.headers == nil {
			o.headers = make(map[string]string)
		}
		o.headers[key] = value
	}
}

// progressReader wraps an io.Reader to track upload progress
type progressReader struct {
	reader       io.Reader
	bytesRead    int64
	callback     ProgressFunc
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.bytesRead += int64(n)
	if pr.callback != nil && n > 0 {
		pr.callback(pr.bytesRead)
	}
	return n, err
}
