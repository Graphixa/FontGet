package repo

import (
	"fmt"

	"fontget/internal/config"
)

// GetManifest returns the font manifest, loading it if necessary
func GetManifest(cache *Cache, progress ProgressCallback) (*FontManifest, error) {
	return GetManifestWithRefresh(cache, progress, false)
}

// GetCachedManifest returns the font manifest from cache only (no refresh)
func GetCachedManifest() (*FontManifest, error) {
	// Load sources configuration
	sourcesConfig, err := config.LoadSourcesConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load sources configuration: %w", err)
	}

	// Load all enabled sources from cache only (no refresh)
	manifest, err := loadAllSourcesFromCacheOnly(sourcesConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load sources from cache: %w", err)
	}

	return manifest, nil
}

// GetManifestWithRefresh returns the font manifest with optional cache refresh
func GetManifestWithRefresh(cache *Cache, progress ProgressCallback, forceRefresh bool) (*FontManifest, error) {
	// Load sources configuration
	sourcesConfig, err := config.LoadSourcesConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load sources configuration: %w", err)
	}

	// Load all enabled sources from FontGet-Sources
	manifest, err := loadAllSourcesWithCache(sourcesConfig, progress, forceRefresh)
	if err != nil {
		return nil, fmt.Errorf("failed to load sources: %w", err)
	}

	return manifest, nil
}
