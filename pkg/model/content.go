package model

import (
	"github.com/tendant/simple-content/internal/domain"
	"github.com/tendant/simple-content/internal/repository"
	"github.com/tendant/simple-content/internal/storage"
)

// Content represents a logical content entity
type Content = domain.Content

// ContentMetadata represents metadata for a content
type ContentMetadata = domain.ContentMetadata

// Object represents a physical object stored in a storage backend
type Object = domain.Object

// ObjectMetadata represents metadata about an object
type ObjectMetadata = domain.ObjectMetadata

// ObjectPreview represents a preview generated from an object
type ObjectPreview = domain.ObjectPreview

// StorageBackend represents a configurable storage backend
type StorageBackend = domain.StorageBackend

// ObjectMeta represents metadata about an object
type ObjectMeta = storage.ObjectMeta

// ListDerivedContentParams represents parameters for listing derived content
type ListDerivedContentParams = repository.ListDerivedContentParams

// Content status constants
const (
	ContentStatusCreated  = domain.ContentStatusCreated
	ContentStatusUploaded = domain.ContentStatusUploaded
)

// Content derivation type constants
const (
	ContentCategoryOriginal                  = "original"
	ContentCategoryThumbnail                 = "thumbnail"
	ContentDerivedDerivationTypeTHUMBNAIL720 = "THUMBNAIL_720"
	ContentDerivedDerivationTypeTHUMBNAIL480 = "THUMBNAIL_480"
	ContentDerivedDerivationTypeTHUMBNAIL256 = "THUMBNAIL_256"
	ContentDerivedDerivationTypeTHUMBNAIL128 = "THUMBNAIL_128"
)

// Object status constants
const (
	ObjectStatusCreated    = domain.ObjectStatusCreated
	ObjectStatusUploading  = domain.ObjectStatusUploading
	ObjectStatusUploaded   = domain.ObjectStatusUploaded
	ObjectStatusProcessing = domain.ObjectStatusProcessing
	ObjectStatusProcessed  = domain.ObjectStatusProcessed
	ObjectStatusFailed     = domain.ObjectStatusFailed
	ObjectStatusDeleted    = domain.ObjectStatusDeleted
)
