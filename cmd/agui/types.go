package main

import "time"

// ============================================
// Request/Response Types (snake_case)
// ============================================

type ContentUploadRequest struct {
	MimeType     string                 `json:"mime_type,omitempty"`
	Filename     string                 `json:"filename,omitempty"`
	Size         *int64                 `json:"size,omitempty"`
	Data         string                 `json:"data,omitempty"`
	URL          string                 `json:"url,omitempty"`
	Contents     []ContentUploadItem    `json:"contents,omitempty"`
	AnalysisType string                 `json:"analysis_type,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

type ContentUploadItem struct {
	MimeType string `json:"mime_type"`
	Filename string `json:"filename"`
	Size     *int64 `json:"size,omitempty"`
	Data     string `json:"data,omitempty"`
	URL      string `json:"url,omitempty"`
}

type ContentUploadResponse struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	UploadURL string `json:"upload_url,omitempty"`
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
