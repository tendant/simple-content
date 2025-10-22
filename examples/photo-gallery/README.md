# Photo Gallery Example

A complete photo management application demonstrating Simple Content library capabilities.

## Features Demonstrated

- ✅ Upload photos with automatic storage
- ✅ Generate multiple thumbnail sizes (128x128, 256x256, 512x512)
- ✅ Rich metadata management (EXIF-like data)
- ✅ Derived content tracking (thumbnails linked to originals)
- ✅ Content queries and listing
- ✅ Filesystem storage with organized structure

## Running the Example

```bash
cd examples/photo-gallery
go run main.go
```

## Output

```
📸 Simple Content - Photo Gallery Example
==========================================

Step 1: Uploading a photo...
✅ Photo uploaded with ID: 7a8e9f3c-...

Step 2: Generating thumbnails...
  ✓ Created 128x128 thumbnail
  ✓ Created 256x256 thumbnail
  ✓ Created 512x512 thumbnail
✅ Thumbnails generated: 128x128, 256x256, 512x512

Step 3: Adding photo metadata...
✅ Metadata added

Step 4: Retrieving photo details...

📷 Photo Details:
─────────────────────────────────────────
Title: Sunset at the Beach
Description: Beautiful sunset over the Pacific Ocean
Camera: Canon EOS R5
Settings: f/8 @ 1/250, ISO 100
Location: Malibu, California
Date: 2024-10-20
Dimensions: 800x600

Thumbnails: 3 available
  - thumbnail_128 (processed)
  - thumbnail_256 (processed)
  - thumbnail_512 (processed)

Step 5: Listing all photos in gallery...

📁 Gallery contains 1 photo(s):
─────────────────────────────────────────
1. Sunset at the Beach
   ID: 7a8e9f3c-...
   Created: 2024-10-22 12:34:56
   Thumbnails: 3

🎉 Photo gallery demo completed successfully!
📁 Check ./gallery-data/ to see the stored files
```

## File Structure

After running, check `./gallery-data/`:

```
gallery-data/
├── originals/
│   └── objects/
│       └── 7a/
│           └── 8e9f3c..._sunset.jpg
└── derived/
    └── thumbnail/
        ├── thumbnail_128/
        │   └── objects/
        ├── thumbnail_256/
        │   └── objects/
        └── thumbnail_512/
            └── objects/
```

## Key Concepts

### 1. Content Upload
```go
content, err := service.UploadContent(ctx, simplecontent.UploadContentRequest{
    OwnerID:      userID,
    TenantID:     tenantID,
    Name:         "Sunset at the Beach",
    DocumentType: "photo",
    Reader:       photoData,
    FileName:     "sunset.jpg",
    MimeType:     "image/jpeg",
})
```

### 2. Derived Content (Thumbnails)
```go
thumbnail, err := service.UploadDerivedContent(ctx, simplecontent.UploadDerivedContentRequest{
    ParentID:       photoID,
    DerivationType: "thumbnail",
    Variant:        "thumbnail_256",
    Reader:         thumbnailData,
    FileName:       "sunset_thumb.jpg",
})
```

### 3. Rich Metadata
```go
service.SetContentMetadata(ctx, simplecontent.SetContentMetadataRequest{
    ContentID:   photoID,
    Title:       "Sunset at the Beach",
    Description: "Beautiful sunset over the Pacific Ocean",
    Tags:        []string{"sunset", "beach", "nature"},
    CustomMetadata: map[string]interface{}{
        "camera":    "Canon EOS R5",
        "location":  "Malibu, California",
        "aperture":  "f/8",
        // ... any custom fields
    },
})
```

### 4. Query and List
```go
// Get all uploaded photos
photos, err := service.GetContentByStatus(ctx, simplecontent.ContentStatusUploaded)

// Get derived content (thumbnails)
thumbnails, err := service.ListDerivedContent(ctx,
    simplecontent.WithParentID(photoID),
    simplecontent.WithDerivationType("thumbnail"),
)
```

## Extending This Example

### Add More Features

**1. Multiple photo albums:**
```go
// Use tenant ID for albums
album1 := uuid.New() // Album 1
album2 := uuid.New() // Album 2

// Upload to specific album
content, _ := service.UploadContent(ctx, simplecontent.UploadContentRequest{
    TenantID: album1, // This photo belongs to album 1
    // ... other fields
})
```

**2. Photo sharing:**
```go
// Check ownership before allowing access
photo, _ := service.GetContent(ctx, photoID)
if photo.OwnerID != currentUserID {
    return errors.New("access denied")
}
```

**3. Photo search:**
```go
// Search by tags in metadata
allPhotos, _ := service.GetContentByStatus(ctx, simplecontent.ContentStatusUploaded)
for _, photo := range allPhotos {
    metadata, _ := service.GetContentMetadata(ctx, photo.ID)
    tags := metadata.Tags
    // Filter by tags...
}
```

**4. Batch operations:**
```go
// Upload multiple photos concurrently
var wg sync.WaitGroup
for _, file := range photoFiles {
    wg.Add(1)
    go func(f File) {
        defer wg.Done()
        uploadPhoto(ctx, service, f)
    }(file)
}
wg.Wait()
```

## Production Considerations

For a production photo gallery:

1. **Use PostgreSQL + S3**:
   ```go
   cfg, _ := config.LoadServerConfig()
   service, _ := config.BuildService(ctx, cfg)
   ```

2. **Add access control**:
   ```go
   // Check permissions before operations
   if !canAccess(currentUser, photoID) {
       return ErrForbidden
   }
   ```

3. **Implement caching**:
   ```go
   // Cache frequently accessed photos and thumbnails
   ```

4. **Add image processing**:
   ```go
   // Auto-rotate based on EXIF orientation
   // Strip sensitive EXIF data
   // Compress images
   ```

5. **CDN integration**:
   ```go
   // Use URL strategy for CDN URLs
   ```

## Related Examples

- [Document Manager](../document-manager/) - PDF management with previews
- [Video Platform](../video-platform/) - Video transcoding and streaming
- [Basic Usage](../basic/) - Simple getting started examples

## Learn More

- [Quickstart Guide](../../QUICKSTART.md)
- [Full Documentation](../../CLAUDE.md)
- [API Reference](../../pkg/simplecontent/)
