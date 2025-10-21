package main

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/tendant/simple-content/pkg/simplecontent"
)

// mockService is a mock implementation of simplecontent.Service for testing
type mockService struct {
	uploadContentFunc      func(ctx context.Context, req simplecontent.UploadContentRequest) (*simplecontent.Content, error)
	getContentDetailsFunc  func(ctx context.Context, contentID uuid.UUID, options ...simplecontent.ContentDetailsOption) (*simplecontent.ContentDetails, error)
}

func (m *mockService) UploadContent(ctx context.Context, req simplecontent.UploadContentRequest) (*simplecontent.Content, error) {
	if m.uploadContentFunc != nil {
		return m.uploadContentFunc(ctx, req)
	}
	return nil, errors.New("not implemented")
}

func (m *mockService) GetContentDetails(ctx context.Context, contentID uuid.UUID, options ...simplecontent.ContentDetailsOption) (*simplecontent.ContentDetails, error) {
	if m.getContentDetailsFunc != nil {
		return m.getContentDetailsFunc(ctx, contentID, options...)
	}
	return nil, errors.New("not implemented")
}

// Stub implementations for other Service interface methods
func (m *mockService) CreateContent(ctx context.Context, req simplecontent.CreateContentRequest) (*simplecontent.Content, error) {
	return nil, errors.New("not implemented")
}

func (m *mockService) GetContent(ctx context.Context, id uuid.UUID) (*simplecontent.Content, error) {
	return nil, errors.New("not implemented")
}

func (m *mockService) UpdateContent(ctx context.Context, req simplecontent.UpdateContentRequest) error {
	return errors.New("not implemented")
}

func (m *mockService) DeleteContent(ctx context.Context, id uuid.UUID) error {
	return errors.New("not implemented")
}

func (m *mockService) ListContent(ctx context.Context, req simplecontent.ListContentRequest) ([]*simplecontent.Content, error) {
	return nil, errors.New("not implemented")
}

func (m *mockService) UploadDerivedContent(ctx context.Context, req simplecontent.UploadDerivedContentRequest) (*simplecontent.Content, error) {
	return nil, errors.New("not implemented")
}

func (m *mockService) UploadObjectForContent(ctx context.Context, req simplecontent.UploadObjectForContentRequest) (*simplecontent.Object, error) {
	return nil, errors.New("not implemented")
}

func (m *mockService) DownloadContent(ctx context.Context, contentID uuid.UUID) (io.ReadCloser, error) {
	return nil, errors.New("not implemented")
}

func (m *mockService) SetContentMetadata(ctx context.Context, req simplecontent.SetContentMetadataRequest) error {
	return errors.New("not implemented")
}

func (m *mockService) GetContentMetadata(ctx context.Context, contentID uuid.UUID) (*simplecontent.ContentMetadata, error) {
	return nil, errors.New("not implemented")
}

func (m *mockService) UpdateContentStatus(ctx context.Context, id uuid.UUID, newStatus simplecontent.ContentStatus) error {
	return errors.New("not implemented")
}

func (m *mockService) UpdateObjectStatus(ctx context.Context, id uuid.UUID, newStatus simplecontent.ObjectStatus) error {
	return errors.New("not implemented")
}

func (m *mockService) GetContentByStatus(ctx context.Context, status simplecontent.ContentStatus) ([]*simplecontent.Content, error) {
	return nil, errors.New("not implemented")
}

func (m *mockService) GetObjectsByStatus(ctx context.Context, status simplecontent.ObjectStatus) ([]*simplecontent.Object, error) {
	return nil, errors.New("not implemented")
}

func (m *mockService) GetObjectsByContentID(ctx context.Context, contentID uuid.UUID) ([]*simplecontent.Object, error) {
	return nil, errors.New("not implemented")
}

func (m *mockService) RegisterBackend(name string, backend simplecontent.BlobStore) {
}

func (m *mockService) GetBackend(name string) (simplecontent.BlobStore, error) {
	return nil, errors.New("not implemented")
}

func (m *mockService) CreateDerivedContent(ctx context.Context, req simplecontent.CreateDerivedContentRequest) (*simplecontent.Content, error) {
	return nil, errors.New("not implemented")
}

func (m *mockService) CreateDerivedContentRelationship(ctx context.Context, req simplecontent.CreateDerivedContentRequest) (*simplecontent.DerivedContent, error) {
	return nil, errors.New("not implemented")
}

func (m *mockService) GetDerivedRelationship(ctx context.Context, contentID uuid.UUID) (*simplecontent.DerivedContent, error) {
	return nil, errors.New("not implemented")
}

func (m *mockService) ListDerivedContent(ctx context.Context, options ...simplecontent.ListDerivedContentOption) ([]*simplecontent.DerivedContent, error) {
	return nil, errors.New("not implemented")
}

func (m *mockService) GetContentDetailsBatch(ctx context.Context, contentIDs []uuid.UUID, options ...simplecontent.ContentDetailsOption) ([]*simplecontent.ContentDetails, error) {
	return nil, errors.New("not implemented")
}

// Test helper to create a temporary test file
func createTestFile(t *testing.T, content string) string {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "test-upload-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}

	return tmpFile.Name()
}

func TestUploadFile_Success(t *testing.T) {
	// Create a temporary test file
	testContent := "test file content"
	testFile := createTestFile(t, testContent)
	defer os.Remove(testFile)

	// Expected values
	expectedID := uuid.New()
	expectedURL := "https://example.com/download/test-file"

	// Create mock service
	mock := &mockService{
		uploadContentFunc: func(ctx context.Context, req simplecontent.UploadContentRequest) (*simplecontent.Content, error) {
			// Verify request parameters
			if req.Name != filepath.Base(testFile) {
				t.Errorf("expected name %s, got %s", filepath.Base(testFile), req.Name)
			}
			if req.FileName != filepath.Base(testFile) {
				t.Errorf("expected filename %s, got %s", filepath.Base(testFile), req.FileName)
			}
			if req.FileSize != int64(len(testContent)) {
				t.Errorf("expected file size %d, got %d", len(testContent), req.FileSize)
			}
			if req.Reader == nil {
				t.Error("expected reader to be set")
			}

			return &simplecontent.Content{
				ID: expectedID,
			}, nil
		},
		getContentDetailsFunc: func(ctx context.Context, contentID uuid.UUID, options ...simplecontent.ContentDetailsOption) (*simplecontent.ContentDetails, error) {
			if contentID != expectedID {
				t.Errorf("expected content ID %s, got %s", expectedID, contentID)
			}
			return &simplecontent.ContentDetails{
				Download: expectedURL,
			}, nil
		},
	}

	// Create service client
	client := NewServiceClient(mock, false)

	// Test upload
	metadata := map[string]interface{}{"key": "value"}
	resp, err := client.UploadFile(testFile, "test-analysis", metadata)

	// Verify results
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.ID != expectedID.String() {
		t.Errorf("expected ID %s, got %s", expectedID.String(), resp.ID)
	}
	if resp.URL != expectedURL {
		t.Errorf("expected URL %s, got %s", expectedURL, resp.URL)
	}
}

func TestUploadFile_FileNotFound(t *testing.T) {
	mock := &mockService{}
	client := NewServiceClient(mock, false)

	// Try to upload non-existent file
	_, err := client.UploadFile("/nonexistent/file.txt", "", nil)

	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
	if !strings.Contains(err.Error(), "failed to open file") {
		t.Errorf("expected 'failed to open file' error, got %v", err)
	}
}

func TestUploadFile_UploadContentError(t *testing.T) {
	// Create a temporary test file
	testFile := createTestFile(t, "test content")
	defer os.Remove(testFile)

	// Create mock service that returns error
	expectedErr := errors.New("upload failed")
	mock := &mockService{
		uploadContentFunc: func(ctx context.Context, req simplecontent.UploadContentRequest) (*simplecontent.Content, error) {
			return nil, expectedErr
		},
	}

	client := NewServiceClient(mock, false)

	// Test upload
	_, err := client.UploadFile(testFile, "", nil)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to upload content") {
		t.Errorf("expected 'failed to upload content' error, got %v", err)
	}
}

func TestUploadFile_GetContentDetailsError(t *testing.T) {
	// Create a temporary test file
	testFile := createTestFile(t, "test content")
	defer os.Remove(testFile)

	expectedID := uuid.New()
	expectedErr := errors.New("get details failed")

	// Create mock service
	mock := &mockService{
		uploadContentFunc: func(ctx context.Context, req simplecontent.UploadContentRequest) (*simplecontent.Content, error) {
			return &simplecontent.Content{
				ID: expectedID,
			}, nil
		},
		getContentDetailsFunc: func(ctx context.Context, contentID uuid.UUID, options ...simplecontent.ContentDetailsOption) (*simplecontent.ContentDetails, error) {
			return nil, expectedErr
		},
	}

	client := NewServiceClient(mock, false)

	// Test upload
	_, err := client.UploadFile(testFile, "", nil)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to get content details") {
		t.Errorf("expected 'failed to get content details' error, got %v", err)
	}
}

func TestUploadFile_WithVerbose(t *testing.T) {
	// Create a temporary test file
	testFile := createTestFile(t, "test content")
	defer os.Remove(testFile)

	expectedID := uuid.New()

	// Create mock service
	mock := &mockService{
		uploadContentFunc: func(ctx context.Context, req simplecontent.UploadContentRequest) (*simplecontent.Content, error) {
			return &simplecontent.Content{
				ID: expectedID,
			}, nil
		},
		getContentDetailsFunc: func(ctx context.Context, contentID uuid.UUID, options ...simplecontent.ContentDetailsOption) (*simplecontent.ContentDetails, error) {
			return &simplecontent.ContentDetails{
				Download: "https://example.com/download",
			}, nil
		},
	}

	// Create client with verbose=true
	client := NewServiceClient(mock, true)

	// Test upload (verbose output will go to stdout, but we just verify it doesn't error)
	resp, err := client.UploadFile(testFile, "", nil)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
}

func TestUploadFile_WithMetadata(t *testing.T) {
	// Create a temporary test file
	testFile := createTestFile(t, "test content")
	defer os.Remove(testFile)

	expectedID := uuid.New()
	expectedMetadata := map[string]interface{}{
		"author": "test-user",
		"tags":   []string{"tag1", "tag2"},
	}

	// Create mock service
	mock := &mockService{
		uploadContentFunc: func(ctx context.Context, req simplecontent.UploadContentRequest) (*simplecontent.Content, error) {
			// Verify metadata is passed through
			if req.CustomMetadata == nil {
				t.Error("expected custom metadata to be set")
			}
			if req.CustomMetadata["author"] != expectedMetadata["author"] {
				t.Errorf("expected author %v, got %v", expectedMetadata["author"], req.CustomMetadata["author"])
			}

			return &simplecontent.Content{
				ID: expectedID,
			}, nil
		},
		getContentDetailsFunc: func(ctx context.Context, contentID uuid.UUID, options ...simplecontent.ContentDetailsOption) (*simplecontent.ContentDetails, error) {
			return &simplecontent.ContentDetails{
				Download: "https://example.com/download",
			}, nil
		},
	}

	client := NewServiceClient(mock, false)

	// Test upload with metadata
	resp, err := client.UploadFile(testFile, "analysis-type", expectedMetadata)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
}
