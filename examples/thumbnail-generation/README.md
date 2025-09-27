# Thumbnail Generation Example

This example demonstrates how to use the simple-content library's new unified API to upload images and automatically generate thumbnails in multiple sizes using the simplified content operations.

## Features

- **Unified Image Upload**: Upload original images using the new `UploadContent()` operation
- **Automatic Thumbnail Generation**: Create thumbnails in multiple sizes (128px, 256px, 512px)
- **Derived Content**: Use `UploadDerivedContent()` to link thumbnails to their parent
- **Content-Focused API**: Work with content concepts instead of storage objects
- **Storage Backend**: Uses filesystem storage for this example
- **Image Processing**: Built-in image resizing using the Lanczos3 algorithm
- **Simplified Workflow**: Single-call operations replace multi-step object workflows

## Prerequisites

Before running this example, ensure you have:

1. Go 1.24+ installed
2. All dependencies installed (run `go mod tidy` from the project root)

## Running the Example

1. Navigate to the project root:
   ```bash
   cd /path/to/simple-content
   ```

2. Run the thumbnail generation example:
   ```bash
   go run ./examples/thumbnail-generation/main.go
   ```

## What the Example Does

### 1. Setup
- Creates a `ThumbnailService` that wraps the simple-content library
- Sets up filesystem storage in `./data/storage`
- Uses in-memory repository for this demo

### 2. Sample Image Creation
- If no sample image exists, creates a colorful gradient image (`./data/sample_image.jpg`)

### 3. Image Upload with Thumbnails
- Uploads the original image using the unified `UploadContent()` operation
- Automatically generates thumbnails in three sizes: 128px, 256px, and 512px
- Each thumbnail uses `UploadDerivedContent()` for single-call derived content creation
- Storage object details are handled internally by the service

### 4. Content Listing
- Lists all content (original and derived) with their metadata
- Shows the relationship between parent content and thumbnails

### 5. Thumbnail Downloads
- Downloads generated thumbnails to `./data/downloads/`
- Demonstrates how to retrieve processed content

## Output Structure

After running the example, you'll find:

```
./data/
├── sample_image.jpg          # Original sample image (if created)
├── storage/                  # Filesystem storage backend
│   ├── <uuid1>              # Original image object
│   ├── <uuid2>              # 128px thumbnail object
│   ├── <uuid3>              # 256px thumbnail object
│   └── <uuid4>              # 512px thumbnail object
└── downloads/               # Downloaded thumbnails
    ├── thumbnail_128px.jpg
    ├── thumbnail_256px.jpg
    └── thumbnail_512px.jpg
```

## Key Code Components

### ThumbnailService
Wraps the simple-content service with convenience methods for image processing:
- `UploadImageWithThumbnails()` - Complete upload and thumbnail generation workflow using unified API
- `uploadOriginalImage()` - Uses `UploadContent()` for single-call image upload
- `generateThumbnail()` - Uses `UploadDerivedContent()` for single-call thumbnail creation
- `resizeImage()` - Handles actual image processing

### Content Relationships
- Original images are created as standard `Content` entities
- Thumbnails are created as `DerivedContent` with:
  - `DerivationType`: "thumbnail" (user-facing category)
  - `Variant`: "thumbnail_128", "thumbnail_256", etc. (specific size)
  - Metadata tracking processing parameters

### Unified API Pattern
- Original images use the simplified upload workflow:
  1. Single `UploadContent()` call handles content creation, object creation, and data upload
- Thumbnails use the derived content workflow:
  1. Single `UploadDerivedContent()` call handles derived content creation and upload
  2. Automatic parent-child relationship linking
- Object management is handled internally by the service

## Customization

### Different Image Sizes
Modify the `thumbnailSizes` slice in `UploadImageWithThumbnails()`:
```go
thumbnailSizes := []int{64, 128, 256, 512, 1024} // Add more sizes
```

### Storage Backend
Change from filesystem to S3 or memory using the config system:
```go
// For S3 storage
cfg, err := config.Load(
    config.WithStorageBackend("s3", map[string]interface{}{
        "region": "us-west-2",
        "bucket": "my-thumbnails",
        "access_key_id": os.Getenv("AWS_ACCESS_KEY_ID"),
        "secret_access_key": os.Getenv("AWS_SECRET_ACCESS_KEY"),
    }),
)

svc, err := cfg.BuildService()
```

### Image Processing
Replace the `resizeImage()` method to use different libraries or algorithms:
- Add support for WebP, AVIF formats
- Implement smart cropping
- Add image optimization
- Include EXIF data preservation

### Metadata Enhancement
Add more comprehensive metadata tracking:
```go
CustomMetadata: map[string]interface{}{
    "original_width":  originalWidth,
    "original_height": originalHeight,
    "thumbnail_width": thumbnailWidth,
    "thumbnail_height": thumbnailHeight,
    "compression_quality": quality,
    "processing_time": processingDuration,
}
```

## Production Considerations

### Async Processing
For production use, consider implementing asynchronous thumbnail generation:
- Queue thumbnail generation jobs
- Use background workers
- Implement progress tracking
- Handle failures gracefully

### Storage Optimization
- Use appropriate storage backends (S3 for scale)
- Implement CDN integration for fast delivery
- Consider object lifecycle policies

### Error Handling
- Implement retry logic for failed thumbnails
- Add monitoring and alerting
- Handle storage backend failures

### Performance
- Batch process multiple images
- Use connection pooling for database operations
- Implement caching for frequently accessed thumbnails
- Consider using specialized image processing services

## API Migration Notes

This example showcases the new unified API design:

### Old Multi-Step Workflow (Deprecated):
```go
// Old way (3 steps per upload):
content := svc.CreateContent(...)
object := svc.CreateObject(...)
svc.UploadObject(...)
```

### New Unified Workflow (Current):
```go
// New way (1 step per upload):
content, err := svc.UploadContent(ctx, req)

// For derived content:
thumbnail, err := svc.UploadDerivedContent(ctx, derivedReq)
```

The unified API significantly reduces complexity while providing the same functionality through content-focused operations.

This example provides a solid foundation for building production-ready image processing workflows using the simple-content library's new simplified API.