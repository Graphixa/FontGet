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
	// Load manifest configuration
	manifest, err := config.LoadManifest()
	if err != nil {
		return nil, fmt.Errorf("failed to load manifest configuration: %w", err)
	}

	// Load all enabled sources from cache only (no refresh)
	fontManifest, err := loadAllSourcesFromCacheOnly(manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to load sources from cache: %w", err)
	}

	return fontManifest, nil
}

// GetManifestWithRefresh returns the font manifest with optional cache refresh
func GetManifestWithRefresh(cache *Cache, progress ProgressCallback, forceRefresh bool) (*FontManifest, error) {
	// Load manifest configuration
	manifest, err := config.LoadManifest()
	if err != nil {
		return nil, fmt.Errorf("failed to load manifest configuration: %w", err)
	}

	// Load all enabled sources from FontGet-Sources
	fontManifest, err := loadAllSourcesWithCache(manifest, progress, forceRefresh)
	if err != nil {
		return nil, fmt.Errorf("failed to load sources: %w", err)
	}

	return fontManifest, nil
}
