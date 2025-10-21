package testutil

import (
	"log"
	"net/http/httptest"

	"github.com/go-chi/chi/v5"
	"github.com/tendant/simple-content/pkg/simplecontent"
	"github.com/tendant/simple-content/pkg/simplecontent/api"
	memoryrepo "github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
	memorystorage "github.com/tendant/simple-content/pkg/simplecontent/storage/memory"
)

// SetupTestServer creates a test server with all routes configured
func SetupTestServer() *httptest.Server {
	// Create repository and storage backend
	repo := memoryrepo.New()
	memBackend := memorystorage.New()

	// Create unified service
	svc, err := simplecontent.New(
		simplecontent.WithRepository(repo),
		simplecontent.WithBlobStore("memory", memBackend),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Create storage service for advanced operations
	storageSvc, err := simplecontent.NewStorageService(
		simplecontent.WithRepository(repo),
		simplecontent.WithBlobStore("memory", memBackend),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Create handlers
	contentHandler := api.NewContentHandler(svc, storageSvc)
	filesHandler := api.NewFilesHandler(svc, storageSvc)

	// Create router
	r := chi.NewRouter()

	r.Mount("/content", contentHandler.Routes())
	r.Mount("/files", filesHandler.Routes())

	// Create test server
	return httptest.NewServer(r)
}
