package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/tendant/simple-content/pkg/simplecontent"
	"github.com/tendant/simple-content/pkg/simplecontentpresets"
)

func main() {
	fmt.Println("=== Simple Content - Development Preset Example ===")
	fmt.Println()

	// Create service with development preset
	// - In-memory database (no setup required)
	// - Filesystem storage at ./dev-data/
	// - One-line setup!
	svc, cleanup, err := simplecontentpresets.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	defer cleanup() // Remove ./dev-data/ when done

	fmt.Println("✓ Service created with development preset")
	fmt.Println("  - Database: In-memory")
	fmt.Println("  - Storage: Filesystem (./dev-data/)")
	fmt.Println()

	ctx := context.Background()

	// Upload a document
	fmt.Println("Uploading document...")
	content, err := svc.UploadContent(ctx, simplecontent.UploadContentRequest{
		OwnerID:      uuid.New(),
		TenantID:     uuid.New(),
		Name:         "Development Guide",
		DocumentType: "text/plain",
		Reader:       strings.NewReader("This is a guide for local development."),
		FileName:     "dev-guide.txt",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✓ Document uploaded: %s (ID: %s)\n", content.Name, content.ID)
	fmt.Println()

	// Upload an image
	fmt.Println("Uploading image...")
	imageContent := "fake-image-data-for-demo"
	image, err := svc.UploadContent(ctx, simplecontent.UploadContentRequest{
		OwnerID:      uuid.New(),
		TenantID:     uuid.New(),
		Name:         "Development Screenshot",
		DocumentType: "image/png",
		Reader:       strings.NewReader(imageContent),
		FileName:     "screenshot.png",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✓ Image uploaded: %s (ID: %s)\n", image.Name, image.ID)
	fmt.Println()

	// Create a derived thumbnail
	fmt.Println("Creating thumbnail...")
	thumbnail, err := svc.UploadDerivedContent(ctx, simplecontent.UploadDerivedContentRequest{
		ParentID:       image.ID,
		DerivationType: "thumbnail",
		Variant:        "thumbnail_256",
		Reader:         strings.NewReader("fake-thumbnail-data"),
		FileName:       "screenshot_thumb.png",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✓ Thumbnail created (ID: %s)\n", thumbnail.ID)
	fmt.Println()

	// Download content
	fmt.Println("Downloading document...")
	downloadReader, err := svc.DownloadContent(ctx, content.ID)
	if err != nil {
		log.Fatal(err)
	}
	defer downloadReader.Close()
	fmt.Println("✓ Document downloaded successfully")
	fmt.Println()

	// Get content details
	fmt.Println("Retrieving content details...")
	details, err := svc.GetContentDetails(ctx, image.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✓ Content Details:\n")
	fmt.Printf("  ID: %s\n", details.ID)
	fmt.Printf("  File: %s\n", details.FileName)
	fmt.Printf("  Type: %s\n", details.MimeType)
	fmt.Printf("  Ready: %v\n", details.Ready)
	if details.Thumbnail != "" {
		fmt.Printf("  Thumbnail URL: %s\n", details.Thumbnail)
	}
	fmt.Println()

	fmt.Println("=== Development Example Complete ===")
	fmt.Println("Note: ./dev-data/ will be cleaned up automatically")
}
