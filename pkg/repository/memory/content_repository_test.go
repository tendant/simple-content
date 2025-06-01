package memory_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/tendant/simple-content/internal/domain"
	repoMemory "github.com/tendant/simple-content/pkg/repository/memory"
)

func TestContentRepository_Create(t *testing.T) {
	repo := repoMemory.NewContentRepository()
	ctx := context.Background()

	content := &domain.Content{
		ID:             uuid.New(),
		DerivationType: "original",
	}

	err := repo.Create(ctx, content)
	assert.NoError(t, err)

	// Try to create the same content again (should fail)
	err = repo.Create(ctx, content)
	assert.Error(t, err)
}

func TestContentRepository_Get(t *testing.T) {
	repo := repoMemory.NewContentRepository()
	ctx := context.Background()

	contentID := uuid.New()
	content := &domain.Content{
		ID:             contentID,
		DerivationType: "original",
	}

	err := repo.Create(ctx, content)
	assert.NoError(t, err)

	// Get the content
	retrieved, err := repo.Get(ctx, contentID)
	assert.NoError(t, err)
	assert.Equal(t, contentID, retrieved.ID)
	assert.Equal(t, "original", retrieved.DerivationType)

	// Try to get non-existent content
	_, err = repo.Get(ctx, uuid.New())
	assert.Error(t, err)
}
