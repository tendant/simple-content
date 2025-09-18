// Deprecated: This module is deprecated and will be removed in a future version.
// Please use the new module instead.
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

type UploadS3ObjectParams = storage.UploadParams

type Backend = storage.Backend

type UploadObjectParams = storage.UploadParams

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

const (
	ContentStorageTypeS3    = "s3"
	ContentStorageTypeMinio = "minio"
)

const (
	MimeTypeWordDocx = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	MimeTypeWordDoc  = "application/msword"
	MimeTypeWordDotx = "application/vnd.openxmlformats-officedocument.wordprocessingml.template"
	MimeTypeWordDot  = "application/vnd.ms-word.document.macroEnabled.12"
	MimeTypeWordXLSX = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	MimeTypeWordXLS  = "application/vnd.ms-excel"
	MimeTypeWordXLSM = "application/vnd.ms-excel.sheet.macroEnabled.12"
	MimeTypeWordXLTX = "application/vnd.openxmlformats-officedocument.spreadsheetml.template"
	MimeTypeWordPPT  = "application/vnd.ms-powerpoint"
	MimeTypeWordPPTX = "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	MimeTypeWordPPTM = "application/vnd.ms-powerpoint.presentation.macroEnabled.12"
	MimeTypeWordPOTX = "application/vnd.openxmlformats-officedocument.presentationml.template"
	MimeTypeWordPPSX = "application/vnd.openxmlformats-officedocument.presentationml.slideshow"
)

var MicrosoftMimeTypeMap = map[string]string{
	MimeTypeWordDocx: "docx",
	MimeTypeWordDoc:  "doc",
	MimeTypeWordDotx: "dotx",
	MimeTypeWordDot:  "dot",
	MimeTypeWordPPTX: "pptx",
	MimeTypeWordXLSX: "xlsx",
	MimeTypeWordXLS:  "xls",
	MimeTypeWordXLSM: "xlsm",
	MimeTypeWordXLTX: "xltx",
	MimeTypeWordPPT:  "ppt",
	MimeTypeWordPPTM: "pptm",
	MimeTypeWordPOTX: "potx",
	MimeTypeWordPPSX: "ppsx",
}
