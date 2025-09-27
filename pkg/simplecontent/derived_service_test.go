package simplecontent_test

import (
    "context"
    "fmt"
    "testing"

    "github.com/google/uuid"
    simplecontent "github.com/tendant/simple-content/pkg/simplecontent"
    memoryrepo "github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
)

func TestCreateDerived_InferDerivationTypeFromVariant(t *testing.T) {
    svc := mustService(t)
    ctx := context.Background()

    parent, err := svc.CreateContent(ctx, simplecontent.CreateContentRequest{
        OwnerID: uuid.New(), TenantID: uuid.New(), Name: "parent",
    })
    if err != nil { t.Fatalf("create parent: %v", err) }

    derived, err := svc.CreateDerivedContent(ctx, simplecontent.CreateDerivedContentRequest{
        ParentID: parent.ID,
        OwnerID: parent.OwnerID,
        TenantID: parent.TenantID,
        Variant:  "thumbnail_256", // derivation_type omitted; should infer "thumbnail"
    })
    if err != nil { t.Fatalf("create derived: %v", err) }
    if got := derived.DerivationType; got != "thumbnail" {
        t.Fatalf("expected derivation_type 'thumbnail', got %q", got)
    }
}

func TestListDerivedAndGetRelationship(t *testing.T) {
    svc := mustService(t)
    ctx := context.Background()

    parent, err := svc.CreateContent(ctx, simplecontent.CreateContentRequest{
        OwnerID: uuid.New(), TenantID: uuid.New(), Name: "parent",
    })
    if err != nil { t.Fatalf("create parent: %v", err) }

    // create two variants
    for _, v := range []string{"thumbnail_128","thumbnail_256"} {
        if _, err := svc.CreateDerivedContent(ctx, simplecontent.CreateDerivedContentRequest{
            ParentID: parent.ID,
            OwnerID: parent.OwnerID,
            TenantID: parent.TenantID,
            Variant:  v,
        }); err != nil {
            t.Fatalf("create derived %s: %v", v, err)
        }
    }

    rels, err := svc.ListDerivedContent(ctx, simplecontent.WithParentID(parent.ID))
    if err != nil { t.Fatalf("list derived: %v", err) }
    if len(rels) < 2 { t.Fatalf("expected >=2 derived, got %d", len(rels)) }

    // Check we can resolve one relationship by content id
    rel, err := svc.GetDerivedRelationship(ctx, rels[0].ContentID)
    if err != nil { t.Fatalf("get relationship: %v", err) }
    if rel.ParentID != parent.ID { t.Fatalf("parent mismatch") }
}

// TestBackwardCompatibility_ListDerivedContentParams ensures that existing code
// using ListDerivedContentParams continues to work with no changes after enhancements
func TestBackwardCompatibility_ListDerivedContentParams(t *testing.T) {
    svc := mustService(t)
    ctx := context.Background()

    // Create parent content
    parent, err := svc.CreateContent(ctx, simplecontent.CreateContentRequest{
        OwnerID: uuid.New(), TenantID: uuid.New(), Name: "parent",
    })
    if err != nil { t.Fatalf("create parent: %v", err) }

    // Create derived content using existing API patterns
    variants := []string{"thumbnail_128", "thumbnail_256", "preview_720"}
    for _, variant := range variants {
        _, err := svc.CreateDerivedContent(ctx, simplecontent.CreateDerivedContentRequest{
            ParentID: parent.ID,
            OwnerID: parent.OwnerID,
            TenantID: parent.TenantID,
            Variant:  variant,
        })
        if err != nil { t.Fatalf("create derived %s: %v", variant, err) }
    }

    // Test 1: New ListDerivedContent method with options
    t.Run("Options_ListDerivedContent", func(t *testing.T) {
        results, err := svc.ListDerivedContent(ctx, simplecontent.WithParentID(parent.ID))
        if err != nil { t.Fatalf("ListDerivedContent failed: %v", err) }
        if len(results) != 3 { t.Fatalf("expected 3 results, got %d", len(results)) }

        // Verify existing fields are still populated
        for _, result := range results {
            if result.ParentID != parent.ID { t.Errorf("ParentID mismatch") }
            if result.ContentID == uuid.Nil { t.Errorf("ContentID should be set") }
            if result.DerivationType == "" { t.Errorf("DerivationType should be set") }
            if result.CreatedAt.IsZero() { t.Errorf("CreatedAt should be set") }

            // New URL fields should be empty by default (no URLs requested)
            if result.DownloadURL != "" { t.Errorf("DownloadURL should be empty by default") }
            if result.PreviewURL != "" { t.Errorf("PreviewURL should be empty by default") }
            if result.ThumbnailURL != "" { t.Errorf("ThumbnailURL should be empty by default") }
        }
    })

    // Test 2: Options pattern works equivalently to old parameters
    t.Run("Options_Pattern_Equivalent", func(t *testing.T) {
        // Test options pattern equivalent to old parameters
        results, err := svc.ListDerivedContent(ctx, simplecontent.WithParentID(parent.ID))
        if err != nil { t.Fatalf("ListDerivedContent failed: %v", err) }
        if len(results) != 3 { t.Fatalf("expected 3 results, got %d", len(results)) }
    })

    // Test 3: Options pattern with filtering
    t.Run("Options_Pattern_Filtering", func(t *testing.T) {
        // Test equivalent filtering with options pattern
        results, err := svc.ListDerivedContent(ctx,
            simplecontent.WithParentID(parent.ID),
            simplecontent.WithDerivationType("thumbnail"),
            simplecontent.WithLimit(10),
            simplecontent.WithOffset(0),
        )
        if err != nil { t.Fatalf("filtering failed: %v", err) }

        // Should get 2 thumbnail results (128 and 256)
        if len(results) != 2 { t.Fatalf("expected 2 thumbnail results, got %d", len(results)) }

        for _, result := range results {
            if result.DerivationType != "thumbnail" {
                t.Errorf("expected thumbnail, got %s", result.DerivationType)
            }
        }
    })
}

// TestBackwardCompatibility_DerivedContentStruct ensures that DerivedContent struct
// remains backward compatible with existing code
func TestBackwardCompatibility_DerivedContentStruct(t *testing.T) {
    svc := mustService(t)
    ctx := context.Background()

    // Create parent and derived content
    parent, err := svc.CreateContent(ctx, simplecontent.CreateContentRequest{
        OwnerID: uuid.New(), TenantID: uuid.New(), Name: "parent",
    })
    if err != nil { t.Fatalf("create parent: %v", err) }

    derivedContent, err := svc.CreateDerivedContent(ctx, simplecontent.CreateDerivedContentRequest{
        ParentID: parent.ID,
        OwnerID: parent.OwnerID,
        TenantID: parent.TenantID,
        Variant:  "thumbnail_256",
        Metadata: map[string]interface{}{
            "width": 256,
            "height": 256,
        },
    })
    if err != nil { t.Fatalf("create derived: %v", err) }

    // Get the derived relationship to test DerivedContent fields
    derived, err := svc.GetDerivedRelationship(ctx, derivedContent.ID)
    if err != nil { t.Fatalf("get derived relationship: %v", err) }

    // Test that all existing fields are accessible and work as before
    t.Run("Existing_Fields_Accessible", func(t *testing.T) {
        // These field accesses should compile and work exactly as before
        _ = derived.ParentID
        _ = derived.ContentID
        _ = derived.DerivationType
        _ = derived.DerivationParams
        _ = derived.ProcessingMetadata
        _ = derived.CreatedAt
        _ = derived.UpdatedAt
        _ = derived.DocumentType
        _ = derived.Status

        // Verify values are set correctly
        if derived.ParentID != parent.ID { t.Errorf("ParentID mismatch") }
        if derived.DerivationType != "thumbnail" { t.Errorf("DerivationType should be 'thumbnail'") }
        if derived.Variant != "thumbnail_256" { t.Errorf("Variant should be 'thumbnail_256'") }
    })

    // Test that new fields have appropriate zero values
    t.Run("New_Fields_Zero_Values", func(t *testing.T) {
        // New URL fields should be empty strings by default
        if derived.DownloadURL != "" { t.Errorf("DownloadURL should be empty by default") }
        if derived.PreviewURL != "" { t.Errorf("PreviewURL should be empty by default") }
        if derived.ThumbnailURL != "" { t.Errorf("ThumbnailURL should be empty by default") }

        // New optional enhancement fields should be nil/empty by default
        if derived.Objects != nil { t.Errorf("Objects should be nil by default") }
        if derived.Metadata != nil { t.Errorf("Metadata should be nil by default") }
        if derived.ParentContent != nil { t.Errorf("ParentContent should be nil by default") }
    })
}

// TestBackwardCompatibility_ServiceInterface ensures that existing service interface
// methods continue to work without any changes
func TestBackwardCompatibility_ServiceInterface(t *testing.T) {
    svc := mustService(t)
    ctx := context.Background()

    // Create test data
    parent, err := svc.CreateContent(ctx, simplecontent.CreateContentRequest{
        OwnerID: uuid.New(), TenantID: uuid.New(), Name: "parent",
    })
    if err != nil { t.Fatalf("create parent: %v", err) }

    derived, err := svc.CreateDerivedContent(ctx, simplecontent.CreateDerivedContentRequest{
        ParentID: parent.ID,
        OwnerID: parent.OwnerID,
        TenantID: parent.TenantID,
        Variant:  "thumbnail_256",
    })
    if err != nil { t.Fatalf("create derived: %v", err) }

    // Test 1: Existing service methods should work unchanged
    t.Run("Existing_Service_Methods", func(t *testing.T) {
        // These methods should compile and work exactly as before
        results, err := svc.ListDerivedContent(ctx, simplecontent.WithParentID(parent.ID))
        if err != nil { t.Fatalf("ListDerivedContent failed: %v", err) }
        if len(results) != 1 { t.Fatalf("expected 1 result, got %d", len(results)) }

        relationship, err := svc.GetDerivedRelationship(ctx, derived.ID)
        if err != nil { t.Fatalf("GetDerivedRelationship failed: %v", err) }
        if relationship.ParentID != parent.ID { t.Errorf("ParentID mismatch") }
    })

    // Test 2: Method signatures for simplified API
    t.Run("Method_Signatures_Simplified", func(t *testing.T) {
        // This test ensures the simplified API signatures work correctly
        var _ func(context.Context, uuid.UUID) (*simplecontent.DerivedContent, error) = svc.GetDerivedRelationship

        // Single method with options pattern
        var _ func(context.Context, ...simplecontent.ListDerivedContentOption) ([]*simplecontent.DerivedContent, error) = svc.ListDerivedContent
    })
}

// TestBackwardCompatibility_CreateDerivedContentRequest ensures that existing
// creation patterns continue to work
func TestBackwardCompatibility_CreateDerivedContentRequest(t *testing.T) {
    svc := mustService(t)
    ctx := context.Background()

    parent, err := svc.CreateContent(ctx, simplecontent.CreateContentRequest{
        OwnerID: uuid.New(), TenantID: uuid.New(), Name: "parent",
    })
    if err != nil { t.Fatalf("create parent: %v", err) }

    // Test 1: Existing creation patterns should work unchanged
    t.Run("Existing_Creation_Patterns", func(t *testing.T) {
        // Pattern 1: Basic creation with just variant
        derived1, err := svc.CreateDerivedContent(ctx, simplecontent.CreateDerivedContentRequest{
            ParentID: parent.ID,
            OwnerID: parent.OwnerID,
            TenantID: parent.TenantID,
            Variant:  "thumbnail_128",
        })
        if err != nil { t.Fatalf("basic creation failed: %v", err) }
        if derived1.DerivationType != "thumbnail" { t.Errorf("DerivationType inference failed") }

        // Pattern 2: Creation with explicit derivation type and variant
        derived2, err := svc.CreateDerivedContent(ctx, simplecontent.CreateDerivedContentRequest{
            ParentID:       parent.ID,
            OwnerID:        parent.OwnerID,
            TenantID:       parent.TenantID,
            DerivationType: "preview",
            Variant:        "preview_720",
        })
        if err != nil { t.Fatalf("explicit creation failed: %v", err) }
        if derived2.DerivationType != "preview" { t.Errorf("DerivationType should be 'preview'") }

        // Pattern 3: Creation with metadata (existing behavior)
        derived3, err := svc.CreateDerivedContent(ctx, simplecontent.CreateDerivedContentRequest{
            ParentID: parent.ID,
            OwnerID:  parent.OwnerID,
            TenantID: parent.TenantID,
            Variant:  "thumbnail_256",
            Metadata: map[string]interface{}{
                "width":  256,
                "height": 256,
                "quality": 85,
            },
        })
        if err != nil { t.Fatalf("creation with metadata failed: %v", err) }
        if derived3.DerivationType != "thumbnail" { t.Errorf("DerivationType should be 'thumbnail'") }
    })
}

// TestBackwardCompatibility_DataConsistency ensures that data created before
// the enhancement can still be queried and filtered correctly
func TestBackwardCompatibility_DataConsistency(t *testing.T) {
    svc := mustService(t)
    ctx := context.Background()

    parent, err := svc.CreateContent(ctx, simplecontent.CreateContentRequest{
        OwnerID: uuid.New(), TenantID: uuid.New(), Name: "parent",
    })
    if err != nil { t.Fatalf("create parent: %v", err) }

    // Simulate "legacy" data created before enhancement (no explicit Variant field)
    // This would represent data created by older versions of the system
    t.Run("Legacy_Data_Compatibility", func(t *testing.T) {
        // Create derived content that might have been created before enhancement
        _, err := svc.CreateDerivedContent(ctx, simplecontent.CreateDerivedContentRequest{
            ParentID:       parent.ID,
            OwnerID:        parent.OwnerID,
            TenantID:       parent.TenantID,
            DerivationType: "thumbnail",
            // Note: No explicit Variant - simulating legacy data
            Metadata: map[string]interface{}{
                "width": 256,
                "height": 256,
            },
        })
        if err != nil { t.Fatalf("legacy creation failed: %v", err) }

        // Should still be findable via new options method
        results, err := svc.ListDerivedContent(ctx, simplecontent.WithParentID(parent.ID))
        if err != nil { t.Fatalf("listing legacy data failed: %v", err) }
        if len(results) == 0 { t.Fatalf("legacy data not found") }

        // Should work with enhanced filtering too
        filtered, err := svc.ListDerivedContent(ctx,
            simplecontent.WithParentID(parent.ID),
            simplecontent.WithDerivationType("thumbnail"),
        )
        if err != nil { t.Fatalf("filtering legacy data failed: %v", err) }
        if len(filtered) == 0 { t.Fatalf("legacy data not found in enhanced filtering") }
    })
}

func mustService(t *testing.T) simplecontent.Service {
    t.Helper()
    repo := memoryrepo.New()
    svc, err := simplecontent.New(simplecontent.WithRepository(repo))
    if err != nil { t.Fatalf("service new: %v", err) }
    return svc
}

// Helper functions for backward compatibility tests
func intPtr(i int) *int {
    return &i
}

func stringPtr(s string) *string {
    return &s
}

// Tests for the new option pattern

func TestListDerivedContent_BasicFiltering(t *testing.T) {
    svc := mustService(t)
    ctx := context.Background()

    parent, err := svc.CreateContent(ctx, simplecontent.CreateContentRequest{
        OwnerID: uuid.New(), TenantID: uuid.New(), Name: "parent",
    })
    if err != nil { t.Fatalf("create parent: %v", err) }

    // Create some derived content
    variants := []string{"thumbnail_128", "thumbnail_256", "preview_720"}
    for _, variant := range variants {
        _, err := svc.CreateDerivedContent(ctx, simplecontent.CreateDerivedContentRequest{
            ParentID: parent.ID,
            OwnerID: parent.OwnerID,
            TenantID: parent.TenantID,
            Variant:  variant,
        })
        if err != nil { t.Fatalf("create derived %s: %v", variant, err) }
    }

    // Test option pattern - get all thumbnails
    results, err := svc.ListDerivedContent(ctx,
        simplecontent.WithParentID(parent.ID),
        simplecontent.WithDerivationType("thumbnail"),
    )
    if err != nil { t.Fatalf("list with options: %v", err) }
    if len(results) != 2 { t.Fatalf("expected 2 thumbnails, got %d", len(results)) }

    // Test specific variant
    results, err = svc.ListDerivedContent(ctx,
        simplecontent.WithParentID(parent.ID),
        simplecontent.WithVariant("thumbnail_256"),
    )
    if err != nil { t.Fatalf("list with variant option: %v", err) }
    if len(results) != 1 { t.Fatalf("expected 1 result, got %d", len(results)) }
    if results[0].Variant != "thumbnail_256" { t.Fatalf("expected variant thumbnail_256, got %s", results[0].Variant) }
}

func TestListDerivedContent_MultipleVariants(t *testing.T) {
    svc := mustService(t)
    ctx := context.Background()

    parent, err := svc.CreateContent(ctx, simplecontent.CreateContentRequest{
        OwnerID: uuid.New(), TenantID: uuid.New(), Name: "parent",
    })
    if err != nil { t.Fatalf("create parent: %v", err) }

    // Create various derived content
    variants := []string{"thumbnail_128", "thumbnail_256", "preview_720", "preview_1080"}
    for _, variant := range variants {
        _, err := svc.CreateDerivedContent(ctx, simplecontent.CreateDerivedContentRequest{
            ParentID: parent.ID,
            OwnerID: parent.OwnerID,
            TenantID: parent.TenantID,
            Variant:  variant,
        })
        if err != nil { t.Fatalf("create derived %s: %v", variant, err) }
    }

    // Test multiple variants
    results, err := svc.ListDerivedContent(ctx,
        simplecontent.WithParentID(parent.ID),
        simplecontent.WithVariants("thumbnail_256", "preview_1080"),
    )
    if err != nil { t.Fatalf("list with variants option: %v", err) }
    if len(results) != 2 { t.Fatalf("expected 2 results, got %d", len(results)) }

    // Verify we got the right variants
    variants_found := make(map[string]bool)
    for _, result := range results {
        variants_found[result.Variant] = true
    }
    if !variants_found["thumbnail_256"] || !variants_found["preview_1080"] {
        t.Fatalf("didn't get expected variants")
    }
}

func TestListDerivedContent_WithURLsAndMetadata(t *testing.T) {
    svc := mustService(t)
    ctx := context.Background()

    parent, err := svc.CreateContent(ctx, simplecontent.CreateContentRequest{
        OwnerID: uuid.New(), TenantID: uuid.New(), Name: "parent",
    })
    if err != nil { t.Fatalf("create parent: %v", err) }

    _, err = svc.CreateDerivedContent(ctx, simplecontent.CreateDerivedContentRequest{
        ParentID: parent.ID,
        OwnerID: parent.OwnerID,
        TenantID: parent.TenantID,
        Variant:  "thumbnail_256",
    })
    if err != nil { t.Fatalf("create derived: %v", err) }

    // Test with URLs option
    results, err := svc.ListDerivedContent(ctx,
        simplecontent.WithParentID(parent.ID),
        simplecontent.WithURLs(),
    )
    if err != nil { t.Fatalf("list with URLs option: %v", err) }
    if len(results) != 1 { t.Fatalf("expected 1 result, got %d", len(results)) }

    // Note: URLs would be populated if storage backend supports it
    // For memory backend, they'll be empty but the option should work
}

func TestListDerivedContent_Pagination(t *testing.T) {
    svc := mustService(t)
    ctx := context.Background()

    parent, err := svc.CreateContent(ctx, simplecontent.CreateContentRequest{
        OwnerID: uuid.New(), TenantID: uuid.New(), Name: "parent",
    })
    if err != nil { t.Fatalf("create parent: %v", err) }

    // Create multiple derived content
    for i := 0; i < 5; i++ {
        _, err := svc.CreateDerivedContent(ctx, simplecontent.CreateDerivedContentRequest{
            ParentID: parent.ID,
            OwnerID: parent.OwnerID,
            TenantID: parent.TenantID,
            Variant:  fmt.Sprintf("thumbnail_%d", 128+i*32),
        })
        if err != nil { t.Fatalf("create derived %d: %v", i, err) }
    }

    // Test pagination
    results, err := svc.ListDerivedContent(ctx,
        simplecontent.WithParentID(parent.ID),
        simplecontent.WithLimit(2),
        simplecontent.WithOffset(1),
    )
    if err != nil { t.Fatalf("list with pagination: %v", err) }
    if len(results) != 2 { t.Fatalf("expected 2 results with limit, got %d", len(results)) }

    // Test combined pagination option
    results, err = svc.ListDerivedContent(ctx,
        simplecontent.WithParentID(parent.ID),
        simplecontent.WithPagination(3, 2),
    )
    if err != nil { t.Fatalf("list with pagination option: %v", err) }
    if len(results) != 3 { t.Fatalf("expected 3 results with combined pagination, got %d", len(results)) }
}

func TestListDerivedContent_EmptyOptions(t *testing.T) {
    svc := mustService(t)
    ctx := context.Background()

    // Test with no options - should return empty list since no parent specified
    results, err := svc.ListDerivedContent(ctx)
    if err != nil { t.Fatalf("list with no options: %v", err) }
    // Should return empty or all derived content depending on implementation
    // The key is that it shouldn't crash
    _ = results
}

func TestOptionPatternVsLegacyConvenienceFunctions(t *testing.T) {
    svc := mustService(t)
    ctx := context.Background()

    parent, err := svc.CreateContent(ctx, simplecontent.CreateContentRequest{
        OwnerID: uuid.New(), TenantID: uuid.New(), Name: "parent",
    })
    if err != nil { t.Fatalf("create parent: %v", err) }

    // Create thumbnails
    sizes := []string{"128", "256", "512"}
    for _, size := range sizes {
        _, err := svc.CreateDerivedContent(ctx, simplecontent.CreateDerivedContentRequest{
            ParentID: parent.ID,
            OwnerID: parent.OwnerID,
            TenantID: parent.TenantID,
            Variant:  "thumbnail_" + size,
        })
        if err != nil { t.Fatalf("create thumbnail %s: %v", size, err) }
    }

    // Test option pattern replacing GetThumbnailsBySize
    results, err := svc.ListDerivedContent(ctx,
        simplecontent.WithParentID(parent.ID),
        simplecontent.WithDerivationType("thumbnail"),
        simplecontent.WithVariants("thumbnail_256", "thumbnail_512"),
        simplecontent.WithURLs(),
    )
    if err != nil { t.Fatalf("get thumbnails with options: %v", err) }
    if len(results) != 2 { t.Fatalf("expected 2 thumbnails, got %d", len(results)) }

    // Test option pattern replacing ListDerivedByTypeAndVariant
    results, err = svc.ListDerivedContent(ctx,
        simplecontent.WithParentID(parent.ID),
        simplecontent.WithDerivationType("thumbnail"),
        simplecontent.WithVariant("thumbnail_256"),
    )
    if err != nil { t.Fatalf("list by type and variant with options: %v", err) }
    if len(results) != 1 { t.Fatalf("expected 1 result, got %d", len(results)) }
    if results[0].Variant != "thumbnail_256" { t.Fatalf("expected variant thumbnail_256, got %s", results[0].Variant) }

    // Test legacy convenience function still works
    legacyResults, err := simplecontent.GetThumbnailsBySize(ctx, svc, parent.ID, []string{"256", "512"})
    if err != nil { t.Fatalf("legacy get thumbnails: %v", err) }
    if len(legacyResults) != 2 { t.Fatalf("expected 2 thumbnails from legacy, got %d", len(legacyResults)) }
}
