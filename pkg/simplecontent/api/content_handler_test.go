package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendant/simple-content/pkg/simplecontent"
	"github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
	memorystorage "github.com/tendant/simple-content/pkg/simplecontent/storage/memory"
)

// setupContentHandlerTest creates a ContentHandler with in-memory repositories for testing
func setupContentHandlerTest(t *testing.T) (*ContentHandler, simplecontent.Service, simplecontent.StorageService) {
	repo := memory.New()
	blobStore := memorystorage.New()

	// Create service with blob store
	service, err := simplecontent.New(
		simplecontent.WithRepository(repo),
		simplecontent.WithBlobStore("memory", blobStore),
		simplecontent.WithEventSink(simplecontent.NewNoopEventSink()),
	)
	require.NoError(t, err)

	// Create storage service with blob store
	storageService, err := simplecontent.NewStorageService(
		simplecontent.WithRepository(repo),
		simplecontent.WithBlobStore("memory", blobStore),
		simplecontent.WithEventSink(simplecontent.NewNoopEventSink()),
	)
	require.NoError(t, err)

	handler := NewContentHandler(service, storageService)
	return handler, service, storageService
}

func TestContentHandler_CreateContent_Success(t *testing.T) {
	handler, _, _ := setupContentHandlerTest(t)
	router := chi.NewRouter()
	router.Post("/", handler.CreateContent)

	ownerID := uuid.New()
	tenantID := uuid.New()

	reqBody := CreateContentRequest{
		OwnerID:      ownerID.String(),
		TenantID:     tenantID.String(),
		DocumentType: "document",
		FileName:     "test.pdf",
		OwnerType:    "user",
		MimeType:     "application/pdf",
		FileSize:     1024,
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp ContentResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.NotEmpty(t, resp.ID)
	assert.Equal(t, ownerID.String(), resp.OwnerID)
	assert.Equal(t, tenantID.String(), resp.TenantID)
	assert.Equal(t, "document", resp.DocumentType)
	assert.NotEmpty(t, resp.Status)
}

func TestContentHandler_CreateContent_InvalidOwnerID(t *testing.T) {
	handler, _, _ := setupContentHandlerTest(t)
	router := chi.NewRouter()
	router.Post("/", handler.CreateContent)

	reqBody := CreateContentRequest{
		OwnerID:      "invalid-uuid",
		TenantID:     uuid.New().String(),
		DocumentType: "document",
		FileName:     "test.pdf",
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid owner ID")
}

func TestContentHandler_GetContent_Success(t *testing.T) {
	handler, service, _ := setupContentHandlerTest(t)
	router := chi.NewRouter()
	router.Get("/{id}", handler.GetContent)

	// Create a content first
	content, err := service.CreateContent(context.Background(), simplecontent.CreateContentRequest{
		TenantID:     uuid.New(),
		OwnerID:      uuid.New(),
		OwnerType:    "user",
		Name:         "test.pdf",
		DocumentType: "document",
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/"+content.ID.String(), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", content.ID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp ContentResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, content.ID.String(), resp.ID)
	assert.Equal(t, "document", resp.DocumentType)
}

func TestContentHandler_GetContent_NotFound(t *testing.T) {
	handler, _, _ := setupContentHandlerTest(t)
	router := chi.NewRouter()
	router.Get("/{id}", handler.GetContent)

	nonExistentID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/"+nonExistentID.String(), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", nonExistentID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestContentHandler_DeleteContent_Success(t *testing.T) {
	handler, service, _ := setupContentHandlerTest(t)
	router := chi.NewRouter()
	router.Delete("/{id}", handler.DeleteContent)

	// Create a content first
	content, err := service.CreateContent(context.Background(), simplecontent.CreateContentRequest{
		TenantID:     uuid.New(),
		OwnerID:      uuid.New(),
		OwnerType:    "user",
		Name:         "test.pdf",
		DocumentType: "document",
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodDelete, "/"+content.ID.String(), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", content.ID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	// Verify content is deleted
	_, err = service.GetContent(context.Background(), content.ID)
	assert.Error(t, err)
}

func TestContentHandler_GetContentsByIDs_Success(t *testing.T) {
	handler, service, _ := setupContentHandlerTest(t)
	router := chi.NewRouter()
	router.Get("/bulk", handler.GetContentsByIDs)

	// Create multiple contents
	content1, err := service.CreateContent(context.Background(), simplecontent.CreateContentRequest{
		TenantID:     uuid.New(),
		OwnerID:      uuid.New(),
		OwnerType:    "user",
		Name:         "test1.pdf",
		DocumentType: "document",
	})
	require.NoError(t, err)

	content2, err := service.CreateContent(context.Background(), simplecontent.CreateContentRequest{
		TenantID:     uuid.New(),
		OwnerID:      uuid.New(),
		OwnerType:    "user",
		Name:         "test2.pdf",
		DocumentType: "document",
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/bulk?id="+content1.ID.String()+"&id="+content2.ID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp []ContentResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Len(t, resp, 2)
}

func TestContentHandler_GetContentsByIDs_MissingIDParameter(t *testing.T) {
	handler, _, _ := setupContentHandlerTest(t)
	router := chi.NewRouter()
	router.Get("/bulk", handler.GetContentsByIDs)

	req := httptest.NewRequest(http.MethodGet, "/bulk", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Missing required 'id' parameter")
}

func TestContentHandler_ListObjects_Success(t *testing.T) {
	handler, service, storageService := setupContentHandlerTest(t)
	router := chi.NewRouter()
	router.Get("/{id}/objects", handler.ListObjects)

	// Create a content
	content, err := service.CreateContent(context.Background(), simplecontent.CreateContentRequest{
		TenantID:     uuid.New(),
		OwnerID:      uuid.New(),
		OwnerType:    "user",
		Name:         "test.pdf",
		DocumentType: "document",
	})
	require.NoError(t, err)

	// Create an object
	object, err := storageService.CreateObject(context.Background(), simplecontent.CreateObjectRequest{
		ContentID:          content.ID,
		StorageBackendName: "memory",
		Version:            1,
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/"+content.ID.String()+"/objects", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", content.ID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp []ObjectResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Len(t, resp, 1)
	assert.Equal(t, object.ID.String(), resp[0].ID)
}

func TestContentHandler_CreateDerivedContent_Success(t *testing.T) {
	handler, service, _ := setupContentHandlerTest(t)
	router := chi.NewRouter()
	router.Post("/{id}/derived", handler.CreateDerivedContent)

	// Create parent content
	parentContent, err := service.CreateContent(context.Background(), simplecontent.CreateContentRequest{
		TenantID:     uuid.New(),
		OwnerID:      uuid.New(),
		OwnerType:    "user",
		Name:         "parent.pdf",
		DocumentType: "document",
	})
	require.NoError(t, err)

	// Update parent content status to "uploaded" so it's ready for derivation
	err = service.UpdateContentStatus(context.Background(), parentContent.ID, simplecontent.ContentStatusUploaded)
	require.NoError(t, err)

	reqBody := CreateDerivedContentRequest{
		DerivationType: "thumbnail",
		OwnerID:        parentContent.OwnerID.String(),
		TenantID:       parentContent.TenantID.String(),
		DerivationParams: map[string]interface{}{
			"size": "256x256",
		},
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/"+parentContent.ID.String()+"/derived", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", parentContent.ID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp CreateDerivedContentResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, parentContent.ID.String(), resp.ParentContentID)
	assert.NotEmpty(t, resp.DerivedContentID)
	assert.Equal(t, "thumbnail", resp.DerivationType)
}

func TestGetLatestVersionObject(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name     string
		objects  []*simplecontent.Object
		expected *simplecontent.Object
	}{
		{
			name:     "empty slice",
			objects:  []*simplecontent.Object{},
			expected: nil,
		},
		{
			name: "single object",
			objects: []*simplecontent.Object{
				{ID: uuid.New(), Version: 1, CreatedAt: now},
			},
			expected: &simplecontent.Object{Version: 1},
		},
		{
			name: "multiple objects with different versions",
			objects: []*simplecontent.Object{
				{ID: uuid.New(), Version: 1, CreatedAt: now},
				{ID: uuid.New(), Version: 3, CreatedAt: now},
				{ID: uuid.New(), Version: 2, CreatedAt: now},
			},
			expected: &simplecontent.Object{Version: 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetLatestVersionObject(tt.objects)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tt.expected.Version, result.Version)
			}
		})
	}
}
