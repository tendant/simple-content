package presets

import (
	"github.com/tendant/simple-content/pkg/simplecontent"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDevelopment(t *testing.T) {
	t.Run("default configuration", func(t *testing.T) {
		svc, cleanup, err := NewDevelopment()
		require.NoError(t, err)
		require.NotNil(t, svc)
		require.NotNil(t, cleanup)

		// Verify service works
		ctx := context.Background()
		content, err := svc.UploadContent(ctx, simplecontent.UploadContentRequest{
			OwnerID:      uuid.New(),
			TenantID:     uuid.New(),
			Name:         "test.txt",
			DocumentType: "text/plain",
			Reader:       strings.NewReader("Hello Development!"),
			FileName:     "test.txt",
		})
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, content.ID)

		// Cleanup
		cleanup()

		// Verify storage directory was removed
		_, err = os.Stat("./dev-data")
		assert.True(t, os.IsNotExist(err), "dev-data should be removed after cleanup")
	})

	t.Run("custom storage directory", func(t *testing.T) {
		customDir := "./custom-dev-data"
		svc, cleanup, err := NewDevelopment(WithDevStorage(customDir))
		require.NoError(t, err)
		require.NotNil(t, svc)
		defer cleanup()

		// Upload content
		ctx := context.Background()
		_, err = svc.UploadContent(ctx, simplecontent.UploadContentRequest{
			OwnerID:      uuid.New(),
			TenantID:     uuid.New(),
			Name:         "test.txt",
			DocumentType: "text/plain",
			Reader:       strings.NewReader("Custom directory!"),
			FileName:     "test.txt",
		})
		require.NoError(t, err)

		// Verify custom directory exists
		_, err = os.Stat(customDir)
		assert.NoError(t, err, "custom storage directory should exist")
	})

}

func TestNewTesting(t *testing.T) {
	t.Run("default configuration", func(t *testing.T) {
		svc := NewTesting(t)
		require.NotNil(t, svc)

		// Verify service works
		ctx := context.Background()
		content, err := svc.UploadContent(ctx, simplecontent.UploadContentRequest{
			OwnerID:      uuid.New(),
			TenantID:     uuid.New(),
			Name:         "test.txt",
			DocumentType: "text/plain",
			Reader:       strings.NewReader("Hello Testing!"),
			FileName:     "test.txt",
		})
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, content.ID)

		// Cleanup happens automatically via t.Cleanup()
	})

	t.Run("parallel execution", func(t *testing.T) {
		// Test that multiple tests can run in parallel
		t.Run("test1", func(t *testing.T) {
			t.Parallel()
			svc := NewTesting(t)
			ctx := context.Background()
			_, err := svc.UploadContent(ctx, simplecontent.UploadContentRequest{
				OwnerID:      uuid.New(),
				TenantID:     uuid.New(),
				Name:         "test1.txt",
				DocumentType: "text/plain",
				Reader:       strings.NewReader("Test 1"),
				FileName:     "test1.txt",
			})
			require.NoError(t, err)
		})

		t.Run("test2", func(t *testing.T) {
			t.Parallel()
			svc := NewTesting(t)
			ctx := context.Background()
			_, err := svc.UploadContent(ctx, simplecontent.UploadContentRequest{
				OwnerID:      uuid.New(),
				TenantID:     uuid.New(),
				Name:         "test2.txt",
				DocumentType: "text/plain",
				Reader:       strings.NewReader("Test 2"),
				FileName:     "test2.txt",
			})
			require.NoError(t, err)
		})
	})
}

func TestTestService(t *testing.T) {
	t.Run("convenience function", func(t *testing.T) {
		svc := TestService(t)
		require.NotNil(t, svc)

		// Verify service works
		ctx := context.Background()
		content, err := svc.UploadContent(ctx, simplecontent.UploadContentRequest{
			OwnerID:      uuid.New(),
			TenantID:     uuid.New(),
			Name:         "test.txt",
			DocumentType: "text/plain",
			Reader:       strings.NewReader("Hello!"),
			FileName:     "test.txt",
		})
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, content.ID)
	})
}

func TestNewProduction(t *testing.T) {
	t.Run("validation - requires postgres", func(t *testing.T) {
		// Set memory database (should fail)
		_, err := NewProduction(WithProdDatabase("memory", ""))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "requires DATABASE_TYPE=postgres")
	})

	t.Run("validation - requires database URL", func(t *testing.T) {
		// No database URL (should fail)
		_, err := NewProduction(WithProdDatabase("postgres", ""))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DATABASE_URL is required")
	})

	t.Run("validation - requires persistent storage", func(t *testing.T) {
		// Memory storage in production (should fail)
		_, err := NewProduction(
			WithProdDatabase("postgres", "postgresql://localhost/test"),
			WithProdStorage("memory"),
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "requires persistent storage")
	})

	// Note: Full production test would require actual Postgres/S3
	// so we only test validation here
}

func TestDevelopmentCleanup(t *testing.T) {
	t.Run("cleanup removes storage directory", func(t *testing.T) {
		storageDir := "./test-dev-data"
		svc, cleanup, err := NewDevelopment(WithDevStorage(storageDir))
		require.NoError(t, err)
		require.NotNil(t, svc)

		// Upload content to create files
		ctx := context.Background()
		_, err = svc.UploadContent(ctx, simplecontent.UploadContentRequest{
			OwnerID:      uuid.New(),
			TenantID:     uuid.New(),
			Name:         "test.txt",
			DocumentType: "text/plain",
			Reader:       strings.NewReader("Test content"),
			FileName:     "test.txt",
		})
		require.NoError(t, err)

		// Verify directory exists
		_, err = os.Stat(storageDir)
		require.NoError(t, err, "storage directory should exist before cleanup")

		// Cleanup
		cleanup()

		// Verify directory is removed
		_, err = os.Stat(storageDir)
		assert.True(t, os.IsNotExist(err), "storage directory should be removed after cleanup")
	})

	t.Run("defer cleanup pattern", func(t *testing.T) {
		storageDir := "./test-defer-cleanup"

		func() {
			svc, cleanup, err := NewDevelopment(WithDevStorage(storageDir))
			require.NoError(t, err)
			defer cleanup() // Cleanup on function return

			// Use service
			ctx := context.Background()
			_, err = svc.UploadContent(ctx, simplecontent.UploadContentRequest{
				OwnerID:      uuid.New(),
				TenantID:     uuid.New(),
				Name:         "test.txt",
				DocumentType: "text/plain",
				Reader:       strings.NewReader("Defer test"),
				FileName:     "test.txt",
			})
			require.NoError(t, err)

			// Directory should exist during function
			_, err = os.Stat(storageDir)
			require.NoError(t, err)
		}()

		// After function returns (defer executed), directory should be gone
		_, err := os.Stat(storageDir)
		assert.True(t, os.IsNotExist(err), "storage directory should be removed after defer cleanup")
	})
}

func TestPresetIsolation(t *testing.T) {
	t.Run("testing presets are isolated", func(t *testing.T) {
		// Create two test services
		svc1 := NewTesting(t)
		svc2 := NewTesting(t)

		ctx := context.Background()

		// Upload to svc1
		content1, err := svc1.UploadContent(ctx, simplecontent.UploadContentRequest{
			OwnerID:      uuid.New(),
			TenantID:     uuid.New(),
			Name:         "test1.txt",
			DocumentType: "text/plain",
			Reader:       strings.NewReader("Service 1"),
			FileName:     "test1.txt",
		})
		require.NoError(t, err)

		// Content should NOT exist in svc2 (isolated)
		_, err = svc2.GetContent(ctx, content1.ID)
		assert.Error(t, err, "content from svc1 should not exist in svc2")
	})
}
