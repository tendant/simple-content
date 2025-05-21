package integration

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendant/simple-content/tests/testutil"
)

func TestDerivedContentWorkflow(t *testing.T) {
	// Setup test server
	server := testutil.SetupTestServer()
	defer server.Close()

	// 1. Create original content
	original := testutil.CreateContent(t, server.URL)
	assert.Equal(t, "original", original.DerivationType)
	assert.Equal(t, 0, original.DerivationLevel)

	// 2. Set metadata for original content
	originalMetadata := map[string]interface{}{
		"content_type": "video/mp4",
		"title":        "Original Video",
		"description":  "An original video content",
		"tags":         []string{"video", "original"},
		"file_size":    15728640,
		"created_by":   "User 1",
		"metadata": map[string]interface{}{
			"duration":   "00:05:30",
			"resolution": "1920x1080",
		},
	}
	testutil.SetContentMetadata(t, server.URL, original.ID, originalMetadata)

	// 3. Create derived content (level 1)
	derived1 := testutil.CreateDerivedContent(t, server.URL, original.ID)
	assert.Equal(t, original.ID, derived1.ParentID)
	assert.Equal(t, "derived", derived1.DerivationType)
	assert.Equal(t, 1, derived1.DerivationLevel)

	// 4. Set metadata for derived content
	derived1Metadata := map[string]interface{}{
		"content_type": "image/jpeg",
		"title":        "Thumbnail",
		"description":  "A thumbnail extracted from the original video",
		"tags":         []string{"image", "thumbnail", "derived"},
		"file_size":    524288,
		"created_by":   "System",
		"metadata": map[string]interface{}{
			"width":            1280,
			"height":           720,
			"source_timestamp": "00:01:15",
		},
	}
	testutil.SetContentMetadata(t, server.URL, derived1.ID, derived1Metadata)

	// 5. Create second-level derived content
	derived2 := testutil.CreateDerivedContent(t, server.URL, derived1.ID)
	assert.Equal(t, derived1.ID, derived2.ParentID)
	assert.Equal(t, "derived", derived2.DerivationType)
	assert.Equal(t, 2, derived2.DerivationLevel)

	// 6. Set metadata for second-level derived content
	derived2Metadata := map[string]interface{}{
		"content_type": "text/plain",
		"title":        "Image Description",
		"description":  "A text description of the thumbnail",
		"tags":         []string{"text", "description", "derived"},
		"file_size":    1024,
		"created_by":   "System",
		"metadata": map[string]interface{}{
			"language":   "en",
			"word_count": 150,
		},
	}
	testutil.SetContentMetadata(t, server.URL, derived2.ID, derived2Metadata)

	// 7. Get direct derived content
	directDerived := testutil.GetDerivedContent(t, server.URL, original.ID)
	assert.Len(t, directDerived, 1)
	assert.Equal(t, derived1.ID, directDerived[0].ID)

	// 8. Get derived content tree
	tree := testutil.GetDerivedContentTree(t, server.URL, original.ID)
	assert.Len(t, tree, 3) // original + derived1 + derived2

	// 9. Verify metadata independence
	originalMetadataRetrieved := testutil.GetContentMetadata(t, server.URL, original.ID)
	derived1MetadataRetrieved := testutil.GetContentMetadata(t, server.URL, derived1.ID)
	derived2MetadataRetrieved := testutil.GetContentMetadata(t, server.URL, derived2.ID)

	assert.Equal(t, "video/mp4", originalMetadataRetrieved.ContentType)
	assert.Equal(t, "Original Video", originalMetadataRetrieved.Title)
	assert.Equal(t, "00:05:30", originalMetadataRetrieved.Metadata["duration"])

	assert.Equal(t, "image/jpeg", derived1MetadataRetrieved.ContentType)
	assert.Equal(t, "Thumbnail", derived1MetadataRetrieved.Title)

	// Check width value, handling both int and float64 cases
	width := derived1MetadataRetrieved.Metadata["width"]
	var widthValue int
	switch v := width.(type) {
	case int:
		widthValue = v
	case float64:
		widthValue = int(v)
	default:
		t.Fatalf("unexpected type for width: %T", width)
	}
	assert.Equal(t, 1280, widthValue)

	assert.Equal(t, "text/plain", derived2MetadataRetrieved.ContentType)
	assert.Equal(t, "Image Description", derived2MetadataRetrieved.Title)

	// Check word_count value, handling both int and float64 cases
	wordCount := derived2MetadataRetrieved.Metadata["word_count"]
	var wordCountValue int
	switch v := wordCount.(type) {
	case int:
		wordCountValue = v
	case float64:
		wordCountValue = int(v)
	default:
		t.Fatalf("unexpected type for word_count: %T", wordCount)
	}
	assert.Equal(t, 150, wordCountValue)
}

func TestDerivedContentMaxDepth(t *testing.T) {
	// Setup test server
	server := testutil.SetupTestServer()
	defer server.Close()

	// Create original content (level 0)
	content := testutil.CreateContent(t, server.URL)
	assert.Equal(t, 0, content.DerivationLevel)

	// Create derived content levels 1-5
	for i := 1; i <= 5; i++ {
		derived := testutil.CreateDerivedContent(t, server.URL, content.ID)
		assert.Equal(t, i, derived.DerivationLevel)
		content = derived // Use this as parent for next level
	}

	// Attempt to create level 6 (should fail)
	resp := testutil.AttemptCreateDerivedContent(t, server.URL, content.ID)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	// Check error message
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Contains(t, string(body), "maximum derivation depth")
}
