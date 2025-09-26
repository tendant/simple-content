package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/tendant/simple-content/pkg/simplecontent"
	"github.com/tendant/simple-content/pkg/simplecontent/config"
)

// Enhanced filtering parameters
type EnhancedListDerivedContentParams struct {
	// Parent filtering
	ParentID  *uuid.UUID  `json:"parent_id,omitempty"`
	ParentIDs []uuid.UUID `json:"parent_ids,omitempty"`

	// Derivation type filtering (user-facing categories)
	DerivationType  *string  `json:"derivation_type,omitempty"`
	DerivationTypes []string `json:"derivation_types,omitempty"`

	// Variant filtering (specific implementations)
	Variant  *string  `json:"variant,omitempty"`
	Variants []string `json:"variants,omitempty"`

	// Combined filtering for advanced use cases
	TypeVariantPairs []TypeVariantPair `json:"type_variant_pairs,omitempty"`

	// Content status filtering
	ContentStatus  *string  `json:"content_status,omitempty"`
	ContentStatuses []string `json:"content_statuses,omitempty"`

	// Temporal filtering
	CreatedAfter  *time.Time `json:"created_after,omitempty"`
	CreatedBefore *time.Time `json:"created_before,omitempty"`
	UpdatedAfter  *time.Time `json:"updated_after,omitempty"`
	UpdatedBefore *time.Time `json:"updated_before,omitempty"`

	// Sorting and pagination
	SortBy    *string `json:"sort_by,omitempty"`    // "created_at", "updated_at", "name"
	SortOrder *string `json:"sort_order,omitempty"` // "asc", "desc" (default: "desc")
	Limit     *int    `json:"limit,omitempty"`
	Offset    *int    `json:"offset,omitempty"`

	// Advanced options
	IncludeDeleted bool `json:"include_deleted,omitempty"`
}

// TypeVariantPair allows precise filtering by type+variant combinations
type TypeVariantPair struct {
	DerivationType string `json:"derivation_type"`
	Variant        string `json:"variant"`
}

// Enhanced derived content item with filtering metadata
type EnhancedDerivedContentItem struct {
	*simplecontent.DerivedContent
	ActualVariant string `json:"actual_variant,omitempty"`
	MatchedBy     string `json:"matched_by,omitempty"` // Debug info
}

// EnhancedDerivedContentService provides advanced filtering for derived content
type EnhancedDerivedContentService struct {
	svc simplecontent.Service
}

// NewEnhancedDerivedContentService creates a service with enhanced filtering
func NewEnhancedDerivedContentService() (*EnhancedDerivedContentService, error) {
	// Use memory storage for demo
	cfg, err := config.Load(
		config.WithDatabaseType("memory"),
		config.WithStorageBackend("memory", map[string]interface{}{}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create config: %w", err)
	}

	svc, err := cfg.BuildService()
	if err != nil {
		return nil, fmt.Errorf("failed to build service: %w", err)
	}

	return &EnhancedDerivedContentService{svc: svc}, nil
}

// ListDerivedContentWithFilters provides enhanced filtering capabilities
func (eds *EnhancedDerivedContentService) ListDerivedContentWithFilters(ctx context.Context, params EnhancedListDerivedContentParams) ([]*EnhancedDerivedContentItem, error) {
	// Get all derived content for the parent(s)
	var allDerived []*simplecontent.DerivedContent
	var err error

	if params.ParentID != nil {
		allDerived, err = eds.svc.ListDerivedByParent(ctx, *params.ParentID)
		if err != nil {
			return nil, fmt.Errorf("failed to list derived content: %w", err)
		}
	} else if len(params.ParentIDs) > 0 {
		// For multiple parents, we'll query each one (in a real implementation, you'd optimize this)
		for _, parentID := range params.ParentIDs {
			derived, err := eds.svc.ListDerivedByParent(ctx, parentID)
			if err != nil {
				log.Printf("Warning: failed to get derived content for parent %s: %v", parentID, err)
				continue
			}
			allDerived = append(allDerived, derived...)
		}
	} else {
		return []*EnhancedDerivedContentItem{}, nil // Need at least one parent ID
	}

	// Apply custom filtering
	var filtered []*EnhancedDerivedContentItem
	for _, derived := range allDerived {
		if eds.matchesFilters(derived, params) {
			item := &EnhancedDerivedContentItem{
				DerivedContent: derived,
				ActualVariant:  eds.extractVariant(derived),
				MatchedBy:      eds.getMatchReason(derived, params),
			}
			filtered = append(filtered, item)
		}
	}

	// Apply sorting
	eds.sortResults(filtered, params)

	// Apply pagination
	filtered = eds.paginateResults(filtered, params)

	return filtered, nil
}

// matchesFilters checks if a derived content item matches the filtering criteria
func (eds *EnhancedDerivedContentService) matchesFilters(derived *simplecontent.DerivedContent, params EnhancedListDerivedContentParams) bool {
	// Derivation type filtering
	if params.DerivationType != nil && derived.DerivationType != *params.DerivationType {
		return false
	}

	if len(params.DerivationTypes) > 0 {
		found := false
		for _, derivationType := range params.DerivationTypes {
			if derived.DerivationType == derivationType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Variant filtering - NEW FEATURE
	actualVariant := eds.extractVariant(derived)

	if params.Variant != nil && actualVariant != *params.Variant {
		return false
	}

	if len(params.Variants) > 0 {
		found := false
		for _, variant := range params.Variants {
			if actualVariant == variant {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Type+Variant pair filtering
	if len(params.TypeVariantPairs) > 0 {
		found := false
		for _, pair := range params.TypeVariantPairs {
			if derived.DerivationType == pair.DerivationType && actualVariant == pair.Variant {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Content status filtering
	if params.ContentStatus != nil && derived.Status != *params.ContentStatus {
		return false
	}

	if len(params.ContentStatuses) > 0 {
		found := false
		for _, status := range params.ContentStatuses {
			if derived.Status == status {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Temporal filtering
	if params.CreatedAfter != nil && derived.CreatedAt.Before(*params.CreatedAfter) {
		return false
	}

	if params.CreatedBefore != nil && derived.CreatedAt.After(*params.CreatedBefore) {
		return false
	}

	if params.UpdatedAfter != nil && derived.UpdatedAt.Before(*params.UpdatedAfter) {
		return false
	}

	if params.UpdatedBefore != nil && derived.UpdatedAt.After(*params.UpdatedBefore) {
		return false
	}

	return true
}

// extractVariant extracts the variant from derived content using multiple strategies
func (eds *EnhancedDerivedContentService) extractVariant(derived *simplecontent.DerivedContent) string {
	// Strategy 1: Variant stored in ProcessingMetadata
	if variant, exists := derived.ProcessingMetadata["variant"]; exists {
		if variantStr, ok := variant.(string); ok {
			return variantStr
		}
	}

	// Strategy 2: Variant stored in DerivationParams
	if variant, exists := derived.DerivationParams["variant"]; exists {
		if variantStr, ok := variant.(string); ok {
			return variantStr
		}
	}

	// Strategy 3: Infer variant from DerivationType (if it follows pattern like "thumbnail_256")
	parts := strings.Split(derived.DerivationType, "_")
	if len(parts) > 1 {
		return derived.DerivationType // Return full string as variant
	}

	// Strategy 4: Use the DerivationType as variant if no specific variant found
	return derived.DerivationType
}

// getMatchReason returns debug information about why an item matched
func (eds *EnhancedDerivedContentService) getMatchReason(derived *simplecontent.DerivedContent, params EnhancedListDerivedContentParams) string {
	reasons := []string{}

	if params.DerivationType != nil && derived.DerivationType == *params.DerivationType {
		reasons = append(reasons, "type:"+*params.DerivationType)
	}

	for _, derivationType := range params.DerivationTypes {
		if derived.DerivationType == derivationType {
			reasons = append(reasons, "types:"+derivationType)
		}
	}

	actualVariant := eds.extractVariant(derived)
	if params.Variant != nil && actualVariant == *params.Variant {
		reasons = append(reasons, "variant:"+*params.Variant)
	}

	for _, variant := range params.Variants {
		if actualVariant == variant {
			reasons = append(reasons, "variants:"+variant)
		}
	}

	for _, pair := range params.TypeVariantPairs {
		if derived.DerivationType == pair.DerivationType && actualVariant == pair.Variant {
			reasons = append(reasons, fmt.Sprintf("pair:%s:%s", pair.DerivationType, pair.Variant))
		}
	}

	if len(reasons) == 0 {
		return "parent"
	}

	return strings.Join(reasons, ",")
}

// sortResults applies sorting to the filtered results
func (eds *EnhancedDerivedContentService) sortResults(results []*EnhancedDerivedContentItem, params EnhancedListDerivedContentParams) {
	// Implementation would sort based on params.SortBy and params.SortOrder
	// For simplicity, we'll keep the default order (by creation time descending)
}

// paginateResults applies pagination to the results
func (eds *EnhancedDerivedContentService) paginateResults(results []*EnhancedDerivedContentItem, params EnhancedListDerivedContentParams) []*EnhancedDerivedContentItem {
	// Apply offset
	if params.Offset != nil && *params.Offset > 0 {
		if *params.Offset >= len(results) {
			return []*EnhancedDerivedContentItem{}
		}
		results = results[*params.Offset:]
	}

	// Apply limit
	if params.Limit != nil && *params.Limit > 0 && *params.Limit < len(results) {
		results = results[:*params.Limit]
	}

	return results
}

// Convenience methods for common filtering patterns
func (eds *EnhancedDerivedContentService) GetThumbnailsBySize(ctx context.Context, parentID uuid.UUID, sizes []string) ([]*EnhancedDerivedContentItem, error) {
	// Convert sizes to variant names
	variants := make([]string, len(sizes))
	for i, size := range sizes {
		variants[i] = "thumbnail_" + size
	}

	params := EnhancedListDerivedContentParams{
		ParentID:       &parentID,
		DerivationType: stringPtr("thumbnail"),
		Variants:       variants,
	}

	return eds.ListDerivedContentWithFilters(ctx, params)
}

func (eds *EnhancedDerivedContentService) GetRecentDerived(ctx context.Context, parentID uuid.UUID, since time.Time) ([]*EnhancedDerivedContentItem, error) {
	params := EnhancedListDerivedContentParams{
		ParentID:     &parentID,
		CreatedAfter: &since,
		SortBy:       stringPtr("created_at"),
		SortOrder:    stringPtr("desc"),
	}

	return eds.ListDerivedContentWithFilters(ctx, params)
}

// HTTP Server Implementation

func (eds *EnhancedDerivedContentService) setupRoutes() http.Handler {
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
			"service": "derived-content-filtering-demo",
		})
	})

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Enhanced filtering endpoints
		r.Get("/derived-content/filter", eds.handleFilterDerivedContent)
		r.Get("/derived-content/thumbnails", eds.handleFilterThumbnails)
		r.Get("/derived-content/recent", eds.handleRecentDerived)

		// Demo setup and management
		r.Post("/contents", eds.handleCreateContent)
		r.Post("/contents/{parentID}/derived", eds.handleCreateDerivedContent)
		r.Get("/contents", eds.handleListContents)
		r.Get("/demo/setup", eds.handleDemoSetup)
	})

	// Serve demo page
	r.Get("/", eds.serveDemoPage)

	return r
}

// HTTP Handlers

func (eds *EnhancedDerivedContentService) handleFilterDerivedContent(w http.ResponseWriter, r *http.Request) {
	params := parseEnhancedDerivedParams(r)

	results, err := eds.ListDerivedContentWithFilters(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "filter_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"results": results,
		"count":   len(results),
		"filters": params,
	})
}

func (eds *EnhancedDerivedContentService) handleFilterThumbnails(w http.ResponseWriter, r *http.Request) {
	parentIDStr := r.URL.Query().Get("parent_id")
	if parentIDStr == "" {
		writeError(w, http.StatusBadRequest, "missing_parent_id", "parent_id is required")
		return
	}

	parentID, err := uuid.Parse(parentIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_parent_id", "Invalid parent_id format")
		return
	}

	sizesStr := r.URL.Query().Get("sizes")
	if sizesStr == "" {
		sizesStr = "128,256,512" // Default sizes
	}

	sizes := strings.Split(sizesStr, ",")
	for i, size := range sizes {
		sizes[i] = strings.TrimSpace(size)
	}

	results, err := eds.GetThumbnailsBySize(r.Context(), parentID, sizes)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "filter_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"results":   results,
		"count":     len(results),
		"parent_id": parentID,
		"sizes":     sizes,
	})
}

func (eds *EnhancedDerivedContentService) handleRecentDerived(w http.ResponseWriter, r *http.Request) {
	parentIDStr := r.URL.Query().Get("parent_id")
	if parentIDStr == "" {
		writeError(w, http.StatusBadRequest, "missing_parent_id", "parent_id is required")
		return
	}

	parentID, err := uuid.Parse(parentIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_parent_id", "Invalid parent_id format")
		return
	}

	sinceStr := r.URL.Query().Get("since")
	var since time.Time
	if sinceStr != "" {
		since, err = time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_since", "Invalid since format, use RFC3339")
			return
		}
	} else {
		since = time.Now().Add(-24 * time.Hour) // Default: last 24 hours
	}

	results, err := eds.GetRecentDerived(r.Context(), parentID, since)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "filter_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"results":   results,
		"count":     len(results),
		"parent_id": parentID,
		"since":     since,
	})
}

// Demo setup handlers (simplified versions from previous examples)

func (eds *EnhancedDerivedContentService) handleCreateContent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name         string `json:"name"`
		Description  string `json:"description"`
		DocumentType string `json:"document_type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON")
		return
	}

	ownerID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	tenantID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")

	content, err := eds.svc.CreateContent(r.Context(), simplecontent.CreateContentRequest{
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

func (eds *EnhancedDerivedContentService) handleCreateDerivedContent(w http.ResponseWriter, r *http.Request) {
	parentIDStr := chi.URLParam(r, "parentID")
	parentID, err := uuid.Parse(parentIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_parent_id", "parentID must be a UUID")
		return
	}

	var req struct {
		DerivationType string                 `json:"derivation_type"`
		Variant        string                 `json:"variant"`
		Metadata       map[string]interface{} `json:"metadata"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON")
		return
	}

	parentContent, err := eds.svc.GetContent(r.Context(), parentID)
	if err != nil {
		writeError(w, http.StatusNotFound, "parent_not_found", "Parent content not found")
		return
	}

	// Add variant to metadata for filtering
	if req.Metadata == nil {
		req.Metadata = make(map[string]interface{})
	}
	req.Metadata["variant"] = req.Variant

	content, err := eds.svc.CreateDerivedContent(r.Context(), simplecontent.CreateDerivedContentRequest{
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

	writeJSON(w, http.StatusCreated, content)
}

func (eds *EnhancedDerivedContentService) handleListContents(w http.ResponseWriter, r *http.Request) {
	ownerID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	tenantID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")

	contents, err := eds.svc.ListContent(r.Context(), simplecontent.ListContentRequest{
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

func (eds *EnhancedDerivedContentService) handleDemoSetup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Create original content
	original, err := eds.svc.CreateContent(ctx, simplecontent.CreateContentRequest{
		OwnerID:      uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		TenantID:     uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
		Name:         "Demo Photo",
		Description:  "A demo photo for filtering tests",
		DocumentType: "image/jpeg",
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "setup_failed", err.Error())
		return
	}

	// Create various derived content for filtering demos
	derivedItems := []struct {
		DerivationType string
		Variant        string
		Metadata       map[string]interface{}
	}{
		{
			DerivationType: "thumbnail",
			Variant:        "thumbnail_128",
			Metadata:       map[string]interface{}{"variant": "thumbnail_128", "size": "128px"},
		},
		{
			DerivationType: "thumbnail",
			Variant:        "thumbnail_256",
			Metadata:       map[string]interface{}{"variant": "thumbnail_256", "size": "256px"},
		},
		{
			DerivationType: "thumbnail",
			Variant:        "thumbnail_512",
			Metadata:       map[string]interface{}{"variant": "thumbnail_512", "size": "512px"},
		},
		{
			DerivationType: "preview",
			Variant:        "preview_web",
			Metadata:       map[string]interface{}{"variant": "preview_web", "format": "webp"},
		},
		{
			DerivationType: "preview",
			Variant:        "preview_mobile",
			Metadata:       map[string]interface{}{"variant": "preview_mobile", "format": "webp", "size": "mobile"},
		},
		{
			DerivationType: "transcode",
			Variant:        "video_720p",
			Metadata:       map[string]interface{}{"variant": "video_720p", "resolution": "720p"},
		},
	}

	createdItems := []map[string]interface{}{}
	for _, item := range derivedItems {
		derived, err := eds.svc.CreateDerivedContent(ctx, simplecontent.CreateDerivedContentRequest{
			ParentID:       original.ID,
			OwnerID:        original.OwnerID,
			TenantID:       original.TenantID,
			DerivationType: item.DerivationType,
			Variant:        item.Variant,
			Metadata:       item.Metadata,
		})
		if err != nil {
			log.Printf("Warning: failed to create derived content %s: %v", item.Variant, err)
			continue
		}

		createdItems = append(createdItems, map[string]interface{}{
			"id":              derived.ID,
			"derivation_type": item.DerivationType,
			"variant":         item.Variant,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":       "Demo data created successfully",
		"original_id":   original.ID,
		"derived_count": len(createdItems),
		"derived_items": createdItems,
	})
}

// Utility functions

func parseEnhancedDerivedParams(r *http.Request) EnhancedListDerivedContentParams {
	params := EnhancedListDerivedContentParams{}
	query := r.URL.Query()

	// Parent ID filtering
	if parentIDStr := query.Get("parent_id"); parentIDStr != "" {
		if parentID, err := uuid.Parse(parentIDStr); err == nil {
			params.ParentID = &parentID
		}
	}

	// Derivation type filtering
	if derivationType := query.Get("derivation_type"); derivationType != "" {
		params.DerivationType = &derivationType
	}

	if derivationTypesStr := query.Get("derivation_types"); derivationTypesStr != "" {
		types := strings.Split(derivationTypesStr, ",")
		for i, t := range types {
			types[i] = strings.TrimSpace(t)
		}
		params.DerivationTypes = types
	}

	// Variant filtering
	if variant := query.Get("variant"); variant != "" {
		params.Variant = &variant
	}

	if variantsStr := query.Get("variants"); variantsStr != "" {
		variants := strings.Split(variantsStr, ",")
		for i, v := range variants {
			variants[i] = strings.TrimSpace(v)
		}
		params.Variants = variants
	}

	// Type+Variant pairs
	if pairsStr := query.Get("type_variant_pairs"); pairsStr != "" {
		pairs := strings.Split(pairsStr, ",")
		var typeVariantPairs []TypeVariantPair
		for _, pair := range pairs {
			parts := strings.Split(strings.TrimSpace(pair), ":")
			if len(parts) == 2 {
				typeVariantPairs = append(typeVariantPairs, TypeVariantPair{
					DerivationType: strings.TrimSpace(parts[0]),
					Variant:        strings.TrimSpace(parts[1]),
				})
			}
		}
		params.TypeVariantPairs = typeVariantPairs
	}

	// Pagination
	if limitStr := query.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			params.Limit = &limit
		}
	}

	if offsetStr := query.Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			params.Offset = &offset
		}
	}

	return params
}

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

func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func (eds *EnhancedDerivedContentService) serveDemoPage(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Enhanced Derived Content Filtering Demo</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 1200px; margin: 0 auto; padding: 20px; }
        .section { margin-bottom: 30px; padding: 20px; border: 1px solid #ddd; border-radius: 5px; }
        .button { background: #007bff; color: white; padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer; margin: 5px; }
        .button:hover { background: #0056b3; }
        .result { background: #f8f9fa; padding: 15px; border-radius: 5px; margin: 10px 0; }
        .json { background: #2d3748; color: #e2e8f0; padding: 15px; border-radius: 5px; overflow-x: auto; }
        .filter-form { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 15px; margin: 15px 0; }
        .filter-group { display: flex; flex-direction: column; }
        .filter-group label { font-weight: bold; margin-bottom: 5px; }
        .filter-group input, .filter-group select { padding: 5px; border: 1px solid #ddd; border-radius: 3px; }
        .examples { background: #e7f3ff; padding: 15px; border-radius: 5px; margin: 10px 0; }
        .example-link { display: inline-block; margin: 5px; padding: 8px 12px; background: #17a2b8; color: white; text-decoration: none; border-radius: 3px; font-size: 12px; }
        .example-link:hover { background: #138496; }
    </style>
</head>
<body>
    <h1>Enhanced Derived Content Filtering Demo</h1>
    <p>This demo shows advanced filtering capabilities for derived content using derivation types and variants.</p>

    <div class="section">
        <h2>Setup Demo Data</h2>
        <p>Create sample content with various derived items for testing filters.</p>
        <button class="button" onclick="setupDemo()">Create Demo Data</button>
        <div id="setupResult"></div>
    </div>

    <div class="section">
        <h2>Filter Controls</h2>
        <div class="filter-form">
            <div class="filter-group">
                <label>Parent ID:</label>
                <input type="text" id="parentId" placeholder="Content UUID">
            </div>
            <div class="filter-group">
                <label>Derivation Type:</label>
                <select id="derivationType">
                    <option value="">All Types</option>
                    <option value="thumbnail">Thumbnail</option>
                    <option value="preview">Preview</option>
                    <option value="transcode">Transcode</option>
                </select>
            </div>
            <div class="filter-group">
                <label>Derivation Types (comma-separated):</label>
                <input type="text" id="derivationTypes" placeholder="thumbnail,preview">
            </div>
            <div class="filter-group">
                <label>Variant:</label>
                <input type="text" id="variant" placeholder="thumbnail_256">
            </div>
            <div class="filter-group">
                <label>Variants (comma-separated):</label>
                <input type="text" id="variants" placeholder="thumbnail_128,thumbnail_256">
            </div>
            <div class="filter-group">
                <label>Type:Variant Pairs:</label>
                <input type="text" id="typeVariantPairs" placeholder="thumbnail:thumbnail_256,preview:preview_web">
            </div>
            <div class="filter-group">
                <label>Limit:</label>
                <input type="number" id="limit" placeholder="10" min="1" max="100">
            </div>
            <div class="filter-group">
                <label>Offset:</label>
                <input type="number" id="offset" placeholder="0" min="0">
            </div>
        </div>
        <button class="button" onclick="filterDerivedContent()">Apply Filters</button>
        <button class="button" onclick="clearFilters()">Clear All</button>
    </div>

    <div class="section">
        <h2>Quick Filter Examples</h2>
        <div class="examples">
            <p><strong>Common filtering patterns:</strong></p>
            <a href="#" class="example-link" onclick="setExample('thumbnails')">All Thumbnails</a>
            <a href="#" class="example-link" onclick="setExample('large-thumbnails')">Large Thumbnails</a>
            <a href="#" class="example-link" onclick="setExample('previews')">All Previews</a>
            <a href="#" class="example-link" onclick="setExample('specific-variants')">Specific Variants</a>
            <a href="#" class="example-link" onclick="setExample('type-variant-pairs')">Type+Variant Pairs</a>
        </div>
    </div>

    <div class="section">
        <h2>Convenience Endpoints</h2>
        <button class="button" onclick="getThumbnailsBySize()">Get Thumbnails by Size</button>
        <button class="button" onclick="getRecentDerived()">Get Recent Derived (24h)</button>
        <button class="button" onclick="listAllContents()">List All Contents</button>
    </div>

    <div class="section">
        <h2>Results</h2>
        <div id="results"></div>
    </div>

    <script>
        let demoContentId = null;

        async function setupDemo() {
            try {
                const response = await fetch('/api/v1/demo/setup');
                const result = await response.json();

                if (response.ok) {
                    demoContentId = result.original_id;
                    document.getElementById('parentId').value = result.original_id;

                    displayResult('Demo Setup', result);
                    document.getElementById('setupResult').innerHTML =
                        '<div class="result">Demo data created! Original ID: ' + result.original_id +
                        '<br>Created ' + result.derived_count + ' derived items.</div>';
                } else {
                    displayError('Setup failed: ' + result.error.message);
                }
            } catch (error) {
                displayError('Setup error: ' + error.message);
            }
        }

        async function filterDerivedContent() {
            const params = new URLSearchParams();

            const parentId = document.getElementById('parentId').value.trim();
            if (!parentId) {
                alert('Parent ID is required');
                return;
            }
            params.set('parent_id', parentId);

            const derivationType = document.getElementById('derivationType').value;
            if (derivationType) params.set('derivation_type', derivationType);

            const derivationTypes = document.getElementById('derivationTypes').value.trim();
            if (derivationTypes) params.set('derivation_types', derivationTypes);

            const variant = document.getElementById('variant').value.trim();
            if (variant) params.set('variant', variant);

            const variants = document.getElementById('variants').value.trim();
            if (variants) params.set('variants', variants);

            const typeVariantPairs = document.getElementById('typeVariantPairs').value.trim();
            if (typeVariantPairs) params.set('type_variant_pairs', typeVariantPairs);

            const limit = document.getElementById('limit').value;
            if (limit) params.set('limit', limit);

            const offset = document.getElementById('offset').value;
            if (offset) params.set('offset', offset);

            try {
                const response = await fetch('/api/v1/derived-content/filter?' + params.toString());
                const result = await response.json();

                if (response.ok) {
                    displayResult('Filtered Results (' + result.count + ' items)', result);
                } else {
                    displayError('Filter failed: ' + result.error.message);
                }
            } catch (error) {
                displayError('Filter error: ' + error.message);
            }
        }

        function setExample(type) {
            clearFilters();

            if (!demoContentId) {
                alert('Please create demo data first');
                return;
            }

            document.getElementById('parentId').value = demoContentId;

            switch (type) {
                case 'thumbnails':
                    document.getElementById('derivationType').value = 'thumbnail';
                    break;
                case 'large-thumbnails':
                    document.getElementById('variants').value = 'thumbnail_256,thumbnail_512';
                    break;
                case 'previews':
                    document.getElementById('derivationType').value = 'preview';
                    break;
                case 'specific-variants':
                    document.getElementById('variants').value = 'thumbnail_256,preview_web';
                    break;
                case 'type-variant-pairs':
                    document.getElementById('typeVariantPairs').value = 'thumbnail:thumbnail_256,preview:preview_web';
                    break;
            }

            filterDerivedContent();
        }

        function clearFilters() {
            document.getElementById('derivationType').value = '';
            document.getElementById('derivationTypes').value = '';
            document.getElementById('variant').value = '';
            document.getElementById('variants').value = '';
            document.getElementById('typeVariantPairs').value = '';
            document.getElementById('limit').value = '';
            document.getElementById('offset').value = '';
        }

        async function getThumbnailsBySize() {
            if (!demoContentId) {
                alert('Please create demo data first');
                return;
            }

            try {
                const response = await fetch('/api/v1/derived-content/thumbnails?parent_id=' + demoContentId + '&sizes=128,256,512');
                const result = await response.json();

                if (response.ok) {
                    displayResult('Thumbnails by Size', result);
                } else {
                    displayError('Failed to get thumbnails: ' + result.error.message);
                }
            } catch (error) {
                displayError('Error: ' + error.message);
            }
        }

        async function getRecentDerived() {
            if (!demoContentId) {
                alert('Please create demo data first');
                return;
            }

            try {
                const response = await fetch('/api/v1/derived-content/recent?parent_id=' + demoContentId);
                const result = await response.json();

                if (response.ok) {
                    displayResult('Recent Derived Content', result);
                } else {
                    displayError('Failed to get recent derived: ' + result.error.message);
                }
            } catch (error) {
                displayError('Error: ' + error.message);
            }
        }

        async function listAllContents() {
            try {
                const response = await fetch('/api/v1/contents');
                const result = await response.json();

                if (response.ok) {
                    displayResult('All Contents', result);
                } else {
                    displayError('Failed to list contents: ' + result.error.message);
                }
            } catch (error) {
                displayError('Error: ' + error.message);
            }
        }

        function displayResult(title, data) {
            const container = document.getElementById('results');
            container.innerHTML =
                '<h3>' + title + '</h3>' +
                '<div class="json">' + JSON.stringify(data, null, 2) + '</div>';
        }

        function displayError(message) {
            const container = document.getElementById('results');
            container.innerHTML = '<div class="result" style="background: #f8d7da; color: #721c24;">' + message + '</div>';
        }
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func main() {
	fmt.Println("=== Enhanced Derived Content Filtering Demo ===")
	fmt.Println()
	fmt.Println("This demo shows advanced filtering capabilities for derived content.")

	// Create service
	service, err := NewEnhancedDerivedContentService()
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
	fmt.Println("Enhanced API Endpoints:")
	fmt.Println("  GET  /api/v1/derived-content/filter        - Advanced filtering")
	fmt.Println("  GET  /api/v1/derived-content/thumbnails    - Filter thumbnails by size")
	fmt.Println("  GET  /api/v1/derived-content/recent        - Get recent derived content")
	fmt.Println("  GET  /api/v1/demo/setup                    - Create demo data")
	fmt.Println()
	fmt.Println("Example filter URLs:")
	fmt.Println("  ?parent_id=uuid&derivation_type=thumbnail")
	fmt.Println("  ?parent_id=uuid&variants=thumbnail_256,thumbnail_512")
	fmt.Println("  ?parent_id=uuid&type_variant_pairs=thumbnail:thumbnail_256,preview:preview_web")
	fmt.Println()

	log.Fatal(server.ListenAndServe())
}