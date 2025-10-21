package main

import "time"

// ============================================
// Request/Response Types (snake_case)
// ============================================

// ContentUploadRequest supports four upload methods:
// 1. Multipart file upload (Content-Type: multipart/form-data)
// 2. JSON with base64 data (Content-Type: application/json)
//    - Single: { "mime_type": "...", "filename": "...", "data": "..." }
//    - Multiple: { "contents": [{ "mime_type": "...", "filename": "...", "data": "..." }, ...] }
// 3. JSON with URL reference (Content-Type: application/json)
//    - Single: { "mime_type": "...", "filename": "...", "url": "..." }
//    - Multiple: { "contents": [{ "mime_type": "...", "filename": "...", "url": "..." }, ...] }
// 4. JSON with metadata only - returns upload URL for large files (Content-Type: application/json)
//    - Single: { "mime_type": "...", "filename": "...", "size": 123456 }
//    - Multiple: { "contents": [{ "mime_type": "...", "filename": "...", "size": 123456 }, ...] }
type ContentUploadRequest struct {
	// For single JSON upload with base64, URL, or metadata-only
	MimeType     *string                `json:"mime_type,omitempty"`
	Filename     *string                `json:"filename,omitempty"`
	Size         *int64                 `json:"size,omitempty"` // File size in bytes (for upload URL request)
	Data         *string                `json:"data,omitempty"` // Base64-encoded file data
	URL          *string                `json:"url,omitempty"`  // URL to publicly accessible content
	
	// For multiple JSON uploads
	Contents     []ContentUploadItem    `json:"contents,omitempty"`
	
	// Optional metadata
	AnalysisType *string                `json:"analysis_type,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

type ContentUploadItem struct {
	MimeType string  `json:"mime_type"`
	Filename string  `json:"filename"`
	Size     *int64  `json:"size,omitempty"` // File size in bytes (for upload URL request)
	Data     *string `json:"data,omitempty"` // Base64-encoded file data
	URL      *string `json:"url,omitempty"`  // URL to publicly accessible content
}

type ContentUploadResponse struct {
	ID                 string `json:"id"`
	URL                string `json:"url"`
	UploadURL          string `json:"upload_url,omitempty"`
	StorageBackendName string `json:"storage_backend_name,omitempty"`
}

type InputContent struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	ID       string `json:"id,omitempty"`
	URL      string `json:"url,omitempty"`
	Data     string `json:"data,omitempty"`
	Filename string `json:"filename,omitempty"`
}

type ContentAnalysisRequest struct {
	Content      []InputContent         `json:"content"`
	AnalysisType string                 `json:"analysis_type,omitempty"`
	Options      map[string]interface{} `json:"options,omitempty"`
}

type ContentAnalysisResponse struct {
	ID                string             `json:"id"`
	Status            string             `json:"status"`
	Result            interface{}        `json:"result,omitempty"`
	GeneratedContents []GeneratedContent `json:"generated_contents,omitempty"`
	Error             string             `json:"error,omitempty"`
	CreatedAt         time.Time          `json:"created_at"`
	CompletedAt       *time.Time         `json:"completed_at,omitempty"`
}

type GeneratedContent struct {
	ID          string     `json:"id"`
	Filename    string     `json:"filename"`
	MimeType    string     `json:"mime_type"`
	Size        int64      `json:"size"`
	DownloadURL string     `json:"download_url"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

type AnalysisStatusResponse struct {
	ID                string             `json:"id"`
	Status            string             `json:"status"`
	Progress          *int               `json:"progress,omitempty"`
	Message           string             `json:"message,omitempty"`
	Result            interface{}        `json:"result,omitempty"`
	GeneratedContents []GeneratedContent `json:"generated_contents,omitempty"`
	Error             string             `json:"error,omitempty"`
}

type ContentDownloadMetadata struct {
	ID        string     `json:"id"`
	Filename  string     `json:"filename"`
	MimeType  string     `json:"mime_type"`
	Size      int64      `json:"size"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type ContentListResponse struct {
	Contents []ContentUploadResponse `json:"contents"`
	Total    int                     `json:"total"`
}

type BatchDownloadRequest struct {
	ContentIDs []string `json:"content_ids"`
}

// ============================================
// Service Client Parameter Structs
// ============================================

type UploadFileParams struct {
	FilePath     string
	AnalysisType string
	Metadata     map[string]interface{}
}

type DownloadContentParams struct {
	ContentIDs []string
	OutputPath string
}

type ListContentsParams struct {
	Limit  int
	Offset int
}
