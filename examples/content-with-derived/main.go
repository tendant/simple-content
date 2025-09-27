package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/tendant/simple-content/pkg/simplecontent"
	"github.com/tendant/simple-content/pkg/simplecontent/config"
)

// ContentWithDerived represents a content item with its derived contents
type ContentWithDerived struct {
	*simplecontent.Content
	DerivedContents []*DerivedContentItem `json:"derived_contents,omitempty"`
	ParentContent   *ContentReference     `json:"parent_content,omitempty"`
}

// DerivedContentItem represents a derived content with full details
type DerivedContentItem struct {
	*simplecontent.Content
	Variant             string                        `json:"variant"`
	DerivationParams    map[string]interface{}        `json:"derivation_params,omitempty"`
	ProcessingMetadata  map[string]interface{}        `json:"processing_metadata,omitempty"`
	Objects             []*simplecontent.Object       `json:"objects,omitempty"`
	Metadata            *simplecontent.ContentMetadata `json:"metadata,omitempty"`
}

// ContentReference represents a lightweight parent content reference
type ContentReference struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	DocumentType   string    `json:"document_type"`
	DerivationType string    `json:"derivation_type,omitempty"`
}

// GetContentWithDerivedOptions provides options for the fetch operation
type GetContentWithDerivedOptions struct {
	IncludeObjects   bool     `json:"include_objects"`
	IncludeMetadata  bool     `json:"include_metadata"`
	MaxDepth         int      `json:"max_depth"`
	DerivationFilter []string `json:"derivation_filter,omitempty"`
}

// ExtendedContentService wraps the simple-content service with enhanced derived content capabilities
type ExtendedContentService struct {
	svc simplecontent.Service
}

// NewExtendedContentService creates a service with enhanced derived content capabilities
func NewExtendedContentService() (*ExtendedContentService, error) {
	// Use memory storage for demo - the default config already sets up memory storage
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to create config: %w", err)
	}

	svc, err := cfg.BuildService()
	if err != nil {
		return nil, fmt.Errorf("failed to build service: %w", err)
	}

	return &ExtendedContentService{svc: svc}, nil
}

// GetContentWithDerived retrieves content with its derived contents
func (ecs *ExtendedContentService) GetContentWithDerived(ctx context.Context, contentID uuid.UUID, opts *GetContentWithDerivedOptions) (*ContentWithDerived, error) {
	if opts == nil {
		opts = &GetContentWithDerivedOptions{
			IncludeObjects:  false,
			IncludeMetadata: false,
			MaxDepth:        1,
		}
	}

	// Get the main content
	content, err := ecs.svc.GetContent(ctx, contentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get content: %w", err)
	}

	result := &ContentWithDerived{
		Content: content,
	}

	// If this is derived content, get parent reference
	if content.DerivationType != "" {
		parentRef, err := ecs.getParentReference(ctx, contentID)
		if err != nil {
			// Log but don't fail - parent reference is optional
			log.Printf("Warning: failed to get parent reference for content %s: %v", contentID, err)
		} else {
			result.ParentContent = parentRef
		}
	}

	// Get derived contents
	derivedContents, err := ecs.getDerivedContentsWithDetails(ctx, contentID, opts, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get derived contents: %w", err)
	}
	result.DerivedContents = derivedContents

	return result, nil
}

// GetMultipleContentWithDerived retrieves multiple contents with their derived contents
func (ecs *ExtendedContentService) GetMultipleContentWithDerived(ctx context.Context, contentIDs []uuid.UUID, opts *GetContentWithDerivedOptions) ([]*ContentWithDerived, error) {
	if len(contentIDs) == 0 {
		return []*ContentWithDerived{}, nil
	}

	results := make([]*ContentWithDerived, 0, len(contentIDs))

	for _, contentID := range contentIDs {
		contentWithDerived, err := ecs.GetContentWithDerived(ctx, contentID, opts)
		if err != nil {
			// Log error but continue with other content
			log.Printf("Warning: failed to get content with derived for %s: %v", contentID, err)
			continue
		}
		results = append(results, contentWithDerived)
	}

	return results, nil
}

// GetContentHierarchy retrieves a complete content hierarchy
func (ecs *ExtendedContentService) GetContentHierarchy(ctx context.Context, rootContentID uuid.UUID, opts *GetContentWithDerivedOptions) (*ContentWithDerived, error) {
	if opts == nil {
		opts = &GetContentWithDerivedOptions{
			IncludeObjects:  false,
			IncludeMetadata: false,
			MaxDepth:        10, // Higher default for hierarchy
		}
	}

	// Ensure max depth is set for hierarchy
	if opts.MaxDepth == 0 {
		opts.MaxDepth = 10
	}

	return ecs.GetContentWithDerived(ctx, rootContentID, opts)
}

// Helper methods

func (ecs *ExtendedContentService) getParentReference(ctx context.Context, contentID uuid.UUID) (*ContentReference, error) {
	relationship, err := ecs.svc.GetDerivedRelationship(ctx, contentID)
	if err != nil {
		return nil, err
	}

	parentContent, err := ecs.svc.GetContent(ctx, relationship.ParentID)
	if err != nil {
		return nil, err
	}

	return &ContentReference{
		ID:             parentContent.ID,
		Name:           parentContent.Name,
		DocumentType:   parentContent.DocumentType,
		DerivationType: parentContent.DerivationType,
	}, nil
}

func (ecs *ExtendedContentService) getDerivedContentsWithDetails(ctx context.Context, parentID uuid.UUID, opts *GetContentWithDerivedOptions, currentDepth int) ([]*DerivedContentItem, error) {
	// Check depth limit
	if currentDepth >= opts.MaxDepth {
		return []*DerivedContentItem{}, nil
	}

	// Get derived relationships
	relationships, err := ecs.svc.ListDerivedContent(ctx, simplecontent.WithParentID(parentID))
	if err != nil {
		return nil, err
	}

	if len(relationships) == 0 {
		return []*DerivedContentItem{}, nil
	}

	results := make([]*DerivedContentItem, 0, len(relationships))

	for _, rel := range relationships {
		// Apply derivation filter if specified
		if len(opts.DerivationFilter) > 0 {
			if !contains(opts.DerivationFilter, rel.DerivationType) {
				continue
			}
		}

		// Get the derived content
		derivedContent, err := ecs.svc.GetContent(ctx, rel.ContentID)
		if err != nil {
			log.Printf("Warning: failed to get derived content %s: %v", rel.ContentID, err)
			continue
		}

		item := &DerivedContentItem{
			Content:            derivedContent,
			Variant:            rel.DerivationType, // This should be the variant from the relationship
			DerivationParams:   rel.DerivationParams,
			ProcessingMetadata: rel.ProcessingMetadata,
		}

		// Include objects if requested
		if opts.IncludeObjects {
			objects, err := ecs.svc.GetObjectsByContentID(ctx, derivedContent.ID)
			if err != nil {
				log.Printf("Warning: failed to get objects for content %s: %v", derivedContent.ID, err)
			} else {
				item.Objects = objects
			}
		}

		// Include metadata if requested
		if opts.IncludeMetadata {
			metadata, err := ecs.svc.GetContentMetadata(ctx, derivedContent.ID)
			if err != nil {
				log.Printf("Warning: failed to get metadata for content %s: %v", derivedContent.ID, err)
			} else {
				item.Metadata = metadata
			}
		}

		results = append(results, item)
	}

	return results, nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// HTTP Server Implementation

func (ecs *ExtendedContentService) setupRoutes() http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	// CORS for development
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"status":  "healthy",
			"service": "content-with-derived-demo",
		})
	})

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Enhanced content endpoints
		r.Get("/contents/{contentID}/with-derived", ecs.handleGetContentWithDerived)
		r.Post("/contents/batch/with-derived", ecs.handleGetMultipleContentWithDerived)
		r.Get("/contents/{contentID}/hierarchy", ecs.handleGetContentHierarchy)

		// Basic content management for demo setup
		r.Post("/contents", ecs.handleCreateContent)
		r.Post("/contents/{parentID}/derived", ecs.handleCreateDerivedContent)
		r.Get("/contents", ecs.handleListContents)
		r.Get("/demo/setup", ecs.handleDemoSetup)
	})

	// Serve demo page
	r.Get("/", ecs.serveDemoPage)

	return r
}

// HTTP Handlers

func (ecs *ExtendedContentService) handleGetContentWithDerived(w http.ResponseWriter, r *http.Request) {
	contentIDStr := chi.URLParam(r, "contentID")
	contentID, err := uuid.Parse(contentIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_content_id", "contentID must be a UUID")
		return
	}

	// Parse query parameters
	opts := &GetContentWithDerivedOptions{
		IncludeObjects:  r.URL.Query().Get("include_objects") == "true",
		IncludeMetadata: r.URL.Query().Get("include_metadata") == "true",
		MaxDepth:        1, // Default
	}

	if maxDepthStr := r.URL.Query().Get("max_depth"); maxDepthStr != "" {
		if maxDepth, err := strconv.Atoi(maxDepthStr); err == nil && maxDepth > 0 {
			opts.MaxDepth = maxDepth
		}
	}

	if derivationFilter := r.URL.Query().Get("derivation_filter"); derivationFilter != "" {
		opts.DerivationFilter = strings.Split(derivationFilter, ",")
	}

	result, err := ecs.GetContentWithDerived(r.Context(), contentID, opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "fetch_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (ecs *ExtendedContentService) handleGetMultipleContentWithDerived(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ContentIDs []string                      `json:"content_ids"`
		Options    *GetContentWithDerivedOptions `json:"options"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON in request body")
		return
	}

	if len(req.ContentIDs) == 0 {
		writeError(w, http.StatusBadRequest, "missing_content_ids", "content_ids is required")
		return
	}

	// Parse UUIDs
	contentIDs := make([]uuid.UUID, 0, len(req.ContentIDs))
	for _, idStr := range req.ContentIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_content_id", fmt.Sprintf("Invalid content ID: %s", idStr))
			return
		}
		contentIDs = append(contentIDs, id)
	}

	if req.Options == nil {
		req.Options = &GetContentWithDerivedOptions{
			IncludeObjects:  false,
			IncludeMetadata: false,
			MaxDepth:        1,
		}
	}

	results, err := ecs.GetMultipleContentWithDerived(r.Context(), contentIDs, req.Options)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "fetch_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"contents": results,
		"count":    len(results),
	})
}

func (ecs *ExtendedContentService) handleGetContentHierarchy(w http.ResponseWriter, r *http.Request) {
	contentIDStr := chi.URLParam(r, "contentID")
	contentID, err := uuid.Parse(contentIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_content_id", "contentID must be a UUID")
		return
	}

	// Parse query parameters - hierarchy typically includes more data
	opts := &GetContentWithDerivedOptions{
		IncludeObjects:  r.URL.Query().Get("include_objects") == "true",
		IncludeMetadata: r.URL.Query().Get("include_metadata") == "true",
		MaxDepth:        10, // Higher default for hierarchy
	}

	if maxDepthStr := r.URL.Query().Get("max_depth"); maxDepthStr != "" {
		if maxDepth, err := strconv.Atoi(maxDepthStr); err == nil && maxDepth > 0 {
			opts.MaxDepth = maxDepth
		}
	}

	if derivationFilter := r.URL.Query().Get("derivation_filter"); derivationFilter != "" {
		opts.DerivationFilter = strings.Split(derivationFilter, ",")
	}

	result, err := ecs.GetContentHierarchy(r.Context(), contentID, opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "fetch_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// Demo setup handlers

func (ecs *ExtendedContentService) handleCreateContent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name         string `json:"name"`
		Description  string `json:"description"`
		DocumentType string `json:"document_type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON")
		return
	}

	// Use demo UUIDs
	ownerID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	tenantID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")

	content, err := ecs.svc.CreateContent(r.Context(), simplecontent.CreateContentRequest{
		OwnerID:      ownerID,
		TenantID:     tenantID,
		Name:         req.Name,
		Description:  req.Description,
		DocumentType: req.DocumentType,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, content)
}

func (ecs *ExtendedContentService) handleCreateDerivedContent(w http.ResponseWriter, r *http.Request) {
	parentIDStr := chi.URLParam(r, "parentID")
	parentID, err := uuid.Parse(parentIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_parent_id", "parentID must be a UUID")
		return
	}

	var req struct {
		Name           string                 `json:"name"`
		Description    string                 `json:"description"`
		DocumentType   string                 `json:"document_type"`
		DerivationType string                 `json:"derivation_type"`
		Variant        string                 `json:"variant"`
		Metadata       map[string]interface{} `json:"metadata"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON")
		return
	}

	// Get parent content for owner/tenant info
	parentContent, err := ecs.svc.GetContent(r.Context(), parentID)
	if err != nil {
		writeError(w, http.StatusNotFound, "parent_not_found", "Parent content not found")
		return
	}

	content, err := ecs.svc.CreateDerivedContent(r.Context(), simplecontent.CreateDerivedContentRequest{
		ParentID:       parentID,
		OwnerID:        parentContent.OwnerID,
		TenantID:       parentContent.TenantID,
		DerivationType: req.DerivationType,
		Variant:        req.Variant,
		Metadata:       req.Metadata,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create_failed", err.Error())
		return
	}

	// Update the content with custom name and description
	content.Name = req.Name
	content.Description = req.Description
	content.DocumentType = req.DocumentType

	err = ecs.svc.UpdateContent(r.Context(), simplecontent.UpdateContentRequest{Content: content})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "update_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, content)
}

func (ecs *ExtendedContentService) handleListContents(w http.ResponseWriter, r *http.Request) {
	// Use demo UUIDs
	ownerID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	tenantID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")

	contents, err := ecs.svc.ListContent(r.Context(), simplecontent.ListContentRequest{
		OwnerID:  ownerID,
		TenantID: tenantID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"contents": contents,
		"count":    len(contents),
	})
}

func (ecs *ExtendedContentService) handleDemoSetup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Create original content
	original, err := ecs.svc.CreateContent(ctx, simplecontent.CreateContentRequest{
		OwnerID:      uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		TenantID:     uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
		Name:         "Original Photo",
		Description:  "A high-resolution photograph",
		DocumentType: "image/jpeg",
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "setup_failed", err.Error())
		return
	}

	// Create thumbnails
	sizes := []string{"128", "256", "512"}
	for _, size := range sizes {
		_, err = ecs.svc.CreateDerivedContent(ctx, simplecontent.CreateDerivedContentRequest{
			ParentID:       original.ID,
			OwnerID:        original.OwnerID,
			TenantID:       original.TenantID,
			DerivationType: "thumbnail",
			Variant:        "thumbnail_" + size,
			Metadata: map[string]interface{}{
				"size":      size + "px",
				"algorithm": "lanczos3",
			},
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "setup_failed", err.Error())
			return
		}
	}

	// Create preview
	_, err = ecs.svc.CreateDerivedContent(ctx, simplecontent.CreateDerivedContentRequest{
		ParentID:       original.ID,
		OwnerID:        original.OwnerID,
		TenantID:       original.TenantID,
		DerivationType: "preview",
		Variant:        "preview_web",
		Metadata: map[string]interface{}{
			"format":  "webp",
			"quality": 80,
		},
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "setup_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":            "Demo data created successfully",
		"original_id":        original.ID,
		"derived_count":      4,
		"thumbnail_variants": []string{"thumbnail_128", "thumbnail_256", "thumbnail_512"},
		"preview_variants":   []string{"preview_web"},
	})
}

// Utility functions

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

func (ecs *ExtendedContentService) serveDemoPage(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Content with Derived Demo</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 1200px; margin: 0 auto; padding: 20px; }
        .section { margin-bottom: 30px; padding: 20px; border: 1px solid #ddd; border-radius: 5px; }
        .button { background: #007bff; color: white; padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer; margin: 5px; }
        .button:hover { background: #0056b3; }
        .result { background: #f8f9fa; padding: 15px; border-radius: 5px; margin: 10px 0; }
        .json { background: #2d3748; color: #e2e8f0; padding: 15px; border-radius: 5px; overflow-x: auto; }
        .options { margin: 15px 0; }
        .options label { display: inline-block; margin-right: 15px; }
        .options input[type="checkbox"] { margin-right: 5px; }
        .options input[type="number"] { width: 60px; margin-left: 5px; }
        .filter-input { width: 200px; margin-left: 5px; }
        .content-item { border: 1px solid #ccc; padding: 10px; margin: 5px 0; border-radius: 3px; }
        .derived-item { margin-left: 20px; border-left: 3px solid #007bff; padding-left: 10px; }
    </style>
</head>
<body>
    <h1>Content with Derived Demo</h1>
    <p>This demo shows how to fetch content along with its derived contents in a single API call.</p>

    <div class="section">
        <h2>Setup Demo Data</h2>
        <p>Create sample content with thumbnails and previews.</p>
        <button class="button" onclick="setupDemo()">Create Demo Data</button>
        <div id="setupResult"></div>
    </div>

    <div class="section">
        <h2>Fetch Options</h2>
        <div class="options">
            <label><input type="checkbox" id="includeObjects"> Include Objects</label>
            <label><input type="checkbox" id="includeMetadata" checked> Include Metadata</label>
            <label>Max Depth: <input type="number" id="maxDepth" value="3" min="1" max="10"></label>
            <label>Filter: <input type="text" id="derivationFilter" class="filter-input" placeholder="thumbnail,preview"></label>
        </div>
    </div>

    <div class="section">
        <h2>Content Operations</h2>
        <button class="button" onclick="listContents()">List All Contents</button>
        <button class="button" onclick="getContentWithDerived()">Get Content with Derived</button>
        <button class="button" onclick="getContentHierarchy()">Get Content Hierarchy</button>
        <div id="operationResult"></div>
    </div>

    <div class="section">
        <h2>Available Contents</h2>
        <div id="contentsList"></div>
    </div>

    <script>
        let demoContentId = null;

        async function setupDemo() {
            try {
                const response = await fetch('/api/v1/demo/setup');
                const result = await response.json();

                if (response.ok) {
                    demoContentId = result.original_id;
                    document.getElementById('setupResult').innerHTML =
                        '<div class="result">Demo data created successfully!<br>Original ID: ' + result.original_id + '</div>';
                    await listContents();
                } else {
                    document.getElementById('setupResult').innerHTML =
                        '<div class="result" style="background: #f8d7da;">Error: ' + result.error.message + '</div>';
                }
            } catch (error) {
                document.getElementById('setupResult').innerHTML =
                    '<div class="result" style="background: #f8d7da;">Error: ' + error.message + '</div>';
            }
        }

        async function listContents() {
            try {
                const response = await fetch('/api/v1/contents');
                const result = await response.json();

                if (response.ok) {
                    displayContents(result.contents);
                } else {
                    console.error('Failed to list contents:', result);
                }
            } catch (error) {
                console.error('Error listing contents:', error);
            }
        }

        function displayContents(contents) {
            const container = document.getElementById('contentsList');
            if (!contents || contents.length === 0) {
                container.innerHTML = '<p>No content found. Create demo data first.</p>';
                return;
            }

            const html = contents.map(content =>
                '<div class="content-item">' +
                '<strong>' + content.name + '</strong> (' + content.id + ')<br>' +
                'Type: ' + content.document_type + '<br>' +
                'Derivation: ' + (content.derivation_type || 'Original') + '<br>' +
                '<button class="button" onclick="fetchContentWithDerived(\'' + content.id + '\')">Fetch with Derived</button>' +
                '</div>'
            ).join('');

            container.innerHTML = html;
        }

        function getOptions() {
            return {
                include_objects: document.getElementById('includeObjects').checked,
                include_metadata: document.getElementById('includeMetadata').checked,
                max_depth: parseInt(document.getElementById('maxDepth').value) || 3,
                derivation_filter: document.getElementById('derivationFilter').value.split(',').map(s => s.trim()).filter(s => s)
            };
        }

        async function getContentWithDerived() {
            if (!demoContentId) {
                alert('Please create demo data first');
                return;
            }
            await fetchContentWithDerived(demoContentId);
        }

        async function fetchContentWithDerived(contentId) {
            try {
                const options = getOptions();
                const params = new URLSearchParams();

                if (options.include_objects) params.set('include_objects', 'true');
                if (options.include_metadata) params.set('include_metadata', 'true');
                params.set('max_depth', options.max_depth.toString());
                if (options.derivation_filter.length > 0) {
                    params.set('derivation_filter', options.derivation_filter.join(','));
                }

                const response = await fetch('/api/v1/contents/' + contentId + '/with-derived?' + params.toString());
                const result = await response.json();

                if (response.ok) {
                    displayResult('Content with Derived', result);
                } else {
                    displayError('Failed to fetch content with derived: ' + result.error.message);
                }
            } catch (error) {
                displayError('Error: ' + error.message);
            }
        }

        async function getContentHierarchy() {
            if (!demoContentId) {
                alert('Please create demo data first');
                return;
            }

            try {
                const options = getOptions();
                const params = new URLSearchParams();

                if (options.include_objects) params.set('include_objects', 'true');
                if (options.include_metadata) params.set('include_metadata', 'true');
                params.set('max_depth', options.max_depth.toString());
                if (options.derivation_filter.length > 0) {
                    params.set('derivation_filter', options.derivation_filter.join(','));
                }

                const response = await fetch('/api/v1/contents/' + demoContentId + '/hierarchy?' + params.toString());
                const result = await response.json();

                if (response.ok) {
                    displayResult('Content Hierarchy', result);
                } else {
                    displayError('Failed to fetch content hierarchy: ' + result.error.message);
                }
            } catch (error) {
                displayError('Error: ' + error.message);
            }
        }

        function displayResult(title, data) {
            const container = document.getElementById('operationResult');
            container.innerHTML =
                '<h3>' + title + '</h3>' +
                '<div class="json">' + JSON.stringify(data, null, 2) + '</div>';
        }

        function displayError(message) {
            const container = document.getElementById('operationResult');
            container.innerHTML = '<div class="result" style="background: #f8d7da;">' + message + '</div>';
        }

        // Load contents on page load
        listContents();
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func main() {
	fmt.Println("=== Content with Derived Demo Server ===")
	fmt.Println()
	fmt.Println("This demo shows how to fetch content with derived contents efficiently.")

	// Create service
	service, err := NewExtendedContentService()
	if err != nil {
		log.Fatal("Failed to create service:", err)
	}

	// Start HTTP server
	port := "8080"
	server := &http.Server{
		Addr:    ":" + port,
		Handler: service.setupRoutes(),
	}

	fmt.Printf("Server starting on http://localhost:%s\n", port)
	fmt.Println("Open your browser to see the demo!")
	fmt.Println()
	fmt.Println("API Endpoints:")
	fmt.Println("  GET  /api/v1/contents/{id}/with-derived     - Get content with derived")
	fmt.Println("  POST /api/v1/contents/batch/with-derived    - Get multiple contents with derived")
	fmt.Println("  GET  /api/v1/contents/{id}/hierarchy        - Get content hierarchy")
	fmt.Println("  GET  /api/v1/demo/setup                     - Create demo data")
	fmt.Println()

	log.Fatal(server.ListenAndServe())
}