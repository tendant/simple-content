package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tendant/simple-content/internal/domain"
	memoryRepo "github.com/tendant/simple-content/pkg/repository/memory"
	"github.com/tendant/simple-content/pkg/service"
)

// MockObjectRepository is a mock implementation of repository.ObjectRepository
type MockObjectRepository struct {
	mock.Mock
}

func (m *MockObjectRepository) Create(ctx context.Context, object *domain.Object) error {
	args := m.Called(ctx, object)
	return args.Error(0)
}

func (m *MockObjectRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Object, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Object), args.Error(1)
}

func (m *MockObjectRepository) GetByContentID(ctx context.Context, contentID uuid.UUID) ([]*domain.Object, error) {
	args := m.Called(ctx, contentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Object), args.Error(1)
}

func (m *MockObjectRepository) Update(ctx context.Context, object *domain.Object) error {
	args := m.Called(ctx, object)
	return args.Error(0)
}

func (m *MockObjectRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockObjectRepository) GetByObjectKeyAndStorageBackendName(ctx context.Context, objectKey string, storageBackendName string) (*domain.Object, error) {
	args := m.Called(ctx, objectKey, storageBackendName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Object), args.Error(1)
}

// setupContentHandlerTest creates a ContentHandler with mock repositories for testing
func setupContentHandlerTest() (*ContentHandler, *MockObjectRepository) {
	// Create mock repositories
	mockObjectRepo := new(MockObjectRepository)

	// Setup services with real repositories for everything except objects
	contentService := service.NewContentService(
		memoryRepo.NewContentRepository(),
		memoryRepo.NewContentMetadataRepository(),
	)

	objectService := service.NewObjectService(
		mockObjectRepo, // Use our mock object repository
		memoryRepo.NewObjectMetadataRepository(),
		memoryRepo.NewContentRepository(),
		memoryRepo.NewContentMetadataRepository(),
	)

	return NewContentHandler(contentService, objectService), mockObjectRepo
}

func TestContentHandler_ListObjects_Success(t *testing.T) {
	// Setup
	handler, mockObjectRepo := setupContentHandlerTest()
	router := chi.NewRouter()
	router.Get("/{id}/objects", handler.ListObjects)

	// Create test data
	contentID := uuid.New()
	now := time.Now().UTC()
	objects := []*domain.Object{
		{
			ID:                 uuid.New(),
			ContentID:          contentID,
			StorageBackendName: "s3-default",
			Version:            1,
			ObjectKey:          "test-object-key",
			Status:             domain.ObjectStatusUploaded,
			CreatedAt:          now,
			UpdatedAt:          now,
		},
		{
			ID:                 uuid.New(),
			ContentID:          contentID,
			StorageBackendName: "s3-default",
			Version:            2,
			ObjectKey:          "test-object-key-v2",
			Status:             domain.ObjectStatusUploaded,
			CreatedAt:          now.Add(time.Hour),
			UpdatedAt:          now.Add(time.Hour),
		},
	}

	// Setup mock expectations
	mockObjectRepo.On("GetByContentID", mock.Anything, contentID).Return(objects, nil)

	// Make request
	req := httptest.NewRequest("GET", "/"+contentID.String()+"/objects", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	var resp []ObjectResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	// Should only return the latest version (version 2)
	assert.Equal(t, 1, len(resp))
	assert.Equal(t, objects[1].ID.String(), resp[0].ID)
	assert.Equal(t, objects[1].ContentID.String(), resp[0].ContentID)
	assert.Equal(t, objects[1].StorageBackendName, resp[0].StorageBackendName)
	assert.Equal(t, objects[1].Version, resp[0].Version)
	assert.Equal(t, objects[1].ObjectKey, resp[0].ObjectKey)
	assert.Equal(t, objects[1].Status, resp[0].Status)

	// Verify all expectations were met
	mockObjectRepo.AssertExpectations(t)
}

func TestContentHandler_ListObjects_NoObjects(t *testing.T) {
	// Setup
	handler, mockObjectRepo := setupContentHandlerTest()
	router := chi.NewRouter()
	router.Get("/{id}/objects", handler.ListObjects)

	// Create test data
	contentID := uuid.New()

	// Setup mock expectations - return empty slice
	mockObjectRepo.On("GetByContentID", mock.Anything, contentID).Return([]*domain.Object{}, nil)

	// Make request
	req := httptest.NewRequest("GET", "/"+contentID.String()+"/objects", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "No objects found for content")

	// Verify all expectations were met
	mockObjectRepo.AssertExpectations(t)
}
