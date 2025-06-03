package testutil

import (
	"net/http/httptest"

	"github.com/go-chi/chi/v5"
	"github.com/tendant/simple-content/internal/api"
	repoMemory "github.com/tendant/simple-content/pkg/repository/memory"
	"github.com/tendant/simple-content/pkg/service"
	storageMemory "github.com/tendant/simple-content/pkg/storage/memory"
)

// SetupTestServer creates a test server with all routes configured
func SetupTestServer() *httptest.Server {
	// Create repositories
	contentRepo := repoMemory.NewContentRepository()
	metadataRepo := repoMemory.NewContentMetadataRepository()
	objectRepo := repoMemory.NewObjectRepository()
	objectMetadataRepo := repoMemory.NewObjectMetadataRepository()
	storageBackendRepo := repoMemory.NewStorageBackendRepository()

	// Create storage backend
	storageBackend := storageMemory.NewMemoryBackend()

	// Create services
	contentService := service.NewContentService(contentRepo, metadataRepo)
	objectService := service.NewObjectService(objectRepo, objectMetadataRepo, contentRepo, metadataRepo)
	storageBackendService := service.NewStorageBackendService(storageBackendRepo)

	// Register the memory backend
	objectService.RegisterBackend("memory", storageBackend)

	// Create handlers
	contentHandler := api.NewContentHandler(contentService, objectService)
	objectHandler := api.NewObjectHandler(objectService)
	storageBackendHandler := api.NewStorageBackendHandler(storageBackendService)

	// Create router
	r := chi.NewRouter()

	r.Mount("/content", contentHandler.Routes())
	r.Mount("/object", objectHandler.Routes())
	r.Mount("/storage-backend", storageBackendHandler.Routes())

	// Create test server
	return httptest.NewServer(r)
}
