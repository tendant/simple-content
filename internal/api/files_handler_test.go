package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendant/simple-content/internal/storage"
	"github.com/tendant/simple-content/pkg/model"
	memoryRepo "github.com/tendant/simple-content/pkg/repository/memory"
	"github.com/tendant/simple-content/pkg/service"
)

var ErrObjectNotFound = errors.New("object not found")

// MockStorageBackend is a mock storage backend for testing that supports URLs
type MockStorageBackend struct {
	mu      sync.RWMutex
	objects map[string][]byte
}

func NewMockStorageBackend() storage.Backend {
	return &MockStorageBackend{
		objects: make(map[string][]byte),
	}
}

func (b *MockStorageBackend) GetObjectMeta(ctx context.Context, objectKey string) (*storage.ObjectMeta, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	data, exists := b.objects[objectKey]
	if !exists {
		return nil, ErrObjectNotFound
	}

	return &storage.ObjectMeta{
		Key:      objectKey,
		Size:     int64(len(data)),
		Metadata: make(map[string]string),
	}, nil
}

func (b *MockStorageBackend) GetUploadURL(ctx context.Context, objectKey string) (string, error) {
	return "https://mock-storage.example.com/upload/" + objectKey, nil
}

func (b *MockStorageBackend) Upload(ctx context.Context, objectKey string, reader io.Reader) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	b.objects[objectKey] = data
	return nil
}

func (b *MockStorageBackend) GetDownloadURL(ctx context.Context, objectKey string, downloadFilename string) (string, error) {
	return "https://mock-storage.example.com/download/" + objectKey, nil
}

func (b *MockStorageBackend) GetPreviewURL(ctx context.Context, objectKey string) (string, error) {
	return "https://mock-storage.example.com/preview/" + objectKey, nil
}

func (b *MockStorageBackend) Download(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	data, exists := b.objects[objectKey]
	if !exists {
		return nil, ErrObjectNotFound
	}

	return io.NopCloser(bytes.NewReader(data)), nil
}

func (b *MockStorageBackend) Delete(ctx context.Context, objectKey string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.objects[objectKey]; !exists {
		return ErrObjectNotFound
	}

	delete(b.objects, objectKey)
	return nil
}

func setupFilesHandler() *FilesHandler {
	// Setup repositories
	contentRepo := memoryRepo.NewContentRepository()
	contentMetadataRepo := memoryRepo.NewContentMetadataRepository()
	objectRepo := memoryRepo.NewObjectRepository()
	objectMetadataRepo := memoryRepo.NewObjectMetadataRepository()
	storageBackendRepo := memoryRepo.NewStorageBackendRepository()

	// Setup mock storage backend that supports URLs
	mockBackend := NewMockStorageBackend()

	// Setup services
	contentService := service.NewContentService(contentRepo, contentMetadataRepo)
	objectService := service.NewObjectService(objectRepo, objectMetadataRepo, contentRepo, contentMetadataRepo)

	// Register the mock backend
	objectService.RegisterBackend("s3-default", mockBackend)

	// Create default storage backend
	ctx := context.Background()
	storageBackendService := service.NewStorageBackendService(storageBackendRepo)
	_, err := storageBackendService.CreateStorageBackend(ctx, "s3-default", "memory", map[string]interface{}{})
	if err != nil {
		panic(err)
	}

	return NewFilesHandler(contentService, objectService)
}

func TestFilesHandler_CreateFile(t *testing.T) {
	handler := setupFilesHandler()
	router := chi.NewRouter()
	router.Mount("/files", handler.Routes())

	// Create request
	req := CreateFileRequest{
		OwnerID:      uuid.New().String(),
		OwnerType:    "user",
		TenantID:     uuid.New().String(),
		FileName:     "test.txt",
		MimeType:     "text/plain",
		FileSize:     1024,
		DocumentType: "document", // Add the required DocumentType field
	}

	reqBody, err := json.Marshal(req)
	require.NoError(t, err)

	// Make request
	httpReq := httptest.NewRequest("POST", "/files/", bytes.NewReader(reqBody))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	// Check response
	if w.Code != http.StatusOK {
		t.Logf("Response body: %s", w.Body.String())
	}
	assert.Equal(t, http.StatusOK, w.Code)

	var resp CreateFileResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.NotEmpty(t, resp.ContentID)
	assert.NotEmpty(t, resp.ObjectID)
	assert.NotEmpty(t, resp.UploadURL)
	assert.NotZero(t, resp.CreatedAt)
}

func TestFilesHandler_CompleteUpload(t *testing.T) {
	handler := setupFilesHandler()
	router := chi.NewRouter()
	router.Mount("/files", handler.Routes())

	// First create a file
	ownerID := uuid.New()
	tenantID := uuid.New()
	createParams := service.CreateContentParams{
		OwnerID:  ownerID,
		TenantID: tenantID,
	}
	content, err := handler.contentService.CreateContent(context.Background(), createParams)
	require.NoError(t, err)

	createObjectParams := service.CreateObjectParams{
		ContentID:          content.ID,
		StorageBackendName: "s3-default",
		Version:            1,
	}
	object, err := handler.objectService.CreateObject(context.Background(), createObjectParams)
	require.NoError(t, err)

	// Set content metadata which is required for CompleteUpload
	metadataParams := service.SetContentMetadataParams{
		ContentID:   content.ID,
		ContentType: "text/plain",
		Title:       "Test File",
		Description: "Test file for upload",
		Tags:        []string{"test", "upload"},
		FileSize:    int64(len("test file content")),
		CreatedBy:   "test-user",
	}
	err = handler.contentService.SetContentMetadata(context.Background(), metadataParams)
	require.NoError(t, err)

	// Simulate uploading the file to storage (this is what would happen client-side)
	backend, err := handler.objectService.GetBackend("s3-default")
	require.NoError(t, err)

	// Upload some test data
	testData := "test file content"
	err = backend.Upload(context.Background(), object.ObjectKey, bytes.NewReader([]byte(testData)))
	require.NoError(t, err)

	// Make request
	httpReq := httptest.NewRequest("POST", "/files/"+content.ID.String()+"/complete", nil)
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	// Check response
	if w.Code != http.StatusOK {
		t.Logf("Response body: %s", w.Body.String())
	}
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "completed", resp["status"])

	// Verify content status is updated to uploaded
	updatedContent, err := handler.contentService.GetContent(context.Background(), content.ID)
	require.NoError(t, err)
	assert.Equal(t, model.ContentStatusUploaded, updatedContent.Status)

	// Verify object status is updated to uploaded
	updatedObject, err := handler.objectService.GetObject(context.Background(), object.ID)
	require.NoError(t, err)
	assert.Equal(t, model.ObjectStatusUploaded, updatedObject.Status)
}

func TestFilesHandler_UpdateMetadata(t *testing.T) {
	handler := setupFilesHandler()
	router := chi.NewRouter()
	router.Mount("/files", handler.Routes())

	// First create a file
	ownerID := uuid.New()
	tenantID := uuid.New()
	createParams := service.CreateContentParams{
		OwnerID:  ownerID,
		TenantID: tenantID,
	}
	content, err := handler.contentService.CreateContent(context.Background(), createParams)
	require.NoError(t, err)

	// Create update metadata request
	req := UpdateMetadataRequest{
		Title:       "Test File",
		Description: "A test file",
		Tags:        []string{"test", "file"},
		Metadata: map[string]interface{}{
			"custom_field": "custom_value",
		},
	}

	reqBody, err := json.Marshal(req)
	require.NoError(t, err)

	// Make request
	httpReq := httptest.NewRequest("PATCH", "/files/"+content.ID.String(), bytes.NewReader(reqBody))
	httpReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	// Check response
	if w.Code != http.StatusOK {
		t.Logf("Response body: %s", w.Body.String())
	}
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "updated", resp["status"])
}

func TestFilesHandler_GetFileInfo(t *testing.T) {
	handler := setupFilesHandler()
	router := chi.NewRouter()
	router.Mount("/files", handler.Routes())

	// First create a file with metadata
	ownerID := uuid.New()
	tenantID := uuid.New()
	createParams := service.CreateContentParams{
		OwnerID:  ownerID,
		TenantID: tenantID,
	}
	content, err := handler.contentService.CreateContent(context.Background(), createParams)
	require.NoError(t, err)

	createObjectParams := service.CreateObjectParams{
		ContentID:          content.ID,
		StorageBackendName: "s3-default",
		Version:            1,
	}
	object, err := handler.objectService.CreateObject(context.Background(), createObjectParams)
	require.NoError(t, err)

	// Set some metadata
	err = handler.objectService.SetObjectMetadata(context.Background(), object.ID, map[string]interface{}{
		"filename": "test.txt",
	})
	require.NoError(t, err)

	// Make request
	httpReq := httptest.NewRequest("GET", "/files/"+content.ID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, httpReq)

	// Check response
	if w.Code != http.StatusOK {
		t.Logf("Response body: %s", w.Body.String())
	}
	assert.Equal(t, http.StatusOK, w.Code)

	var resp FileInfoResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, content.ID.String(), resp.ContentID)
	assert.NotEmpty(t, resp.FileName)
	assert.NotEmpty(t, resp.DownloadURL)
	assert.NotEmpty(t, resp.PreviewURL)
	assert.NotZero(t, resp.CreatedAt)
	assert.NotZero(t, resp.UpdatedAt)
	assert.NotEmpty(t, resp.Status)
}
