package urlstrategy

import (
	"fmt"
)

// URLStrategyType represents the type of URL strategy
type URLStrategyType string

const (
	// CDN strategy for maximum performance with direct CDN URLs
	StrategyTypeCDN URLStrategyType = "cdn"

	// Content-based strategy for application-routed URLs
	StrategyTypeContentBased URLStrategyType = "content-based"

	// Storage-delegated strategy for backward compatibility
	StrategyTypeStorageDelegated URLStrategyType = "storage-delegated"
)

// Config holds configuration for URL strategy creation
type Config struct {
	Type          URLStrategyType
	CDNBaseURL    string // For CDN strategy downloads
	UploadBaseURL string // For CDN strategy uploads
	APIBaseURL    string // For content-based strategy
	BlobStores    map[string]BlobStore // For storage-delegated strategy
}

// NewURLStrategy creates a URL strategy based on the configuration
func NewURLStrategy(config Config) (URLStrategy, error) {
	switch config.Type {
	case StrategyTypeCDN:
		if config.CDNBaseURL == "" {
			return nil, fmt.Errorf("CDN base URL is required for CDN strategy")
		}
		if config.UploadBaseURL != "" {
			return NewCDNStrategyWithUpload(config.CDNBaseURL, config.UploadBaseURL), nil
		}
		return NewCDNStrategy(config.CDNBaseURL), nil

	case StrategyTypeContentBased:
		if config.APIBaseURL == "" {
			return nil, fmt.Errorf("API base URL is required for content-based strategy")
		}
		return NewContentBasedStrategy(config.APIBaseURL), nil

	case StrategyTypeStorageDelegated:
		if config.BlobStores == nil || len(config.BlobStores) == 0 {
			return nil, fmt.Errorf("blob stores are required for storage-delegated strategy")
		}
		return NewStorageDelegatedStrategy(config.BlobStores), nil

	default:
		return nil, fmt.Errorf("unknown URL strategy type: %s", config.Type)
	}
}

// NewDefaultStrategy creates a sensible default URL strategy
// Uses content-based strategy as the default for development/testing
func NewDefaultStrategy(apiBaseURL string) URLStrategy {
	if apiBaseURL == "" {
		apiBaseURL = "/api/v1" // Default API base URL
	}
	return NewContentBasedStrategy(apiBaseURL)
}

// NewRecommendedStrategy creates the recommended URL strategy based on environment
func NewRecommendedStrategy(environment string, cdnURL string, apiURL string) URLStrategy {
	return NewRecommendedStrategyWithUpload(environment, cdnURL, "", apiURL)
}

// NewRecommendedStrategyWithUpload creates the recommended URL strategy with upload URL support
func NewRecommendedStrategyWithUpload(environment string, cdnURL string, uploadURL string, apiURL string) URLStrategy {
	switch environment {
	case "production":
		if cdnURL != "" {
			// Production with CDN - maximum performance
			if uploadURL != "" {
				return NewCDNStrategyWithUpload(cdnURL, uploadURL)
			}
			return NewCDNStrategy(cdnURL)
		}
		fallthrough
	case "staging":
		// Staging or production without CDN - use content-based
		if apiURL == "" {
			apiURL = "/api/v1"
		}
		return NewContentBasedStrategy(apiURL)
	default:
		// Development - use content-based for easier debugging
		if apiURL == "" {
			apiURL = "/api/v1"
		}
		return NewContentBasedStrategy(apiURL)
	}
}