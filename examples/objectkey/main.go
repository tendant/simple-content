package main

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/tendant/simple-content/pkg/simplecontent/objectkey"
)

func main() {
	fmt.Println("Object Key Generation Examples")
	fmt.Println("==============================")

	// Sample UUIDs for demonstration
	contentID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	objectID := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	parentContentID := uuid.MustParse("6ba7b811-9dad-11d1-80b4-00c04fd430c8")

	// 1. Legacy Generator (current flat structure)
	fmt.Println("\n1. Legacy Generator:")
	legacy := objectkey.NewLegacyGenerator()

	key1 := legacy.GenerateKey(contentID, objectID, nil)
	fmt.Printf("   Without filename: %s\n", key1)

	key2 := legacy.GenerateKey(contentID, objectID, &objectkey.KeyMetadata{
		FileName: "document.pdf",
	})
	fmt.Printf("   With filename: %s\n", key2)

	// 2. Git-like Generator (sharded for better performance)
	fmt.Println("\n2. Git-like Generator (Recommended):")
	gitLike := objectkey.NewGitLikeGenerator()

	// Original content
	key3 := gitLike.GenerateKey(contentID, objectID, &objectkey.KeyMetadata{
		FileName:   "report.pdf",
		IsOriginal: true,
	})
	fmt.Printf("   Original: %s\n", key3)

	// Derived content - thumbnail
	key4 := gitLike.GenerateKey(contentID, objectID, &objectkey.KeyMetadata{
		FileName:        "thumb.jpg",
		IsOriginal:      false,
		DerivationType:  "thumbnail",
		Variant:         "256x256",
		ParentContentID: parentContentID,
	})
	fmt.Printf("   Thumbnail: %s\n", key4)

	// Derived content - preview
	key5 := gitLike.GenerateKey(contentID, objectID, &objectkey.KeyMetadata{
		FileName:        "preview.png",
		IsOriginal:      false,
		DerivationType:  "preview",
		Variant:         "1080p",
		ParentContentID: parentContentID,
	})
	fmt.Printf("   Preview: %s\n", key5)

	// 3. Tenant-aware Generator (multi-tenant)
	fmt.Println("\n3. Tenant-aware Generator:")
	tenantAware := objectkey.NewTenantAwareGitLikeGenerator()

	key6 := tenantAware.GenerateKey(contentID, objectID, &objectkey.KeyMetadata{
		FileName:   "contract.pdf",
		TenantID:   "acme-corp",
		IsOriginal: true,
	})
	fmt.Printf("   Tenant: %s\n", key6)

	key7 := tenantAware.GenerateKey(contentID, objectID, &objectkey.KeyMetadata{
		FileName:        "signature.png",
		TenantID:        "acme-corp",
		IsOriginal:      false,
		DerivationType:  "thumbnail",
		Variant:         "small",
		ParentContentID: parentContentID,
	})
	fmt.Printf("   Tenant + Derived: %s\n", key7)

	// 4. High-performance Generator (3-char sharding)
	fmt.Println("\n4. High-performance Generator:")
	highPerf := objectkey.NewHighPerformanceGenerator()

	key8 := highPerf.GenerateKey(contentID, objectID, &objectkey.KeyMetadata{
		FileName:   "video.mp4",
		IsOriginal: true,
	})
	fmt.Printf("   High-perf: %s\n", key8)

	// 5. Custom Generator
	fmt.Println("\n5. Custom Generator:")
	custom := objectkey.NewCustomFuncGenerator(func(contentID, objectID uuid.UUID, metadata *objectkey.KeyMetadata) string {
		if metadata != nil && metadata.DerivationType != "" {
			return fmt.Sprintf("custom/%s/%s/%s_%s.dat",
				metadata.DerivationType,
				contentID.String()[:8],
				objectID.String()[:8],
				metadata.Variant)
		}
		return fmt.Sprintf("custom/original/%s_%s.dat",
			contentID.String()[:8],
			objectID.String()[:8])
	})

	key9 := custom.GenerateKey(contentID, objectID, &objectkey.KeyMetadata{
		IsOriginal: true,
	})
	fmt.Printf("   Custom original: %s\n", key9)

	key10 := custom.GenerateKey(contentID, objectID, &objectkey.KeyMetadata{
		IsOriginal:     false,
		DerivationType: "thumbnail",
		Variant:        "large",
	})
	fmt.Printf("   Custom derived: %s\n", key10)

	fmt.Println("\nBenefits of Git-like sharding:")
	fmt.Println("- Better filesystem performance (limits directory size)")
	fmt.Println("- Clear separation of original vs derived content")
	fmt.Println("- Organized by content type and variant")
	fmt.Println("- Multi-tenant support when needed")
}