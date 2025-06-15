package repo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// GetManifest returns the font manifest, loading it if necessary
func GetManifest(cache *Cache, progress ProgressCallback) (*FontManifest, error) {
	if cache == nil {
		var err error
		cache, err = NewCache()
		if err != nil {
			fmt.Printf("Error creating cache: %v\n", err)
			return nil, err
		}
	}

	// Get the sources directory
	sourcesDir, err := getSourcesDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get sources directory: %w", err)
	}

	// Ensure the sources directory exists
	_, err = ensureSourcesDir()
	if err != nil {
		return nil, fmt.Errorf("failed to ensure sources directory: %w", err)
	}

	// Check if manifest exists in sources directory
	manifestPath := filepath.Join(sourcesDir, manifestFile)
	info, err := os.Stat(manifestPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("error checking manifest: %w", err)
		}
		// Manifest doesn't exist, we'll download it
		if progress != nil {
			progress(0, 1, "Downloading manifest...")
		}
	} else {
		// Check if manifest is recent (less than 24 hours old)
		if time.Since(info.ModTime()) < updateInterval {
			// Load existing manifest
			data, err := os.ReadFile(manifestPath)
			if err != nil {
				return nil, fmt.Errorf("error reading manifest: %w", err)
			}

			var manifest FontManifest
			if err := json.Unmarshal(data, &manifest); err != nil {
				return nil, fmt.Errorf("error unmarshaling manifest: %w", err)
			}

			if progress != nil {
				progress(1, 1, "Loaded existing manifest")
			}
			return &manifest, nil
		}
		if progress != nil {
			progress(0, 1, "Manifest is outdated, downloading new version...")
		}
	}

	// Download manifest from repository
	resp, err := http.Get(fontManifestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download manifest: HTTP %d", resp.StatusCode)
	}

	// Parse manifest
	var manifest FontManifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("error parsing manifest: %w", err)
	}

	// Save manifest
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error marshaling manifest: %w", err)
	}

	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return nil, fmt.Errorf("error writing manifest: %w", err)
	}

	if progress != nil {
		progress(1, 1, "Downloaded and saved new manifest")
	}
	return &manifest, nil
}
