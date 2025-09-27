package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/nfnt/resize"
	"github.com/tendant/simple-content/pkg/simplecontent"
	fsstorage "github.com/tendant/simple-content/pkg/simplecontent/storage/fs"
	memoryrepo "github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
)

// ThumbnailService wraps the simple-content service with thumbnail generation capabilities
type ThumbnailService struct {
	svc simplecontent.Service
}

// NewThumbnailService creates a new thumbnail service
func NewThumbnailService() (*ThumbnailService, error) {
	// Set up repository and storage
	repo := memoryrepo.New()

	// Use filesystem storage for this example
	fsStore, err := fsstorage.New(fsstorage.Config{
		BaseDir:   "./data/storage",
		URLPrefix: "http://localhost:8080/files/",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create filesystem storage: %w", err)
	}

	// Create service
	svc, err := simplecontent.New(
		simplecontent.WithRepository(repo),
		simplecontent.WithBlobStore("filesystem", fsStore),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}

	return &ThumbnailService{svc: svc}, nil
}

// UploadImageRequest contains parameters for uploading an image
type UploadImageRequest struct {
	FilePath    string
	OwnerID     uuid.UUID
	TenantID    uuid.UUID
	Name        string
	Description string
	Tags        []string
}

// UploadImageResponse contains the result of an image upload
type UploadImageResponse struct {
	Content    *simplecontent.Content
	Object     *simplecontent.Object
	Thumbnails []ThumbnailInfo
}

// ThumbnailInfo contains information about a generated thumbnail
type ThumbnailInfo struct {
	Content *simplecontent.Content
	Object  *simplecontent.Object
	Size    int
}

// UploadImageWithThumbnails uploads an image and generates thumbnails
func (ts *ThumbnailService) UploadImageWithThumbnails(ctx context.Context, req UploadImageRequest) (*UploadImageResponse, error) {
	// 1. Upload the original image
	content, err := ts.uploadOriginalImage(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to upload original image: %w", err)
	}

	log.Printf("Uploaded original image: content=%s", content.ID)

	// 2. Generate thumbnails in multiple sizes
	thumbnailSizes := []int{128, 256, 512}
	var thumbnails []ThumbnailInfo

	for _, size := range thumbnailSizes {
		thumbnail, err := ts.generateThumbnail(ctx, content.ID, req.FilePath, size)
		if err != nil {
			log.Printf("Failed to generate %dpx thumbnail: %v", size, err)
			continue
		}
		thumbnails = append(thumbnails, *thumbnail)
		log.Printf("Generated %dpx thumbnail: content=%s", size, thumbnail.Content.ID)
	}

	return &UploadImageResponse{
		Content:    content,
		Object:     nil, // Object is no longer exposed
		Thumbnails: thumbnails,
	}, nil
}

// uploadOriginalImage uploads the original image file using the unified API
func (ts *ThumbnailService) uploadOriginalImage(ctx context.Context, req UploadImageRequest) (*simplecontent.Content, error) {
	// Open and read the file
	file, err := os.Open(req.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Determine MIME type based on extension
	ext := strings.ToLower(filepath.Ext(req.FilePath))
	var mimeType string
	switch ext {
	case ".jpg", ".jpeg":
		mimeType = "image/jpeg"
	case ".png":
		mimeType = "image/png"
	case ".gif":
		mimeType = "image/gif"
	default:
		mimeType = "application/octet-stream"
	}

	// Upload content with data in one step
	content, err := ts.svc.UploadContent(ctx, simplecontent.UploadContentRequest{
		OwnerID:            req.OwnerID,
		TenantID:           req.TenantID,
		Name:               req.Name,
		Description:        req.Description,
		DocumentType:       mimeType,
		StorageBackendName: "filesystem",
		Reader:             file,
		FileName:           filepath.Base(req.FilePath),
		FileSize:           fileInfo.Size(),
		Tags:               append(req.Tags, "original", "image"),
		CustomMetadata: map[string]interface{}{
			"file_extension":  ext,
			"upload_source":   "thumbnail_generation_example",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload content: %w", err)
	}

	return content, nil
}

// generateThumbnail generates a thumbnail of the specified size using unified API
func (ts *ThumbnailService) generateThumbnail(ctx context.Context, parentContentID uuid.UUID, originalFilePath string, size int) (*ThumbnailInfo, error) {
	// Get parent content for metadata
	parentContent, err := ts.svc.GetContent(ctx, parentContentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get parent content: %w", err)
	}

	// Create thumbnail variant name
	variant := fmt.Sprintf("thumbnail_%d", size)

	// Generate thumbnail image data
	thumbnailData, err := ts.resizeImage(originalFilePath, size)
	if err != nil {
		return nil, fmt.Errorf("failed to resize image: %w", err)
	}

	// Upload derived content with thumbnail data in one step
	thumbnailContent, err := ts.svc.UploadDerivedContent(ctx, simplecontent.UploadDerivedContentRequest{
		ParentID:           parentContentID,
		OwnerID:            parentContent.OwnerID,
		TenantID:           parentContent.TenantID,
		DerivationType:     "thumbnail",
		Variant:            variant,
		StorageBackendName: "filesystem",
		Reader:             bytes.NewReader(thumbnailData),
		FileName:           fmt.Sprintf("thumbnail_%dpx.jpg", size),
		FileSize:           int64(len(thumbnailData)),
		Tags:               []string{"thumbnail", "derived", fmt.Sprintf("%dpx", size)},
		Metadata: map[string]interface{}{
			"thumbnail_size": size,
			"parent_id":      parentContentID.String(),
			"generated_by":   "thumbnail_generation_example",
			"source_type":    "image_resize",
			"algorithm":      "lanczos3",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload derived content: %w", err)
	}

	return &ThumbnailInfo{
		Content: thumbnailContent,
		Object:  nil, // Object is no longer exposed
		Size:    size,
	}, nil
}

// resizeImage resizes an image file to the specified size
func (ts *ThumbnailService) resizeImage(filePath string, size int) ([]byte, error) {
	// Open the original image
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image file: %w", err)
	}
	defer file.Close()

	// Decode the image
	var img image.Image
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".jpg", ".jpeg":
		img, err = jpeg.Decode(file)
	case ".png":
		img, err = png.Decode(file)
	default:
		img, _, err = image.Decode(file)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Resize the image (maintaining aspect ratio)
	thumbnail := resize.Thumbnail(uint(size), uint(size), img, resize.Lanczos3)

	// Encode the thumbnail as JPEG
	var buf bytes.Buffer
	err = jpeg.Encode(&buf, thumbnail, &jpeg.Options{Quality: 80})
	if err != nil {
		return nil, fmt.Errorf("failed to encode thumbnail: %w", err)
	}

	return buf.Bytes(), nil
}

// ListContentWithThumbnails lists content and their associated thumbnails
func (ts *ThumbnailService) ListContentWithThumbnails(ctx context.Context, ownerID, tenantID uuid.UUID) error {
	// List all content
	contents, err := ts.svc.ListContent(ctx, simplecontent.ListContentRequest{
		OwnerID:  ownerID,
		TenantID: tenantID,
	})
	if err != nil {
		return fmt.Errorf("failed to list content: %w", err)
	}

	fmt.Printf("\n=== Content Listing ===\n")
	for _, content := range contents {
		fmt.Printf("Content: %s\n", content.ID)
		fmt.Printf("  Name: %s\n", content.Name)
		fmt.Printf("  Type: %s\n", content.DocumentType)
		fmt.Printf("  Status: %s\n", content.Status)

		if content.DerivationType != "" {
			fmt.Printf("  Derivation Type: %s\n", content.DerivationType)
		}

		// Get content details (replaces old metadata call)
		details, err := ts.svc.GetContentDetails(ctx, content.ID)
		if err == nil {
			fmt.Printf("  File Name: %s\n", details.FileName)
			fmt.Printf("  File Size: %d bytes\n", details.FileSize)
			fmt.Printf("  MIME Type: %s\n", details.MimeType)
			fmt.Printf("  Tags: %v\n", details.Tags)
		}

		// Note: Object details are now abstracted - use content details for URLs

		// If this is original content, list derived content
		if content.DerivationType == "" { // Original content
			derived, err := ts.svc.ListDerivedContent(ctx, simplecontent.WithParentID(content.ID))
			if err == nil && len(derived) > 0 {
				fmt.Printf("  Derived Content:\n")
				for _, d := range derived {
					fmt.Printf("    - %s (Type: %s)\n", d.ContentID, d.DerivationType)
				}
			}
		}

		fmt.Println()
	}

	return nil
}

// DownloadThumbnail downloads a thumbnail to a local file
func (ts *ThumbnailService) DownloadThumbnail(ctx context.Context, contentID uuid.UUID, outputPath string) error {
	// Download content data directly using content ID
	reader, err := ts.svc.DownloadContent(ctx, contentID)
	if err != nil {
		return fmt.Errorf("failed to download content: %w", err)
	}
	defer reader.Close()

	// Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Copy data
	_, err = io.Copy(outFile, reader)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("Downloaded thumbnail to: %s\n", outputPath)
	return nil
}

func main() {
	ctx := context.Background()

	fmt.Println("=== Simple Content Library - Thumbnail Generation Example ===")

	// Create thumbnail service
	ts, err := NewThumbnailService()
	if err != nil {
		log.Fatal("Failed to create thumbnail service:", err)
	}

	// Create sample data directories
	os.MkdirAll("./data/storage", 0755)
	os.MkdirAll("./data/downloads", 0755)

	// For this example, we'll create a simple colored image if none exists
	imagePath := "./data/sample_image.jpg"
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		err = createSampleImage(imagePath)
		if err != nil {
			log.Fatal("Failed to create sample image:", err)
		}
		fmt.Printf("Created sample image: %s\n", imagePath)
	}

	// Generate UUIDs for owner and tenant
	ownerID := uuid.New()
	tenantID := uuid.New()

	// Upload image with thumbnails
	fmt.Println("\n1. Uploading image with thumbnail generation...")
	response, err := ts.UploadImageWithThumbnails(ctx, UploadImageRequest{
		FilePath:    imagePath,
		OwnerID:     ownerID,
		TenantID:    tenantID,
		Name:        "Sample Image",
		Description: "A sample image for thumbnail generation demo",
		Tags:        []string{"demo", "sample"},
	})
	if err != nil {
		log.Fatal("Failed to upload image:", err)
	}

	fmt.Printf("Upload complete! Generated %d thumbnails\n", len(response.Thumbnails))

	// List all content with thumbnails
	fmt.Println("\n2. Listing all content and thumbnails...")
	err = ts.ListContentWithThumbnails(ctx, ownerID, tenantID)
	if err != nil {
		log.Fatal("Failed to list content:", err)
	}

	// Download thumbnails
	fmt.Println("\n3. Downloading thumbnails...")
	for _, thumbnail := range response.Thumbnails {
		outputPath := fmt.Sprintf("./data/downloads/thumbnail_%dpx.jpg", thumbnail.Size)
		err = ts.DownloadThumbnail(ctx, thumbnail.Content.ID, outputPath)
		if err != nil {
			log.Printf("Failed to download %dpx thumbnail: %v", thumbnail.Size, err)
		}
	}

	fmt.Println("\n=== Example completed successfully! ===")
	fmt.Println("Check ./data/downloads/ for generated thumbnails")
}

// createSampleImage creates a simple colored image for the demo
func createSampleImage(path string) error {
	// Create a simple 400x300 colored image
	img := image.NewRGBA(image.Rect(0, 0, 400, 300))

	// Fill with a gradient
	for y := 0; y < 300; y++ {
		for x := 0; x < 400; x++ {
			r := uint8(x * 255 / 400)
			g := uint8(y * 255 / 300)
			b := uint8((x + y) * 255 / 700)
			img.Set(x, y, color.NRGBA{R: r, G: g, B: b, A: 255})
		}
	}

	// Save as JPEG
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return jpeg.Encode(file, img, &jpeg.Options{Quality: 90})
}