package repo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	// Directory structure
	metadataDir    = "metadata"
	licensesDir    = "licenses"
	manifestFile   = "google-fonts.json"
	updateInterval = 24 * time.Hour

	// Font manifest URL in our repository
	fontManifestURL = "https://raw.githubusercontent.com/graphixa/FontGet/main/sources/google-fonts.json"
	googleFontsCSS  = "https://fonts.googleapis.com/css2?family=%s&display=swap"
)

// GoogleFontsResponse represents the response from Google Fonts API
type GoogleFontsResponse struct {
	Kind  string `json:"kind"`
	Items []struct {
		Family       string            `json:"family"`
		Variants     []string          `json:"variants"`
		Subsets      []string          `json:"subsets"`
		Version      string            `json:"version"`
		LastModified string            `json:"lastModified"`
		Files        map[string]string `json:"files"`
		Category     string            `json:"category"`
		Kind         string            `json:"kind"`
	} `json:"items"`
}

// getSourcesDir returns the path to the user's ~/.fontget/sources directory
func getSourcesDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(home, ".fontget", "sources"), nil
}

// ProgressCallback is a function type for reporting progress
type ProgressCallback func(current, total int, message string)

// FontManifest represents the combined manifest of all fonts
type FontManifest struct {
	Version     string                `json:"version"`
	LastUpdated time.Time             `json:"last_updated"`
	Sources     map[string]SourceInfo `json:"sources"`
}

// SourceInfo represents information about a font source
type SourceInfo struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	URL         string              `json:"url"`
	LastUpdated time.Time           `json:"last_updated"`
	Fonts       map[string]FontInfo `json:"fonts"`
}

// FontInfo represents detailed information about a font
type FontInfo struct {
	Name         string            `json:"name"`
	License      string            `json:"license"`
	Variants     []string          `json:"variants"`
	Subsets      []string          `json:"subsets"`
	Version      string            `json:"version"`
	Description  string            `json:"description"`
	LastModified time.Time         `json:"last_modified"`
	MetadataURL  string            `json:"metadata_url"`
	SourceURL    string            `json:"source_url"`
	Categories   []string          `json:"categories,omitempty"`
	Tags         []string          `json:"tags,omitempty"`
	Popularity   int               `json:"popularity,omitempty"`
	Files        map[string]string `json:"files,omitempty"`
}

// ensureSourcesDir ensures the sources directory exists
func ensureSourcesDir() (string, error) {
	sourcesDir, err := getSourcesDir()
	if err != nil {
		return "", err
	}
	dirs := []string{
		sourcesDir,
		filepath.Join(sourcesDir, metadataDir),
		filepath.Join(sourcesDir, licensesDir),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return sourcesDir, nil
}

// needsUpdate checks if the manifest needs to be updated
func needsUpdate() (bool, error) {
	sourcesDir, err := getSourcesDir()
	if err != nil {
		return false, err
	}
	manifestPath := filepath.Join(sourcesDir, manifestFile)
	info, err := os.Stat(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, fmt.Errorf("failed to stat manifest: %w", err)
	}

	return time.Since(info.ModTime()) > updateInterval, nil
}

func updateManifest() error {
	// Get sources directory
	sourcesDir, err := ensureSourcesDir()
	if err != nil {
		return fmt.Errorf("failed to get sources directory: %v", err)
	}

	// Check if manifest needs update
	manifestPath := filepath.Join(sourcesDir, manifestFile)
	if info, err := os.Stat(manifestPath); err == nil {
		// Check if file is less than 24 hours old
		if time.Since(info.ModTime()) < updateInterval {
			// Read cached manifest
			cachedData, err := os.ReadFile(manifestPath)
			if err == nil {
				var cachedManifest FontManifest
				if err := json.Unmarshal(cachedData, &cachedManifest); err == nil {
					// Fetch latest manifest to compare
					resp, err := http.Get(fontManifestURL)
					if err == nil {
						defer resp.Body.Close()
						if resp.StatusCode == http.StatusOK {
							body, err := io.ReadAll(resp.Body)
							if err == nil {
								var latestManifest FontManifest
								if err := json.Unmarshal(body, &latestManifest); err == nil {
									// Compare manifests
									if cachedManifest.LastUpdated.Equal(latestManifest.LastUpdated) {
										return nil
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Fetch latest manifest
	resp, err := http.Get(fontManifestURL)
	if err != nil {
		return fmt.Errorf("failed to fetch manifest: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch manifest: HTTP %d", resp.StatusCode)
	}

	// Read and parse manifest
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %v", err)
	}

	// Parse the manifest
	var manifest FontManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return fmt.Errorf("failed to parse manifest: %v", err)
	}

	// Save manifest
	processedData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %v", err)
	}

	if err := os.WriteFile(manifestPath, processedData, 0644); err != nil {
		return fmt.Errorf("failed to save manifest: %v", err)
	}

	return nil
}

// GetManifest returns the current font manifest, updating if necessary
func GetManifest(progress ProgressCallback) (*FontManifest, error) {
	// Get sources directory
	sourcesDir, err := ensureSourcesDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get sources directory: %v", err)
	}

	// Check if manifest needs update
	needsUpdate, err := needsUpdate()
	if err != nil {
		return nil, fmt.Errorf("failed to check manifest update: %v", err)
	}

	if needsUpdate {
		if progress != nil {
			progress(0, 1, "Updating font catalog...")
		}
		if err := updateManifest(); err != nil {
			return nil, fmt.Errorf("failed to update manifest: %v", err)
		}
		if progress != nil {
			progress(1, 1, "Font catalog updated")
		}
	}

	// Read manifest
	manifestPath := filepath.Join(sourcesDir, manifestFile)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %v", err)
	}

	var manifest FontManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %v", err)
	}

	return &manifest, nil
}

// GetFontInfo retrieves information about a specific font
func GetFontInfo(fontID string) (*FontInfo, error) {
	manifest, err := GetManifest(nil)
	if err != nil {
		return nil, err
	}

	// Check each source
	for _, source := range manifest.Sources {
		if info, ok := source.Fonts[fontID]; ok {
			return &info, nil
		}
	}

	return nil, fmt.Errorf("font not found: %s", fontID)
}

// GetFontFiles returns the font files for a specific font family
func GetFontFiles(fontFamily string) (map[string]string, error) {
	url := fmt.Sprintf(googleFontsCSS, strings.ReplaceAll(fontFamily, " ", "+"))
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch CSS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch CSS: %s", resp.Status)
	}

	css, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read CSS: %w", err)
	}

	// Extract font URLs from CSS
	urlRegex := regexp.MustCompile(`url\((https://[^)]+)\)`)
	matches := urlRegex.FindAllStringSubmatch(string(css), -1)

	files := make(map[string]string)
	for _, match := range matches {
		if len(match) > 1 {
			url := match[1]
			// Extract variant from URL
			variant := "regular"
			if strings.Contains(url, "italic") {
				variant = "italic"
			} else if strings.Contains(url, "wght@") {
				weight := regexp.MustCompile(`wght@(\d+)`).FindStringSubmatch(url)
				if len(weight) > 1 {
					variant = weight[1]
				}
			}
			files[variant] = url
		}
	}

	return files, nil
}

// Repository represents a font repository
type Repository struct {
	manifest *FontManifest
}

// GetRepository returns a new repository instance
func GetRepository() (*Repository, error) {
	manifest, err := GetManifest(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest: %w", err)
	}
	return &Repository{manifest: manifest}, nil
}

// GetManifest returns the current font manifest
func (r *Repository) GetManifest() (*FontManifest, error) {
	return r.manifest, nil
}

// SearchFonts searches for fonts matching the query
func (r *Repository) SearchFonts(query string, category string) ([]SearchResult, error) {
	var results []SearchResult

	// Search through each source
	for sourceID, source := range r.manifest.Sources {
		for id, font := range source.Fonts {
			// Check both the font name and ID
			fontName := strings.ToLower(font.Name)
			fontID := strings.ToLower(id)
			query = strings.ToLower(query)

			// Check for matches
			if strings.Contains(fontName, query) || strings.Contains(fontID, query) {
				// If category filter is specified, check if font matches
				if category != "" {
					found := false
					for _, cat := range font.Categories {
						if strings.EqualFold(cat, category) {
							found = true
							break
						}
					}
					if !found {
						continue
					}
				}

				result := createSearchResult(id, font, sourceID, source.Name)
				results = append(results, result)
			}
		}
	}

	return results, nil
}
