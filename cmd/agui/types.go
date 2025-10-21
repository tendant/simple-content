package main

// ============================================
// AG-UI Protocol Types
// ============================================

// InputContentType represents the type of content in a multimodal message
type InputContentType string

const (
	InputContentTypeText   InputContentType = "text"
	InputContentTypeBinary InputContentType = "binary"
)

// InputContent represents a single item in a multimodal content array
type InputContent struct {
	Type     InputContentType `json:"type"`
	Text     *string          `json:"text,omitempty"`      // For text content
	MimeType *string          `json:"mime_type,omitempty"` // For binary content
	Filename *string          `json:"filename,omitempty"`  // For binary content
	ID       *string          `json:"id,omitempty"`        // Pre-uploaded content ID
	URL      *string          `json:"url,omitempty"`       // External URL
	Data     *string          `json:"data,omitempty"`      // Base64-encoded data
}

// ContentAnalysisRequest represents a request to analyze multimodal content
type ContentAnalysisRequest struct {
	Content      []InputContent         `json:"content"`
	AnalysisType *string                `json:"analysis_type,omitempty"`
	Options      map[string]interface{} `json:"options,omitempty"`
}

// AnalysisStatus represents the status of an analysis job
type AnalysisStatus string

const (
	AnalysisStatusPending    AnalysisStatus = "pending"
	AnalysisStatusProcessing AnalysisStatus = "processing"
	AnalysisStatusCompleted  AnalysisStatus = "completed"
	AnalysisStatusFailed     AnalysisStatus = "failed"
)
