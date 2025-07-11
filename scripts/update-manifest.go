package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	googleFontsRepoURL = "https://api.github.com/repos/google/fonts/git/trees/main?recursive=1"
	manifestPath       = "sources/google-fonts.json"
	rawContentBaseURL  = "https://raw.githubusercontent.com/google/fonts/main/"
)

type GitHubTreeResponse struct {
	Sha  string `json:"sha"`
	URL  string `json:"url"`
	Tree []struct {
		Path string `json:"path"`
		Mode string `json:"mode"`
		Type string `json:"type"`
		Sha  string `json:"sha"`
		URL  string `json:"url"`
		Size int    `json:"size,omitempty"`
	} `json:"tree"`
}

type FontManifest struct {
	Version     string    `json:"version"`
	LastUpdated time.Time `json:"last_updated"`
	Sources     struct {
		GoogleFonts struct {
			Name        string    `json:"name"`
			Description string    `json:"description"`
			URL         string    `json:"url"`
			LastUpdated time.Time `json:"last_updated"`
			Fonts       map[string]struct {
				Name         string            `json:"name"`
				License      string            `json:"license"`
				Variants     []string          `json:"variants"`
				Subsets      []string          `json:"subsets"`
				Version      string            `json:"version"`
				Description  string            `json:"description"`
				LastModified time.Time         `json:"last_modified"`
				MetadataURL  string            `json:"metadata_url"`
				SourceURL    string            `json:"source_url"`
				Categories   []string          `json:"categories"`
				Popularity   int               `json:"popularity"`
				Files        map[string]string `json:"files"`
			} `json:"fonts"`
		} `json:"google-fonts"`
	} `json:"sources"`
}

type FontMetadata struct {
	Name         string   `json:"name"`
	Designer     string   `json:"designer"`
	License      string   `json:"license"`
	LicenseURL   string   `json:"license_url"`
	Category     string   `json:"category"`
	Version      string   `json:"version"`
	Description  string   `json:"description"`
	Subsets      []string `json:"subsets"`
	LastModified string   `json:"last_modified"`
}

// FontProcessingResult represents the result of processing a font family
type FontProcessingResult struct {
	FontID    string
	FontEntry struct {
		Name         string            `json:"name"`
		License      string            `json:"license"`
		Variants     []string          `json:"variants"`
		Subsets      []string          `json:"subsets"`
		Version      string            `json:"version"`
		Description  string            `json:"description"`
		LastModified time.Time         `json:"last_modified"`
		MetadataURL  string            `json:"metadata_url"`
		SourceURL    string            `json:"source_url"`
		Categories   []string          `json:"categories"`
		Popularity   int               `json:"popularity"`
		Files        map[string]string `json:"files"`
	}
	Error error
}

func main() {
	// Create manifest structure
	manifest := FontManifest{
		Version:     "1.0",
		LastUpdated: time.Now(),
	}

	manifest.Sources.GoogleFonts = struct {
		Name        string    `json:"name"`
		Description string    `json:"description"`
		URL         string    `json:"url"`
		LastUpdated time.Time `json:"last_updated"`
		Fonts       map[string]struct {
			Name         string            `json:"name"`
			License      string            `json:"license"`
			Variants     []string          `json:"variants"`
			Subsets      []string          `json:"subsets"`
			Version      string            `json:"version"`
			Description  string            `json:"description"`
			LastModified time.Time         `json:"last_modified"`
			MetadataURL  string            `json:"metadata_url"`
			SourceURL    string            `json:"source_url"`
			Categories   []string          `json:"categories"`
			Popularity   int               `json:"popularity"`
			Files        map[string]string `json:"files"`
		} `json:"fonts"`
	}{
		Name:        "Google Fonts",
		Description: "Open source fonts from Google",
		URL:         "https://fonts.google.com",
		LastUpdated: time.Now(),
		Fonts: make(map[string]struct {
			Name         string            `json:"name"`
			License      string            `json:"license"`
			Variants     []string          `json:"variants"`
			Subsets      []string          `json:"subsets"`
			Version      string            `json:"version"`
			Description  string            `json:"description"`
			LastModified time.Time         `json:"last_modified"`
			MetadataURL  string            `json:"metadata_url"`
			SourceURL    string            `json:"source_url"`
			Categories   []string          `json:"categories"`
			Popularity   int               `json:"popularity"`
			Files        map[string]string `json:"files"`
		}),
	}

	// Fetch repository tree
	resp, err := http.Get(googleFontsRepoURL)
	if err != nil {
		fmt.Printf("Error fetching repository tree: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error: GitHub API returned status %d\n", resp.StatusCode)
		os.Exit(1)
	}

	var treeResp GitHubTreeResponse
	if err := json.NewDecoder(resp.Body).Decode(&treeResp); err != nil {
		fmt.Printf("Error decoding repository tree: %v\n", err)
		os.Exit(1)
	}

	// Process each file in the tree
	fontFamilies := make(map[string]map[string]string) // family -> variant -> path
	metadataFiles := make(map[string]string)           // family -> metadata path

	for _, item := range treeResp.Tree {
		if item.Type != "blob" {
			continue
		}

		path := item.Path
		if strings.HasSuffix(path, ".ttf") {
			// Extract font family and variant from path
			parts := strings.Split(path, "/")
			if len(parts) < 3 {
				continue
			}

			// Skip files in test directories
			if strings.Contains(path, "/test/") || strings.Contains(path, "beta") || strings.Contains(path, "alpha") {
				fmt.Printf("Skipping experimental/test font: %s\n", path)
				continue
			}

			family := parts[1]
			filename := parts[len(parts)-1]
			variant := strings.TrimSuffix(filename, ".ttf")

			if fontFamilies[family] == nil {
				fontFamilies[family] = make(map[string]string)
			}
			fontFamilies[family][variant] = path
		} else if strings.HasSuffix(path, "METADATA.pb") {
			// Store metadata file path
			parts := strings.Split(path, "/")
			if len(parts) < 2 {
				continue
			}
			family := parts[1]
			metadataFiles[family] = path
		}
	}

	// Create a worker pool
	numWorkers := 10 // Adjust based on your system's capabilities
	jobs := make(chan struct {
		family       string
		variants     map[string]string
		metadataPath string
	}, len(fontFamilies))
	results := make(chan FontProcessingResult, len(fontFamilies))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				result := processFontFamily(job.family, job.variants, job.metadataPath)
				results <- result
			}
		}()
	}

	// Send jobs to workers
	go func() {
		for family, variants := range fontFamilies {
			jobs <- struct {
				family       string
				variants     map[string]string
				metadataPath string
			}{
				family:       family,
				variants:     variants,
				metadataPath: metadataFiles[family],
			}
		}
		close(jobs)
	}()

	// Close results channel when all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Process results
	for result := range results {
		if result.Error != nil {
			fmt.Printf("Warning: %v\n", result.Error)
			continue
		}
		manifest.Sources.GoogleFonts.Fonts[result.FontID] = result.FontEntry
	}

	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
		fmt.Printf("Error creating directory: %v\n", err)
		os.Exit(1)
	}

	// Write the manifest file
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling manifest: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
		fmt.Printf("Error writing manifest: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully updated manifest at %s\n", manifestPath)
}

func fetchMetadata(url string) (FontMetadata, error) {
	resp, err := http.Get(url)
	if err != nil {
		return FontMetadata{}, fmt.Errorf("failed to fetch metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return FontMetadata{}, fmt.Errorf("failed to fetch metadata: HTTP %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return FontMetadata{}, fmt.Errorf("failed to read metadata: %w", err)
	}

	// Parse METADATA.pb file
	metadata := FontMetadata{}
	contentStr := string(content)

	// Extract name
	if name := extractMetadataField(contentStr, "name:"); name != "" {
		metadata.Name = name
	}

	// Extract designer
	if designer := extractMetadataField(contentStr, "designer:"); designer != "" {
		metadata.Designer = designer
	}

	// Extract license
	if license := extractMetadataField(contentStr, "license:"); license != "" {
		metadata.License = license
	}

	// Extract license URL
	if licenseURL := extractMetadataField(contentStr, "license_url:"); licenseURL != "" {
		metadata.LicenseURL = licenseURL
	}

	// Extract category
	if category := extractMetadataField(contentStr, "category:"); category != "" {
		metadata.Category = category
	}

	// Extract version
	if version := extractMetadataField(contentStr, "version:"); version != "" {
		metadata.Version = version
	}

	// Extract description
	if description := extractMetadataField(contentStr, "description:"); description != "" {
		metadata.Description = description
	}

	// Extract subsets
	subsetsRe := regexp.MustCompile(`subsets:\s*\[(.*?)\]`)
	if matches := subsetsRe.FindStringSubmatch(contentStr); len(matches) > 1 {
		subsetsStr := matches[1]
		subsets := strings.Split(subsetsStr, ",")
		for i, subset := range subsets {
			subsets[i] = strings.TrimSpace(strings.Trim(subset, `"`))
		}
		metadata.Subsets = subsets
	}

	// Extract last modified
	if lastModified := extractMetadataField(contentStr, "lastModified:"); lastModified != "" {
		metadata.LastModified = lastModified
	}

	return metadata, nil
}

func extractMetadataField(content, field string) string {
	re := regexp.MustCompile(field + `\s*"([^"]*)"`)
	if matches := re.FindStringSubmatch(content); len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func getKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func formatFontName(name string) string {
	// First, split by common separators (hyphens, underscores)
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '-' || r == '_'
	})

	// For each part, handle camelCase
	var result []string
	for _, part := range parts {
		// Split by camelCase
		subParts := regexp.MustCompile(`([A-Z][a-z]+|[0-9]+|[a-z]+)`).FindAllString(part, -1)
		if len(subParts) > 0 {
			// Capitalize first letter of each subpart
			for i, subPart := range subParts {
				if len(subPart) > 0 {
					subParts[i] = strings.ToUpper(subPart[:1]) + strings.ToLower(subPart[1:])
				}
			}
			result = append(result, strings.Join(subParts, " "))
		} else {
			// If no camelCase found, just capitalize the first letter
			if len(part) > 0 {
				result = append(result, strings.ToUpper(part[:1])+strings.ToLower(part[1:]))
			}
		}
	}

	// Join all parts with spaces
	return strings.Join(result, " ")
}

// processFontFamily processes a single font family and returns the result
func processFontFamily(family string, variants map[string]string, metadataPath string) FontProcessingResult {
	result := FontProcessingResult{
		FontID: strings.ToLower(strings.ReplaceAll(family, " ", "")),
	}

	// Create files map
	files := make(map[string]string)
	for variant, path := range variants {
		files[variant] = rawContentBaseURL + path
	}

	// Get metadata
	metadata, err := fetchMetadata(rawContentBaseURL + metadataPath)
	if err != nil {
		// If metadata fetch fails, use directory name as fallback
		fmt.Printf("Note: Using directory name for %s (metadata not available)\n", family)
		return FontProcessingResult{
			FontID: result.FontID,
			FontEntry: struct {
				Name         string            `json:"name"`
				License      string            `json:"license"`
				Variants     []string          `json:"variants"`
				Subsets      []string          `json:"subsets"`
				Version      string            `json:"version"`
				Description  string            `json:"description"`
				LastModified time.Time         `json:"last_modified"`
				MetadataURL  string            `json:"metadata_url"`
				SourceURL    string            `json:"source_url"`
				Categories   []string          `json:"categories"`
				Popularity   int               `json:"popularity"`
				Files        map[string]string `json:"files"`
			}{
				Name:         formatFontName(family),
				License:      "OFL", // Default license
				Variants:     getKeys(variants),
				Subsets:      nil,
				Version:      "",
				Description:  "",
				LastModified: time.Now(),
				MetadataURL:  rawContentBaseURL + metadataPath,
				SourceURL:    fmt.Sprintf("https://fonts.google.com/specimen/%s", strings.ReplaceAll(family, " ", "+")),
				Categories:   []string{"Other"}, // Default category
				Popularity:   0,
				Files:        files,
			},
		}
	}

	// Parse last modified time
	lastModified := time.Now()
	if metadata.LastModified != "" {
		if t, err := time.Parse("2006-01-02", metadata.LastModified); err == nil {
			lastModified = t
		}
	}

	// Determine license
	license := "OFL" // Default to OFL
	if metadata.License != "" {
		license = metadata.License
	}

	// Determine categories
	categories := []string{}
	if metadata.Category != "" {
		// Format category to match the old format
		category := strings.Title(strings.ToLower(strings.ReplaceAll(metadata.Category, "_", " ")))
		categories = append(categories, category)
	}

	result.FontEntry = struct {
		Name         string            `json:"name"`
		License      string            `json:"license"`
		Variants     []string          `json:"variants"`
		Subsets      []string          `json:"subsets"`
		Version      string            `json:"version"`
		Description  string            `json:"description"`
		LastModified time.Time         `json:"last_modified"`
		MetadataURL  string            `json:"metadata_url"`
		SourceURL    string            `json:"source_url"`
		Categories   []string          `json:"categories"`
		Popularity   int               `json:"popularity"`
		Files        map[string]string `json:"files"`
	}{
		Name:         metadata.Name,
		License:      license,
		Variants:     getKeys(variants),
		Subsets:      metadata.Subsets,
		Version:      metadata.Version,
		Description:  metadata.Description,
		LastModified: lastModified,
		MetadataURL:  rawContentBaseURL + metadataPath,
		SourceURL:    fmt.Sprintf("https://fonts.google.com/specimen/%s", strings.ReplaceAll(family, " ", "+")),
		Categories:   categories,
		Popularity:   0, // TODO: Implement popularity tracking
		Files:        files,
	}

	return result
}
