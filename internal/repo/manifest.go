package repo

import (
	"fmt"
	"sync"

	"fontget/internal/config"
)

type cachedManifestSlot struct {
	dir        string
	full       *FontManifest
	matching   *FontManifest
	generation uint64
}

var (
	manifestMemoMu sync.Mutex
	manifestMemo   cachedManifestSlot
)

// InvalidateCachedManifests drops in-process catalog memos. Called when source
// cache files change (install/update) so the next load sees new JSON.
func InvalidateCachedManifests() {
	manifestMemoMu.Lock()
	manifestMemo.full = nil
	manifestMemo.matching = nil
	manifestMemo.dir = ""
	manifestMemo.generation++
	manifestMemoMu.Unlock()
}

// GetManifest returns the font manifest, loading it if necessary
func GetManifest(cache *Cache, progress ProgressCallback) (*FontManifest, error) {
	return GetManifestWithRefresh(cache, progress, false)
}

// GetCachedManifest returns the full font manifest from on-disk source caches
// (including variant download URLs). Results are memoized in-process.
func GetCachedManifest() (*FontManifest, error) {
	return getCachedManifest(false)
}

// getCachedManifestMatching returns a catalog with id/name/license/categories/source
// only (no variant file URL maps). Use this for list/matching; install/download
// must keep using GetCachedManifest so Files/VariantFiles stay populated.
func getCachedManifestMatching() (*FontManifest, error) {
	return getCachedManifest(true)
}

func getCachedManifest(matchingOnly bool) (*FontManifest, error) {
	dir, err := getFontGetDir()
	if err != nil {
		return nil, err
	}

	manifestMemoMu.Lock()
	if manifestMemo.dir == dir {
		if matchingOnly && manifestMemo.matching != nil {
			m := manifestMemo.matching
			manifestMemoMu.Unlock()
			return m, nil
		}
		if !matchingOnly && manifestMemo.full != nil {
			m := manifestMemo.full
			manifestMemoMu.Unlock()
			return m, nil
		}
	}
	gen := manifestMemo.generation
	manifestMemoMu.Unlock()

	cfg, err := config.LoadManifest()
	if err != nil {
		return nil, fmt.Errorf("failed to load manifest configuration: %w", err)
	}

	var fontManifest *FontManifest
	if matchingOnly {
		fontManifest, err = loadAllSourcesFromCacheMatching(cfg)
	} else {
		fontManifest, err = loadAllSourcesFromCacheOnly(cfg)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load sources from cache: %w", err)
	}

	manifestMemoMu.Lock()
	if gen == manifestMemo.generation {
		manifestMemo.dir = dir
		if matchingOnly {
			manifestMemo.matching = fontManifest
		} else {
			manifestMemo.full = fontManifest
		}
	}
	manifestMemoMu.Unlock()
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
