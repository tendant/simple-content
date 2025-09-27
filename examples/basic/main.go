package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/tendant/simple-content/pkg/simplecontent"
	memorystorage "github.com/tendant/simple-content/pkg/simplecontent/storage/memory"
	memoryrepo "github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
)

func main() {
	// Create repository and storage backends
	repo := memoryrepo.New()
	store := memorystorage.New()
	
	// Create service with functional options
	svc, err := simplecontent.New(
		simplecontent.WithRepository(repo),
		simplecontent.WithBlobStore("memory", store),
	)
	if err != nil {
		log.Fatal(err)
	}
	
	ctx := context.Background()
	
	fmt.Println("=== Simple Content Library Example ===")

	// Upload content with data in one step
	fmt.Println("1. Uploading content with data...")
	data := strings.NewReader("Hello, World!")
	content, err := svc.UploadContent(ctx, simplecontent.UploadContentRequest{
		OwnerID:            uuid.New(),
		TenantID:           uuid.New(),
		Name:               "My Document",
		Description:        "A sample document",
		DocumentType:       "text/plain",
		StorageBackendName: "memory",
		Reader:             data,
		FileName:           "sample.txt",
		FileSize:           13, // "Hello, World!" length
		Tags:               []string{"sample", "test", "document"},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Content uploaded successfully: %s\n", content.ID)
	
	// Get content details (includes metadata and URLs)
	fmt.Println("2. Retrieving content details...")
	details, err := svc.GetContentDetails(ctx, content.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Content details: %+v\n", details)

	// Download data using content ID
	fmt.Println("3. Downloading data...")
	reader, err := svc.DownloadContent(ctx, content.ID)
	if err != nil {
		log.Fatal(err)
	}
	defer reader.Close()

	// Read downloaded content
	downloadedContent, err := io.ReadAll(reader)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Downloaded content: %s\n", string(downloadedContent))
	
	// List content
	fmt.Println("4. Listing content...")
	contents, err := svc.ListContent(ctx, simplecontent.ListContentRequest{
		OwnerID:  content.OwnerID,
		TenantID: content.TenantID,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Found %d contents\n", len(contents))
	
	fmt.Println("=== Example completed successfully! ===")
}