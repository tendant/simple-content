package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/nfnt/resize"
	"github.com/tendant/simple-content/pkg/simplecontent"
	memoryrepo "github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
	fsstorage "github.com/tendant/simple-content/pkg/simplecontent/storage/fs"
)

// PhotoGallery demonstrates a complete photo management application
type PhotoGallery struct {
	service simplecontent.Service
	userID  uuid.UUID
	tenant  uuid.UUID
}

func main() {
	fmt.Println("📸 Simple Content - Photo Gallery Example")
	fmt.Println("==========================================\n")

	// Setup
	gallery, err := NewPhotoGallery("./gallery-data")
	if err != nil {
		log.Fatal(err)
	}

	// Demo workflow
	if err := gallery.Run(); err != nil {
		log.Fatal(err)
	}
}

// NewPhotoGallery creates a new photo gallery instance
func NewPhotoGallery(dataDir string) (*PhotoGallery, error) {
	// Create data directory
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Setup filesystem storage
	fsBackend, err := fsstorage.New(fsstorage.Config{
		BaseDir: dataDir,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create storage backend: %w", err)
	}

	// Create service with memory repository and filesystem storage
	svc, err := simplecontent.New(
		simplecontent.WithRepository(memoryrepo.New()),
		simplecontent.WithBlobStore("fs", fsBackend),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}

	return &PhotoGallery{
		service: svc,
		userID:  uuid.New(),
		tenant:  uuid.New(),
	}, nil
}

// Run executes the photo gallery demo
func (g *PhotoGallery) Run() error {
	ctx := context.Background()

	// Step 1: Upload a photo
	fmt.Println("Step 1: Uploading a photo...")
	photoID, err := g.uploadSamplePhoto(ctx)
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}
	fmt.Printf("✅ Photo uploaded with ID: %s\n\n", photoID)

	// Step 2: Generate thumbnails
	fmt.Println("Step 2: Generating thumbnails...")
	if err := g.generateThumbnails(ctx, photoID); err != nil {
		return fmt.Errorf("thumbnail generation failed: %w", err)
	}
	fmt.Println("✅ Thumbnails generated: 128x128, 256x256, 512x512\n")

	// Step 3: Add metadata
	fmt.Println("Step 3: Adding photo metadata...")
	if err := g.addMetadata(ctx, photoID); err != nil {
		return fmt.Errorf("metadata failed: %w", err)
	}
	fmt.Println("✅ Metadata added\n")

	// Step 4: Retrieve photo details
	fmt.Println("Step 4: Retrieving photo details...")
	if err := g.displayPhotoDetails(ctx, photoID); err != nil {
		return fmt.Errorf("retrieval failed: %w", err)
	}

	// Step 5: List all photos
	fmt.Println("\nStep 5: Listing all photos in gallery...")
	if err := g.listPhotos(ctx); err != nil {
		return fmt.Errorf("listing failed: %w", err)
	}

	fmt.Println("\n🎉 Photo gallery demo completed successfully!")
	fmt.Println("📁 Check ./gallery-data/ to see the stored files")

	return nil
}

// uploadSamplePhoto creates and uploads a sample photo
func (g *PhotoGallery) uploadSamplePhoto(ctx context.Context) (uuid.UUID, error) {
	// Create a simple test image (gradient)
	img := createSampleImage(800, 600)

	// Encode to JPEG
	buf := new(bytes.Buffer)
	if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: 90}); err != nil {
		return uuid.Nil, err
	}

	// Upload to content service
	content, err := g.service.UploadContent(ctx, simplecontent.UploadContentRequest{
		OwnerID:      g.userID,
		TenantID:     g.tenant,
		Name:         "Sunset at the Beach",
		DocumentType: "photo",
		Reader:       bytes.NewReader(buf.Bytes()),
		FileName:     "sunset.jpg",
		MimeType:     "image/jpeg",
	})
	if err != nil {
		return uuid.Nil, err
	}

	return content.ID, nil
}

// generateThumbnails creates multiple thumbnail sizes
func (g *PhotoGallery) generateThumbnails(ctx context.Context, photoID uuid.UUID) error {
	// Download original photo
	reader, err := g.service.DownloadContent(ctx, photoID)
	if err != nil {
		return err
	}
	defer reader.Close()

	// Decode image
	img, _, err := image.Decode(reader)
	if err != nil {
		return err
	}

	// Generate thumbnails in different sizes
	sizes := []struct {
		size int
		name string
	}{
		{128, "small"},
		{256, "medium"},
		{512, "large"},
	}

	for _, s := range sizes {
		// Resize
		thumbnail := resize.Thumbnail(uint(s.size), uint(s.size), img, resize.Lanczos3)

		// Encode
		buf := new(bytes.Buffer)
		if err := jpeg.Encode(buf, thumbnail, &jpeg.Options{Quality: 85}); err != nil {
			return err
		}

		// Upload as derived content
		_, err := g.service.UploadDerivedContent(ctx, simplecontent.UploadDerivedContentRequest{
			ParentID:       photoID,
			DerivationType: "thumbnail",
			Variant:        fmt.Sprintf("thumbnail_%d", s.size),
			Reader:         bytes.NewReader(buf.Bytes()),
			FileName:       fmt.Sprintf("sunset_thumb_%s.jpg", s.name),
			MimeType:       "image/jpeg",
		})
		if err != nil {
			return err
		}

		fmt.Printf("  ✓ Created %dx%d thumbnail\n", s.size, s.size)
	}

	return nil
}

// addMetadata adds rich metadata to the photo
func (g *PhotoGallery) addMetadata(ctx context.Context, photoID uuid.UUID) error {
	return g.service.SetContentMetadata(ctx, simplecontent.SetContentMetadataRequest{
		ContentID:   photoID,
		ContentType: "image/jpeg",
		Title:       "Sunset at the Beach",
		Description: "Beautiful sunset over the Pacific Ocean",
		Tags:        []string{"sunset", "beach", "nature", "ocean"},
		FileName:    "sunset.jpg",
		FileSize:    52428,
		CreatedBy:   g.userID.String(),
		CustomMetadata: map[string]interface{}{
			"camera":       "Canon EOS R5",
			"lens":         "RF 24-70mm f/2.8",
			"aperture":     "f/8",
			"shutter":      "1/250",
			"iso":          100,
			"focal_length": "35mm",
			"location":     "Malibu, California",
			"date_taken":   "2024-10-20",
			"width":        800,
			"height":       600,
		},
	})
}

// displayPhotoDetails retrieves and displays photo information
func (g *PhotoGallery) displayPhotoDetails(ctx context.Context, photoID uuid.UUID) error {
	// Get content details
	details, err := g.service.GetContentDetails(ctx, photoID)
	if err != nil {
		return err
	}

	// Get metadata
	metadata, err := g.service.GetContentMetadata(ctx, photoID)
	if err != nil {
		return err
	}

	// Display information
	fmt.Println("\n📷 Photo Details:")
	fmt.Println("─────────────────────────────────────────")
	fmt.Printf("Title: %s\n", metadata.Metadata["title"])
	fmt.Printf("Description: %s\n", metadata.Metadata["description"])
	fmt.Printf("Camera: %s\n", metadata.Metadata["camera"])
	fmt.Printf("Settings: %s @ %s, ISO %v\n",
		metadata.Metadata["aperture"],
		metadata.Metadata["shutter"],
		metadata.Metadata["iso"])
	fmt.Printf("Location: %s\n", metadata.Metadata["location"])
	fmt.Printf("Date: %s\n", metadata.Metadata["date_taken"])
	fmt.Printf("Dimensions: %vx%v\n",
		metadata.Metadata["width"],
		metadata.Metadata["height"])

	// List derived content
	derived, err := g.service.ListDerivedContent(ctx,
		simplecontent.WithParentID(photoID),
		simplecontent.WithDerivationType("thumbnail"),
	)
	if err != nil {
		return err
	}

	fmt.Printf("\nThumbnails: %d available\n", len(derived))
	for _, d := range derived {
		fmt.Printf("  - %s (%s)\n", d.Variant, d.Status)
	}

	return nil
}

// listPhotos lists all photos in the gallery
func (g *PhotoGallery) listPhotos(ctx context.Context) error {
	// Get all uploaded content
	photos, err := g.service.GetContentByStatus(ctx, simplecontent.ContentStatusUploaded)
	if err != nil {
		return err
	}

	fmt.Printf("\n📁 Gallery contains %d photo(s):\n", len(photos))
	fmt.Println("─────────────────────────────────────────")

	for i, photo := range photos {
		metadata, err := g.service.GetContentMetadata(ctx, photo.ID)
		if err != nil {
			continue
		}

		title := "Untitled"
		if t, ok := metadata.Metadata["title"].(string); ok {
			title = t
		}

		fmt.Printf("%d. %s\n", i+1, title)
		fmt.Printf("   ID: %s\n", photo.ID)
		fmt.Printf("   Created: %s\n", photo.CreatedAt.Format("2006-01-02 15:04:05"))

		// Count thumbnails
		derived, _ := g.service.ListDerivedContent(ctx,
			simplecontent.WithParentID(photo.ID),
		)
		fmt.Printf("   Thumbnails: %d\n", len(derived))
		fmt.Println()
	}

	return nil
}

// createSampleImage generates a gradient image for demo purposes
func createSampleImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Create a sunset-like gradient
	for y := 0; y < height; y++ {
		// Color changes from orange (top) to purple (bottom)
		ratio := float64(y) / float64(height)

		r := uint8(255 - ratio*100)  // 255 -> 155
		g := uint8(150 - ratio*150)  // 150 -> 0
		b := uint8(50 + ratio*155)   // 50 -> 205

		for x := 0; x < width; x++ {
			img.Set(x, y, image.RGBA{r, g, b, 255})
		}
	}

	return img
}
