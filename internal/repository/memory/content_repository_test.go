package memory_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/tendant/simple-content/internal/domain"
	"github.com/tendant/simple-content/internal/repository/memory"
)

func TestContentRepository_Create(t *testing.T) {
	repo := memory.NewContentRepository()
	ctx := context.Background()

	content := &domain.Content{
		ID:              uuid.New(),
		DerivationType:  "original",
		DerivationLevel: 0,
	}

	err := repo.Create(ctx, content)
	assert.NoError(t, err)

	// Try to create the same content again (should fail)
	err = repo.Create(ctx, content)
	assert.Error(t, err)
}

func TestContentRepository_Get(t *testing.T) {
	repo := memory.NewContentRepository()
	ctx := context.Background()

	contentID := uuid.New()
	content := &domain.Content{
		ID:              contentID,
		DerivationType:  "original",
		DerivationLevel: 0,
	}

	err := repo.Create(ctx, content)
	assert.NoError(t, err)

	// Get the content
	retrieved, err := repo.Get(ctx, contentID)
	assert.NoError(t, err)
	assert.Equal(t, contentID, retrieved.ID)
	assert.Equal(t, "original", retrieved.DerivationType)
	assert.Equal(t, 0, retrieved.DerivationLevel)

	// Try to get non-existent content
	_, err = repo.Get(ctx, uuid.New())
	assert.Error(t, err)
}

func TestContentRepository_GetByParentID(t *testing.T) {
	repo := memory.NewContentRepository()
	ctx := context.Background()

	// Create parent content
	parentID := uuid.New()
	parentContent := &domain.Content{
		ID:              parentID,
		DerivationType:  "original",
		DerivationLevel: 0,
	}
	err := repo.Create(ctx, parentContent)
	assert.NoError(t, err)

	// Create derived content
	derivedID := uuid.New()
	derivedContent := &domain.Content{
		ID:              derivedID,
		ParentID:        &parentID,
		DerivationType:  "derived",
		DerivationLevel: 1,
	}
	err = repo.Create(ctx, derivedContent)
	assert.NoError(t, err)

	// Create another derived content with a different parent
	otherParentID := uuid.New()
	otherDerivedID := uuid.New()
	otherDerivedContent := &domain.Content{
		ID:              otherDerivedID,
		ParentID:        &otherParentID,
		DerivationType:  "derived",
		DerivationLevel: 1,
	}
	err = repo.Create(ctx, otherDerivedContent)
	assert.NoError(t, err)

	// Test GetByParentID
	results, err := repo.GetByParentID(ctx, parentID)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, derivedID, results[0].ID)

	// Test GetByParentID with non-existent parent
	results, err = repo.GetByParentID(ctx, uuid.New())
	assert.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestContentRepository_GetDerivedContentTree(t *testing.T) {
	repo := memory.NewContentRepository()
	ctx := context.Background()

	// Create a tree of content:
	// root -> child1 -> grandchild
	//      -> child2

	rootID := uuid.New()
	root := &domain.Content{
		ID:              rootID,
		DerivationType:  "original",
		DerivationLevel: 0,
	}
	err := repo.Create(ctx, root)
	assert.NoError(t, err)

	child1ID := uuid.New()
	child1 := &domain.Content{
		ID:              child1ID,
		ParentID:        &rootID,
		DerivationType:  "derived",
		DerivationLevel: 1,
	}
	err = repo.Create(ctx, child1)
	assert.NoError(t, err)

	child2ID := uuid.New()
	child2 := &domain.Content{
		ID:              child2ID,
		ParentID:        &rootID,
		DerivationType:  "derived",
		DerivationLevel: 1,
	}
	err = repo.Create(ctx, child2)
	assert.NoError(t, err)

	grandchildID := uuid.New()
	grandchild := &domain.Content{
		ID:              grandchildID,
		ParentID:        &child1ID,
		DerivationType:  "derived",
		DerivationLevel: 2,
	}
	err = repo.Create(ctx, grandchild)
	assert.NoError(t, err)

	// Test GetDerivedContentTree with max depth 5
	results, err := repo.GetDerivedContentTree(ctx, rootID, 5)
	assert.NoError(t, err)
	assert.Len(t, results, 4) // root + 2 children + 1 grandchild

	// Test GetDerivedContentTree with max depth 1
	results, err = repo.GetDerivedContentTree(ctx, rootID, 1)
	assert.NoError(t, err)
	assert.Len(t, results, 3) // root + 2 children (grandchild excluded)

	// Test GetDerivedContentTree with non-existent root
	_, err = repo.GetDerivedContentTree(ctx, uuid.New(), 5)
	assert.Error(t, err)
}
