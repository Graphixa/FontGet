package repo

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Font represents a font file from the Google Fonts repository
type Font struct {
	Name        string
	Path        string
	SHA         string
	DownloadURL string
}

// DownloadFont downloads a font file and verifies its SHA-256 hash
func DownloadFont(font *Font, targetDir string) (string, error) {
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
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	// Download file
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download font: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, font.DownloadURL)
	}

	// Create target file
	targetPath := filepath.Join(targetDir, font.Name)
	file, err := os.Create(targetPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

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

	return targetPath, nil
}

// GetFont retrieves font information from the Google Fonts repository
func GetFont(fontName string) ([]Font, error) {
	// Normalize font name
	fontName = strings.ToLower(fontName)
	url := fmt.Sprintf("https://api.github.com/repos/google/fonts/contents/ofl/%s", fontName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "fontget-cli")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch font info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("font '%s' not found", fontName)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, url)
	}

	var contents []struct {
		Name        string `json:"name"`
		Path        string `json:"path"`
		DownloadURL string `json:"download_url"`
		Type        string `json:"type"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&contents); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var fonts []Font
	for _, content := range contents {
		if content.Type != "file" || !isFontFile(content.Name) {
			continue
		}

		// Download the font file to calculate its SHA
		fontResp, err := http.Get(content.DownloadURL)
		if err != nil {
			return nil, fmt.Errorf("failed to download font file: %w", err)
		}
		defer fontResp.Body.Close()

		if fontResp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("HTTP %d: %s", fontResp.StatusCode, content.DownloadURL)
		}

		// Calculate SHA-256 of the font file
		hash := sha256.New()
		if _, err := io.Copy(hash, fontResp.Body); err != nil {
			return nil, fmt.Errorf("failed to calculate font hash: %w", err)
		}
		fontSHA := hex.EncodeToString(hash.Sum(nil))

		fonts = append(fonts, Font{
			Name:        content.Name,
			Path:        content.Path,
			SHA:         fontSHA,
			DownloadURL: content.DownloadURL,
		})
	}

	if len(fonts) == 0 {
		return nil, fmt.Errorf("no font files found for '%s'", fontName)
	}

	return fonts, nil
}

// isFontFile checks if the file extension indicates a font file
func isFontFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".ttf", ".otf", ".ttc", ".otc", ".pfb", ".pfm", ".pfa", ".bdf", ".pcf", ".psf", ".psfu":
		return true
	default:
		return false
	}
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

// GetAllFonts returns all fonts from the manifest, sorted alphabetically by name
func GetAllFonts() []SearchResult {
	manifest, err := GetManifest(nil)
	if err != nil {
		return nil
	}

	var results []SearchResult
	for sourceID, source := range manifest.Sources {
		for id, font := range source.Fonts {
			result := createSearchResult(id, font, sourceID, source.Name)
			result.Score = 0 // No score for listing all fonts
			results = append(results, result)
		}
	}

	// Sort results alphabetically by name
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i].Name > results[j].Name {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return results
}
