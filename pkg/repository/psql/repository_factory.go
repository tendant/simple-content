// Deprecated: This module is deprecated and will be removed in a future version.
// Please use the new module instead.
package repository

import "github.com/tendant/simple-content/internal/repository"

// RepositoryFactory creates and returns all repository implementations
type RepositoryFactory struct {
	db DBTX
}

// NewRepositoryFactory creates a new repository factory
func NewRepositoryFactory(db DBTX) *RepositoryFactory {
	return &RepositoryFactory{
		db: db,
	}
}

// NewContentRepository creates a new content repository
func (f *RepositoryFactory) NewContentRepository() repository.ContentRepository {
	return NewPSQLContentRepository(f.db)
}

// NewContentMetadataRepository creates a new content metadata repository
func (f *RepositoryFactory) NewContentMetadataRepository() repository.ContentMetadataRepository {
	return NewPSQLContentMetadataRepository(f.db)
}

// NewObjectRepository creates a new object repository
func (f *RepositoryFactory) NewObjectRepository() repository.ObjectRepository {
	return NewPSQLObjectRepository(f.db)
}

// NewObjectMetadataRepository creates a new object metadata repository
func (f *RepositoryFactory) NewObjectMetadataRepository() repository.ObjectMetadataRepository {
	return NewPSQLObjectMetadataRepository(f.db)
}
