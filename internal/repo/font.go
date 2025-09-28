package repo

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FetchURLContent fetches content from a URL with cross-platform compatibility
func FetchURLContent(url string) (string, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Make request
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch content: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("content not found (HTTP %d)", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read content: %w", err)
	}

	return string(body), nil
}

// FontFileInfo represents metadata about a font file from the Google Fonts repository
type FontFileInfo struct {
	Name        string            `json:"name"`
	ID          string            `json:"id"`
	Source      string            `json:"source"`
	License     string            `json:"license"`
	Category    string            `json:"category"`
	Variants    []string          `json:"variants"`
	Subsets     []string          `json:"subsets"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Files       map[string]string `json:"files"`
}

// FontRepository handles interaction with the Google Fonts repository
type FontRepository struct {
	client *http.Client
}

// NewFontRepository creates a new FontRepository instance
func NewFontRepository() *FontRepository {
	return &FontRepository{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetFontInfo retrieves information about a font from the manifest
func (r *FontRepository) GetFontInfo(fontName string) (*FontFileInfo, error) {
	// Get manifest
	manifest, err := GetManifest(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest: %w", err)
	}

	// Normalize font name for comparison
	fontName = strings.ToLower(fontName)

	// Look for font in all sources
	for _, source := range manifest.Sources {
		for id, font := range source.Fonts {
			// Check both the font name and ID
			if strings.ToLower(font.Name) == fontName || strings.ToLower(id) == fontName {
				// Get the first category if available
				category := ""
				if len(font.Categories) > 0 {
					category = font.Categories[0]
				}
				return &FontFileInfo{
					Name:     font.Name,
					ID:       id,
					Source:   source.Name,
					Category: category,
					Files:    font.Files,
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("font not found: %s", fontName)
}

// DownloadFont downloads a font file from the repository
func (r *FontRepository) DownloadFont(filePath string) (io.ReadCloser, error) {
	// Get manifest
	manifest, err := GetManifest(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest: %w", err)
	}

	// Look for font in all sources
	for _, source := range manifest.Sources {
		for _, font := range source.Fonts {
			for _, url := range font.Files {
				if strings.HasSuffix(url, filePath) {
					resp, err := r.client.Get(url)
					if err != nil {
						return nil, fmt.Errorf("failed to download font file: %w", err)
					}

					if resp.StatusCode != http.StatusOK {
						resp.Body.Close()
						return nil, fmt.Errorf("failed to download font file: status code %d", resp.StatusCode)
					}

					return resp.Body, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("font file not found: %s", filePath)
}

// Font represents a font file from the Google Fonts repository
type FontFile struct {
	Name        string
	Variant     string
	Path        string
	SHA         string
	DownloadURL string
}

// DownloadFont downloads a font file and verifies its SHA-256 hash if available
func DownloadFont(font *FontFile, targetDir string) (string, error) {
	// Create target directory if it doesn't exist
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create target directory: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("GET", font.DownloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "fontget-cli")

	// Download file
	client := &http.Client{
		Timeout: 5 * time.Minute, // Increased timeout for large archive files
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download font: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, font.DownloadURL)
	}

	// Create target file
	targetPath := filepath.Join(targetDir, font.Path)
	file, err := os.Create(targetPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// If we have a SHA hash, verify it
	if font.SHA != "" {
		// Create SHA-256 hash
		hash := sha256.New()
		tee := io.TeeReader(resp.Body, hash)

		// Copy file content
		if _, err := io.Copy(file, tee); err != nil {
			return "", fmt.Errorf("failed to write file: %w", err)
		}

		// Calculate SHA-256
		calculatedHash := hex.EncodeToString(hash.Sum(nil))
		if calculatedHash != font.SHA {
			os.Remove(targetPath) // Clean up the file if hash doesn't match
			return "", fmt.Errorf("SHA-256 verification failed: expected %s, got %s", font.SHA, calculatedHash)
		}
	} else {
		// Just copy the file content if we don't have a SHA hash
		if _, err := io.Copy(file, resp.Body); err != nil {
			return "", fmt.Errorf("failed to write file: %w", err)
		}
	}

	return targetPath, nil
}

// DownloadAndExtractFont downloads a font file (which may be an archive) and extracts it if needed
func DownloadAndExtractFont(font *FontFile, targetDir string) ([]string, error) {

	// Create target directory if it doesn't exist
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create target directory: %w", err)
	}

	// Download the file first
	downloadedPath, err := DownloadFont(font, targetDir)
	if err != nil {
		return nil, err
	}

	// Check if the downloaded file is an archive
	archiveType := DetectArchiveType(downloadedPath)

	if archiveType == ArchiveTypeUnknown {
		// Not an archive, return the single file
		return []string{downloadedPath}, nil
	}

	// It's an archive, extract it
	extractDir := filepath.Join(targetDir, "extracted")
	extractedFiles, err := ExtractArchive(downloadedPath, extractDir)
	if err != nil {
		os.Remove(downloadedPath) // Clean up the archive file
		return nil, fmt.Errorf("failed to extract archive: %w", err)
	}

	// Clean up the archive file
	os.Remove(downloadedPath)

	if len(extractedFiles) == 0 {
		return nil, fmt.Errorf("no font files found in archive")
	}

	return extractedFiles, nil
}

// FontMatch represents a font match with source information
type FontMatch struct {
	ID       string
	Name     string
	Source   string
	FontInfo FontInfo
}

// FindFontMatches finds all fonts matching the given name across all sources
func FindFontMatches(fontName string) ([]FontMatch, error) {
	// Get repository (same as search command)
	r, err := GetRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}
	manifest := r.manifest

	// Normalize font name for comparison
	fontName = strings.ToLower(fontName)
	fontNameNoSpaces := strings.ReplaceAll(fontName, " ", "")

	var matches []FontMatch

	// Search through all sources
	for sourceName, source := range manifest.Sources {
		for id, font := range source.Fonts {
			// Check both the font name and ID with case-insensitive comparison
			fontNameLower := strings.ToLower(font.Name)
			idLower := strings.ToLower(id)
			fontNameNoSpacesLower := strings.ReplaceAll(fontNameLower, " ", "")
			idNoSpacesLower := strings.ReplaceAll(idLower, " ", "")

			// Check for exact match
			if fontNameLower == fontName ||
				fontNameNoSpacesLower == fontNameNoSpaces ||
				idLower == fontName ||
				idNoSpacesLower == fontNameNoSpaces {
				matches = append(matches, FontMatch{
					ID:       id,
					Name:     font.Name,
					Source:   sourceName,
					FontInfo: font,
				})
			}
		}
	}

	return matches, nil
}

// GetFontByID retrieves font information using a specific font ID (e.g., "google.roboto")
func GetFontByID(fontID string) ([]FontFile, error) {
	// Get repository (same as FindFontMatches)
	r, err := GetRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}
	manifest := r.manifest

	// Search through all sources for the specific ID
	for _, source := range manifest.Sources {
		if font, exists := source.Fonts[fontID]; exists {
			return convertFontInfoToFontFiles(font, fontID)
		}
	}

	return nil, fmt.Errorf("font not found: %s", fontID)
}

// convertFontInfoToFontFiles converts FontInfo to []FontFile
func convertFontInfoToFontFiles(font FontInfo, fontID string) ([]FontFile, error) {
	var fonts []FontFile
	seenURLs := make(map[string]bool) // Track seen URLs to avoid duplicates

	// Process each variant using the preserved variant-file mapping
	for _, variantName := range font.Variants {
		var downloadURL string

		// Use variant-specific files if available
		if font.VariantFiles != nil {
			if variantFiles, exists := font.VariantFiles[variantName]; exists {
				for fileType, url := range variantFiles {
					// Accept both individual font files and archive files
					if fileType == "ttf" || fileType == "otf" {
						downloadURL = url
						break
					}
				}
			}
		}

		// Fallback to general files if variant-specific not found
		if downloadURL == "" {
			for fileType, url := range font.Files {
				// Accept both individual font files and archive files
				if fileType == "ttf" || fileType == "otf" {
					downloadURL = url
					break
				}
			}
		}

		if downloadURL != "" {
			// Check if we've already processed this URL (for duplicate variants)
			if seenURLs[downloadURL] {
				continue
			}
			seenURLs[downloadURL] = true

			// For archive files, use the archive filename as the path
			// For individual font files, create a proper filename
			var fileName string
			if isArchiveFile(downloadURL) {
				fileName = filepath.Base(downloadURL)
			} else {
				fileName = createFontFileName(font.Name, variantName, downloadURL)
			}

			fonts = append(fonts, FontFile{
				Name:        font.Name,
				Variant:     variantName,
				Path:        fileName,
				DownloadURL: downloadURL,
			})
		}
	}
	if len(fonts) == 0 {
		return nil, fmt.Errorf("no valid font files found for %s", fontID)
	}

	return fonts, nil
}

// isArchiveFile checks if a URL points to an archive file
func isArchiveFile(url string) bool {
	ext := strings.ToLower(filepath.Ext(url))
	return ext == ".zip" || ext == ".xz" || strings.HasSuffix(strings.ToLower(url), ".tar.xz")
}

// GetFont retrieves font information from the manifest
func GetFont(fontName string) ([]FontFile, error) {
	// Get manifest
	manifest, err := GetManifest(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest: %w", err)
	}

	// Normalize font name for comparison
	fontName = strings.ToLower(fontName)
	fontNameNoSpaces := strings.ReplaceAll(fontName, " ", "")

	var fonts []FontFile
	found := false

	// First try exact matches
	for _, source := range manifest.Sources {
		for id, font := range source.Fonts {
			// Check both the font name and ID with case-insensitive comparison
			fontNameLower := strings.ToLower(font.Name)
			idLower := strings.ToLower(id)
			fontNameNoSpacesLower := strings.ReplaceAll(fontNameLower, " ", "")
			idNoSpacesLower := strings.ReplaceAll(idLower, " ", "")

			// Check for exact match first
			if fontNameLower == fontName || idLower == fontName ||
				fontNameNoSpacesLower == fontNameNoSpaces || idNoSpacesLower == fontNameNoSpaces {
				found = true

				// Process each variant using the preserved variant-file mapping
				for _, variantName := range font.Variants {
					var downloadURL string

					// Use variant-specific files if available
					if font.VariantFiles != nil {
						if variantFiles, exists := font.VariantFiles[variantName]; exists {
							for fileType, url := range variantFiles {
								if (fileType == "ttf" || fileType == "otf") && strings.HasSuffix(url, "."+fileType) {
									downloadURL = url
									break
								}
							}
						}
					}

					// Fallback to general files if variant-specific not found
					if downloadURL == "" {
						for fileType, url := range font.Files {
							if (fileType == "ttf" || fileType == "otf") && strings.HasSuffix(url, "."+fileType) {
								downloadURL = url
								break
							}
						}
					}

					if downloadURL != "" {
						fileName := createFontFileName(font.Name, variantName, downloadURL)
						fonts = append(fonts, FontFile{
							Name:        font.Name,
							Variant:     variantName,
							Path:        fileName,
							DownloadURL: downloadURL,
						})
					}
				}
			}
		}
	}

	if !found {
		return nil, fmt.Errorf("font '%s' not found in the Google Fonts repository", fontName)
	}

	return fonts, nil
}

// createFontFileName creates a proper filename for a font file
func createFontFileName(fontName, variant, url string) string {
	// Get the file extension from the URL or default to .ttf
	ext := filepath.Ext(url)
	if ext == "" {
		ext = ".ttf"
	}

	// Clean the font name for use in filename
	cleanName := strings.ReplaceAll(fontName, " ", "")
	cleanName = strings.ReplaceAll(cleanName, "-", "")
	cleanName = strings.ReplaceAll(cleanName, "_", "")

	// Clean the variant name for use in filename
	cleanVariant := strings.ReplaceAll(variant, " ", "")
	cleanVariant = strings.ReplaceAll(cleanVariant, "-", "")
	cleanVariant = strings.ReplaceAll(cleanVariant, "_", "")

	// Remove the font name from the variant if it's duplicated
	// e.g., "RobotoBlack" -> "Black"
	if strings.HasPrefix(strings.ToLower(cleanVariant), strings.ToLower(cleanName)) {
		cleanVariant = cleanVariant[len(cleanName):]
	}

	// Capitalize first letter of variant
	if len(cleanVariant) > 0 {
		cleanVariant = strings.ToUpper(cleanVariant[:1]) + cleanVariant[1:]
	}

	// Combine name and variant
	if cleanVariant != "" && cleanVariant != "Regular" {
		return cleanName + "-" + cleanVariant + ext
	}
	return cleanName + ext
}

// isFontFile checks if a file is a font file
func isFontFile(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".ttf") ||
		strings.HasSuffix(strings.ToLower(filename), ".otf")
}

// ListInstalledFonts returns a list of font files in the specified directory
func ListInstalledFonts(dir string) ([]string, error) {
	var fonts []string

	// Walk through the directory
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if the file is a font file
		ext := strings.ToLower(filepath.Ext(path))
		if isFontFile(ext) {
			fonts = append(fonts, filepath.Base(path))
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list fonts: %w", err)
	}

	return fonts, nil
}

// GetAllFonts returns a list of all fonts from the manifest
func GetAllFonts() []string {
	// Get the manifest to search through
	manifest, err := GetManifest(nil, nil)
	if err != nil {
		fmt.Printf("Error getting manifest: %v\n", err)
		return nil
	}

	if manifest == nil || manifest.Sources == nil {
		fmt.Println("Manifest or sources is nil")
		return nil
	}

	// Collect all font names from the manifest
	var allFonts []string
	seen := make(map[string]bool) // Track unique font names

	for _, source := range manifest.Sources {
		if source.Fonts == nil {
			continue
		}
		for id, font := range source.Fonts {
			// Use the font name if available, otherwise use the ID
			name := font.Name
			if name == "" {
				name = id
			}

			// Add the font name if we haven't seen it before
			if !seen[name] {
				allFonts = append(allFonts, name)
				seen[name] = true
			}

			// Add the ID if it's different from the name and we haven't seen it
			if name != id && !seen[id] {
				allFonts = append(allFonts, id)
				seen[id] = true
			}

			// Add a space-removed version of the name if it contains spaces
			if strings.Contains(name, " ") {
				noSpaces := strings.ReplaceAll(name, " ", "")
				if !seen[noSpaces] {
					allFonts = append(allFonts, noSpaces)
					seen[noSpaces] = true
				}
			}
		}
	}

	// Removed "Found X fonts in manifest" message for cleaner output
	return allFonts
}

// GetAllFontsCached returns a list of all fonts from the cached manifest (fast)
func GetAllFontsCached() []string {
	// Get cached manifest for speed
	manifest, err := GetCachedManifest()
	if err != nil {
		// If no cache available, return empty list
		return nil
	}

	if manifest == nil || manifest.Sources == nil {
		return nil
	}

	// Collect all font names from the manifest
	var allFonts []string
	seen := make(map[string]bool) // Track unique font names

	for _, source := range manifest.Sources {
		if source.Fonts == nil {
			continue
		}
		for id, font := range source.Fonts {
			// Use the font name if available, otherwise use the ID
			name := font.Name
			if name == "" {
				name = id
			}

			// Add the font name if we haven't seen it before
			if !seen[name] {
				allFonts = append(allFonts, name)
				seen[name] = true
			}

			// Add the ID if it's different from the name and we haven't seen it
			if name != id && !seen[id] {
				allFonts = append(allFonts, id)
				seen[id] = true
			}

			// Add a space-removed version of the name if it contains spaces
			if strings.Contains(name, " ") {
				noSpaces := strings.ReplaceAll(name, " ", "")
				if !seen[noSpaces] {
					allFonts = append(allFonts, noSpaces)
					seen[noSpaces] = true
				}
			}
		}
	}

	return allFonts
}
