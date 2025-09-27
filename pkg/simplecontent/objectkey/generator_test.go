package objectkey

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestLegacyGenerator(t *testing.T) {
	gen := NewLegacyGenerator()
	contentID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	objectID := uuid.MustParse("987fcdeb-51a2-43d1-9f12-345678901234")

	tests := []struct {
		name     string
		metadata *KeyMetadata
		expected string
	}{
		{
			name:     "without filename",
			metadata: nil,
			expected: "C/123e4567-e89b-12d3-a456-426614174000/987fcdeb-51a2-43d1-9f12-345678901234",
		},
		{
			name: "with filename",
			metadata: &KeyMetadata{
				FileName: "document.pdf",
			},
			expected: "C/123e4567-e89b-12d3-a456-426614174000/987fcdeb-51a2-43d1-9f12-345678901234/document.pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.GenerateKey(contentID, objectID, tt.metadata)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGitLikeGenerator(t *testing.T) {
	gen := NewGitLikeGenerator()
	contentID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	objectID := uuid.MustParse("987fcdeb-51a2-43d1-9f12-345678901234")

	tests := []struct {
		name     string
		metadata *KeyMetadata
		contains []string // Check if result contains these parts
	}{
		{
			name: "original content without filename",
			metadata: &KeyMetadata{
				IsOriginal: true,
			},
			contains: []string{"originals/objects/98/"},
		},
		{
			name: "original content with filename",
			metadata: &KeyMetadata{
				IsOriginal: true,
				FileName:   "document.pdf",
			},
			contains: []string{"originals/objects/98/", "_document.pdf"},
		},
		{
			name: "derived content - thumbnail",
			metadata: &KeyMetadata{
				IsOriginal:     false,
				DerivationType: "thumbnail",
				Variant:        "256x256",
				FileName:       "thumb.jpg",
			},
			contains: []string{"derived/thumbnail/256x256/objects/98/", "_thumb.jpg"},
		},
		{
			name: "derived content without variant",
			metadata: &KeyMetadata{
				IsOriginal:     false,
				DerivationType: "preview",
				FileName:       "preview.png",
			},
			contains: []string{"derived/preview/default/objects/98/", "_preview.png"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.GenerateKey(contentID, objectID, tt.metadata)
			for _, part := range tt.contains {
				if !strings.Contains(result, part) {
					t.Errorf("expected result to contain %s, got %s", part, result)
				}
			}
			// Verify sharding - should start with 2-char directory
			if tt.metadata != nil && tt.metadata.DerivationType != "" {
				parts := strings.Split(result, "/")
				if len(parts) < 5 {
					t.Errorf("expected at least 5 path parts for derived content, got %d", len(parts))
				}
			}
		})
	}
}

func TestTenantAwareGitLikeGenerator(t *testing.T) {
	gen := NewTenantAwareGitLikeGenerator()
	contentID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	objectID := uuid.MustParse("987fcdeb-51a2-43d1-9f12-345678901234")

	tests := []struct {
		name     string
		metadata *KeyMetadata
		contains []string
	}{
		{
			name: "default tenant original",
			metadata: &KeyMetadata{
				IsOriginal: true,
				FileName:   "doc.pdf",
			},
			contains: []string{"tenants/default/originals/objects/98/"},
		},
		{
			name: "custom tenant derived",
			metadata: &KeyMetadata{
				TenantID:       "acme-corp",
				IsOriginal:     false,
				DerivationType: "thumbnail",
				Variant:        "small",
				FileName:       "thumb.jpg",
			},
			contains: []string{"tenants/acme-corp/derived/thumbnail/small/objects/98/"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.GenerateKey(contentID, objectID, tt.metadata)
			for _, part := range tt.contains {
				if !strings.Contains(result, part) {
					t.Errorf("expected result to contain %s, got %s", part, result)
				}
			}
		})
	}
}

func TestHashedGitLikeGenerator(t *testing.T) {
	gen := NewHashedGitLikeGenerator()
	contentID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	objectID := uuid.MustParse("987fcdeb-51a2-43d1-9f12-345678901234")

	// Test deterministic behavior
	metadata := &KeyMetadata{
		IsOriginal: true,
		FileName:   "test.txt",
	}

	result1 := gen.GenerateKey(contentID, objectID, metadata)
	result2 := gen.GenerateKey(contentID, objectID, metadata)

	if result1 != result2 {
		t.Errorf("hashed generator should be deterministic, got different results: %s vs %s", result1, result2)
	}

	// Should contain originals path and be properly sharded
	if !strings.Contains(result1, "originals/objects/") {
		t.Errorf("expected result to contain originals/objects/, got %s", result1)
	}
}

func TestCustomFuncGenerator(t *testing.T) {
	customFunc := func(contentID, objectID uuid.UUID, metadata *KeyMetadata) string {
		return "custom/" + objectID.String() + ".dat"
	}

	gen := NewCustomFuncGenerator(customFunc)
	contentID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	objectID := uuid.MustParse("987fcdeb-51a2-43d1-9f12-345678901234")

	result := gen.GenerateKey(contentID, objectID, nil)
	expected := "custom/987fcdeb-51a2-43d1-9f12-345678901234.dat"

	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestSanitization(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal.txt", "normal.txt"},
		{"file with spaces.txt", "file_with_spaces.txt"},
		{"file/with/slashes.txt", "file_with_slashes.txt"},
		{"file:with:colons.txt", "file_with_colons.txt"},
		{"file*with?special<chars>.txt", "file_with_special_chars_.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestShardingDistribution(t *testing.T) {
	gen := NewGitLikeGenerator()
	contentID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	// Generate keys for multiple objects and check distribution
	shardCounts := make(map[string]int)

	for i := 0; i < 1000; i++ {
		objectID := uuid.New()
		metadata := &KeyMetadata{IsOriginal: true}
		key := gen.GenerateKey(contentID, objectID, metadata)

		// Extract shard directory (should be after "originals/objects/")
		parts := strings.Split(key, "/")
		if len(parts) >= 3 {
			shard := parts[2] // originals/objects/{shard}/...
			shardCounts[shard]++
		}
	}

	// Should have reasonable distribution (not all in one shard)
	if len(shardCounts) < 10 {
		t.Errorf("expected more diverse sharding, got only %d shards", len(shardCounts))
	}

	// No single shard should dominate too much
	for shard, count := range shardCounts {
		if count > 200 { // 20% of 1000
			t.Errorf("shard %s has too many objects (%d), sharding may be poor", shard, count)
		}
	}
}

// Benchmark different generators
func BenchmarkGenerators(b *testing.B) {
	contentID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	objectID := uuid.MustParse("987fcdeb-51a2-43d1-9f12-345678901234")
	metadata := &KeyMetadata{
		IsOriginal: true,
		FileName:   "benchmark.txt",
	}

	generators := map[string]Generator{
		"Legacy":           NewLegacyGenerator(),
		"GitLike":          NewGitLikeGenerator(),
		"TenantAware":      NewTenantAwareGitLikeGenerator(),
		"HashedGitLike":    NewHashedGitLikeGenerator(),
	}

	for name, gen := range generators {
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = gen.GenerateKey(contentID, objectID, metadata)
			}
		})
	}
}