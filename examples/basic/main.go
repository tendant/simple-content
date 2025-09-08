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
	
	// Create content
	fmt.Println("1. Creating content...")
	content, err := svc.CreateContent(ctx, simplecontent.CreateContentRequest{
		OwnerID:     uuid.New(),
		TenantID:    uuid.New(),
		Name:        "My Document",
		Description: "A sample document",
		DocumentType: "text/plain",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Created content: %s\n", content.ID)
	
	// Set content metadata
	fmt.Println("2. Setting content metadata...")
	err = svc.SetContentMetadata(ctx, simplecontent.SetContentMetadataRequest{
		ContentID:   content.ID,
		ContentType: "text/plain",
		Title:       "My Sample Document",
		Description: "This is a sample document for testing",
		Tags:        []string{"sample", "test", "document"},
		FileName:    "sample.txt",
		FileSize:    13, // "Hello, World!" length
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("   Metadata set successfully")
	
	// Create object for storage
	fmt.Println("3. Creating object...")
	object, err := svc.CreateObject(ctx, simplecontent.CreateObjectRequest{
		ContentID:          content.ID,
		StorageBackendName: "memory",
		Version:            1,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Created object: %s\n", object.ID)
	
	// Upload data
	fmt.Println("4. Uploading data...")
	data := strings.NewReader("Hello, World!")
	err = svc.UploadObject(ctx, object.ID, data)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("   Data uploaded successfully")
	
	// Get object metadata
	fmt.Println("5. Retrieving object metadata...")
	metadata, err := svc.GetObjectMetadata(ctx, object.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Object metadata: %+v\n", metadata)
	
	// Download data  
	fmt.Println("6. Downloading data...")
	reader, err := svc.DownloadObject(ctx, object.ID)
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
	fmt.Println("7. Listing content...")
	contents, err := svc.ListContent(ctx, simplecontent.ListContentRequest{
		OwnerID:  content.OwnerID,
		TenantID: content.TenantID,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Found %d contents\n", len(contents))
	
	// Get objects by content ID
	fmt.Println("8. Getting objects for content...")
	objects, err := svc.GetObjectsByContentID(ctx, content.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Found %d objects\n", len(objects))
	
	fmt.Println("=== Example completed successfully! ===")
}