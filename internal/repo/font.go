package repo

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"fontget/internal/config"
	"fontget/internal/logging"
	"fontget/internal/network"
	"fontget/internal/output"
	"fontget/internal/platform"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Use a mainstream Windows Chrome UA by default. Some WAFs are more permissive to Windows+Chrome
// than macOS or non-browser user agents.
const browserLikeUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36"

// FetchURLContent fetches content from a URL with cross-platform compatibility
func FetchURLContent(url string) (string, error) {
	// Create HTTP client with timeout
	appConfig := config.GetUserPreferences()
	generalTimeout := config.ParseDuration(appConfig.Network.RequestTimeout, 10*time.Second)
	client := &http.Client{
		Timeout: generalTimeout,
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

// Font represents a font file from the Google Fonts repository
type FontFile struct {
	Name        string
	Variant     string
	Path        string
	SHA         string
	DownloadURL string
}

// DownloadFontOptions configures DownloadFont / DownloadAndExtractFont.
type DownloadFontOptions struct {
	// SuppressVerboseProgressLine omits the per-file "[INFO] Downloading …" verbose line.
	// Use true while Bubble Tea (or any other UI) owns stdout so output does not interleave.
	SuppressVerboseProgressLine bool

	// OnBytesDownloaded, when set, is called periodically as bytes are read from the response body.
	// totalBytes is the HTTP Content-Length when known, otherwise -1.
	// Implementations must be lightweight.
	OnBytesDownloaded func(downloadedBytes int64, totalBytes int64)

	// OnExtractProgress, when set, is called as extract progresses.
	// totalFiles is the number of font files to be extracted when known, otherwise -1.
	OnExtractProgress func(extractedFiles int, totalFiles int)

	// OnResponseHeaders, when set, is called once after we receive an HTTP 200 response.
	// contentType and contentDisposition are raw header values (may be empty).
	// finalURL is the post-redirect URL when available.
	OnResponseHeaders func(info HTTPResponseInfo)
}

// DownloadFont downloads a font file and verifies its SHA-256 hash if available
func DownloadFont(font *FontFile, targetDir string, opts *DownloadFontOptions) (string, error) {
	start := time.Now()
	dbg := func(format string, args ...interface{}) {
		args = append(args, time.Since(start).Milliseconds())
		output.GetDebug().State(format+" (%dms)", args...)
	}
	dbgFileSize := func(path string) {
		info, err := os.Stat(path)
		if err != nil {
			return
		}
		if info.Size() <= 0 {
			return
		}
		dbg("DownloadFont: downloaded size=%d bytes path=%s", info.Size(), path)
	}

	// Create target directory if it doesn't exist
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create target directory: %w", err)
	}

	// Compute target path early so we can fall back to alternate download methods.
	targetPath := filepath.Join(targetDir, font.Path)

	// Create HTTP request
	req, err := http.NewRequest("GET", font.DownloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	// Some upstreams will respond with WAF/bot challenges for non-browser clients.
	// Use a browser-like UA to reduce false bot detection.
	req.Header.Set("User-Agent", browserLikeUserAgent)
	req.Header.Set("Accept", "*/*")

	// Download file
	appConfig := config.GetUserPreferences()
	downloadTimeout := config.ParseDuration(appConfig.Network.DownloadTimeout, 30*time.Second)
	requestTimeout := config.ParseDuration(appConfig.Network.RequestTimeout, 10*time.Second)
	fallbackEnabled := appConfig.Network.EnableExternalDownloadFallback

	host := ""
	if u, parseErr := url.Parse(font.DownloadURL); parseErr == nil && u.Host != "" {
		host = strings.ToLower(u.Host)
	}
	// Font Squirrel often delays/challenges non-browser clients. Prefer failing fast and falling back.
	fastHeaderTimeout := requestTimeout
	if fallbackEnabled && strings.Contains(host, "fontsquirrel.com") {
		if fastHeaderTimeout <= 0 || fastHeaderTimeout > 3*time.Second {
			fastHeaderTimeout = 3 * time.Second
		}
	}

	// Don't use http.Client.Timeout for downloads - it times out even during active transfers
	// Instead, use ResponseHeaderTimeout to detect connection issues early
	// The stall detector handles inactivity detection (no overall timeout needed)
	resp, err := doDownloadRequestWithHeaderTimeout(req, fastHeaderTimeout, 15*time.Second, start, dbg)
	if err != nil {
		// Always attempt external fallbacks when enabled.
		if fallbackEnabled {
			dbg("DownloadFont: standard request failed: %v", err)
			rep, fbErr := network.DownloadWithFallbacks(font.DownloadURL, targetPath, network.DownloadFallbackOptions{
				UserAgent: browserLikeUserAgent,
				Headers: map[string]string{
					"Accept": "*/*",
				},
			})
			if rep != nil {
				for _, step := range rep.Steps {
					dbg("DownloadFont fallback step: tool=%s path=%s result=%s detail=%q", step.Tool, step.Path, step.Result, step.Detail)
				}
			}
			if fbErr == nil {
				toolName, toolPath := rep.UsedTool()
				dbg("DownloadFont: %s -> %s (via %s)", font.DownloadURL, targetPath, toolName)
				logging.GetLogger().Info("External download succeeded using %s (%s)", toolName, toolPath)
				dbgFileSize(targetPath)
				return targetPath, nil
			}
		}
		return "", fmt.Errorf("failed to download font: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Bot/WAF challenge: retry using external tools when enabled.
		if network.IsBotChallenge(resp) {
			logging.GetLogger().Info("Download: HTTP %d bot/WAF challenge from host %s", resp.StatusCode, host)

			wafAction := network.HeaderValueFold(resp.Header, "x-amzn-waf-action")
			dbg("DownloadFont: IsBotChallenge=true status=%d x-amzn-waf-action=%q", resp.StatusCode, wafAction)

			if !fallbackEnabled {
				logging.GetLogger().Info("External download fallback disabled (Network.EnableExternalDownloadFallback=false); not retrying with external tools")
				output.GetVerbose().Warning("Upstream returned HTTP %d (bot/WAF challenge). External download fallback is disabled in config.", resp.StatusCode)
				dbg("DownloadFont: skipping DownloadWithFallbacks (EnableExternalDownloadFallback=false)")
				return "", fmt.Errorf("HTTP %d (blocked by upstream challenge). Enable Network.EnableExternalDownloadFallback in config.yaml or retry later: %s", resp.StatusCode, font.DownloadURL)
			}

			output.GetVerbose().Info("Upstream returned HTTP %d (bot/WAF challenge). Retrying with external download tools if available.", resp.StatusCode)

			rep, fbErr := network.DownloadWithFallbacks(font.DownloadURL, targetPath, network.DownloadFallbackOptions{
				UserAgent: browserLikeUserAgent,
				Headers: map[string]string{
					"Accept": "*/*",
				},
			})
			for _, step := range rep.Steps {
				dbg("DownloadFont fallback step: tool=%s path=%s result=%s detail=%q", step.Tool, step.Path, step.Result, step.Detail)
			}

			if fbErr == nil {
				toolName, toolPath := rep.UsedTool()
				logging.GetLogger().Info("External download succeeded using %s (%s)", toolName, toolPath)
				output.GetVerbose().Info("Download completed using %s after HTTP %d challenge.", toolName, resp.StatusCode)
				dbg("DownloadFont: %s -> %s (via %s)", font.DownloadURL, targetPath, toolName)
				dbgFileSize(targetPath)
				return targetPath, nil
			}

			logging.GetLogger().Error("External download fallback failed for %s: %v", font.DownloadURL, fbErr)
			output.GetVerbose().Error("External download tools did not succeed after HTTP %d challenge. Use --debug for per-tool details.", resp.StatusCode)
			output.GetDebug().Error("DownloadFont: DownloadWithFallbacks failed: %v", fbErr)
			return "", fmt.Errorf("HTTP %d (blocked by upstream challenge): %s", resp.StatusCode, font.DownloadURL)
		}
		// Any non-200: attempt fallbacks when enabled.
		if fallbackEnabled {
			dbg("DownloadFont: HTTP %d from upstream, attempting fallbacks", resp.StatusCode)
			rep, fbErr := network.DownloadWithFallbacks(font.DownloadURL, targetPath, network.DownloadFallbackOptions{
				UserAgent: browserLikeUserAgent,
				Headers: map[string]string{
					"Accept": "*/*",
				},
			})
			if rep != nil {
				for _, step := range rep.Steps {
					dbg("DownloadFont fallback step: tool=%s path=%s result=%s detail=%q", step.Tool, step.Path, step.Result, step.Detail)
				}
			}
			if fbErr == nil {
				toolName, toolPath := rep.UsedTool()
				dbg("DownloadFont: %s -> %s (via %s)", font.DownloadURL, targetPath, toolName)
				logging.GetLogger().Info("External download succeeded using %s (%s)", toolName, toolPath)
				dbgFileSize(targetPath)
				return targetPath, nil
			}
		}
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, font.DownloadURL)
	}

	// Best-effort response info capture for downstream archive detection.
	if opts != nil && opts.OnResponseHeaders != nil {
		finalURL := ""
		if resp != nil && resp.Request != nil && resp.Request.URL != nil {
			finalURL = resp.Request.URL.String()
		}
		opts.OnResponseHeaders(HTTPResponseInfo{
			ContentType:        resp.Header.Get("Content-Type"),
			ContentDisposition: resp.Header.Get("Content-Disposition"),
			FinalURL:           finalURL,
		})
	}

	suppressVerbose := opts != nil && opts.SuppressVerboseProgressLine
	if !suppressVerbose {
		displayName := font.Path
		if displayName == "" {
			displayName = filepath.Base(targetPath)
		}
		if u, parseErr := url.Parse(font.DownloadURL); parseErr == nil && u.Host != "" {
			output.GetVerbose().Info("Downloading %s from %s", displayName, u.Host)
		} else {
			output.GetVerbose().Info("Downloading %s", displayName)
		}
	}

	// Wrap response body with stall detection
	// No overall timeout (0) - downloads can take as long as needed if there's activity
	// Only timeout if no activity for downloadTimeout duration
	stallReader := network.WrapReaderWithStallDetection(resp.Body, downloadTimeout, 0)
	defer stallReader.Close()

	totalBytes := resp.ContentLength
	if totalBytes <= 0 {
		totalBytes = -1
	}
	// Optional byte progress callback.
	var reader io.Reader = stallReader
	if opts != nil && opts.OnBytesDownloaded != nil {
		reader = newProgressReader(stallReader, totalBytes, opts.OnBytesDownloaded)
	}

	// Create target file
	file, err := os.Create(targetPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// If we have a SHA hash, verify it
	if font.SHA != "" {
		// Create SHA-256 hash
		hash := sha256.New()
		tee := io.TeeReader(reader, hash)

		// Copy file content with stall detection
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
		// Use stallReader instead of resp.Body for stall detection
		if _, err := io.Copy(file, reader); err != nil {
			return "", fmt.Errorf("failed to write file: %w", err)
		}
	}

	logging.GetLogger().Info("Download complete: %s -> %s", font.Path, targetPath)
	dbg("DownloadFont: %s -> %s", font.DownloadURL, targetPath)
	dbgFileSize(targetPath)

	return targetPath, nil
}

func doDownloadRequestWithHeaderTimeout(req *http.Request, fastHeaderTimeout time.Duration, slowHeaderTimeout time.Duration, start time.Time, dbg func(string, ...interface{})) (*http.Response, error) {
	if fastHeaderTimeout <= 0 {
		fastHeaderTimeout = 10 * time.Second
	}
	if slowHeaderTimeout <= fastHeaderTimeout {
		slowHeaderTimeout = fastHeaderTimeout
	}

	doOnce := func(timeout time.Duration, forceHTTP1 bool) (*http.Response, error) {
		transport := &http.Transport{
			ResponseHeaderTimeout: timeout,
		}
		if forceHTTP1 {
			transport.ForceAttemptHTTP2 = false
		}
		client := &http.Client{Transport: transport}
		return client.Do(req)
	}

	resp, err := doOnce(fastHeaderTimeout, false)
	if err == nil {
		return resp, nil
	}

	// Retry only on the specific slow-header case seen with Font Squirrel.
	// First retry: longer timeout and force HTTP/1.1 (some sites behave better without HTTP/2).
	if isHTTP2HeaderTimeout(err) && slowHeaderTimeout > fastHeaderTimeout {
		if dbg != nil {
			dbg("DownloadFont: header timeout hit (%v), retrying with %v (http1)", fastHeaderTimeout, slowHeaderTimeout)
		} else {
			output.GetDebug().State("DownloadFont: header timeout hit (%v), retrying with %v (http1) (%dms)", fastHeaderTimeout, slowHeaderTimeout, time.Since(start).Milliseconds())
		}
		return doOnce(slowHeaderTimeout, true)
	}

	return nil, err
}

func doDownloadRequestOnce(req *http.Request, headerTimeout time.Duration, forceHTTP1 bool) (*http.Response, error) {
	if headerTimeout <= 0 {
		headerTimeout = 10 * time.Second
	}
	transport := &http.Transport{
		ResponseHeaderTimeout: headerTimeout,
	}
	if forceHTTP1 {
		transport.ForceAttemptHTTP2 = false
	}
	client := &http.Client{Transport: transport}
	return client.Do(req)
}

func isHTTP2HeaderTimeout(err error) bool {
	if err == nil {
		return false
	}
	// Common Go HTTP/2 message:
	// "http2: timeout awaiting response headers"
	if strings.Contains(err.Error(), "timeout awaiting response headers") {
		return true
	}
	// Generic net.Error timeout can also occur; keep it tight so we don't retry everything.
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout() && strings.Contains(err.Error(), "http2:")
}

// DownloadAndExtractFont downloads a font file (which may be an archive) and extracts it if needed
func DownloadAndExtractFont(font *FontFile, targetDir string, opts *DownloadFontOptions) ([]string, error) {
	start := time.Now()

	isProbablyHTML := func(path string) bool {
		f, err := os.Open(path)
		if err != nil {
			return false
		}
		defer f.Close()
		var buf [512]byte
		n, _ := f.Read(buf[:])
		b := bytes.TrimSpace(bytes.ToLower(buf[:n]))
		return bytes.HasPrefix(b, []byte("<!doctype html")) || bytes.HasPrefix(b, []byte("<html"))
	}

	validateFontFile := func(path string) error {
		// Ensure it's parseable as a font (TTF/OTF/collections supported by sfnt).
		_, err := platform.ExtractFontMetadata(path)
		if err != nil {
			return err
		}
		return nil
	}

	// Create target directory if it doesn't exist
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create target directory: %w", err)
	}

	// Download the file first (capture response headers if available)
	var httpInfo HTTPResponseInfo
	downloadOpts := opts
	if opts != nil {
		cpy := *opts
		prev := cpy.OnResponseHeaders
		cpy.OnResponseHeaders = func(info HTTPResponseInfo) {
			httpInfo = info
			if prev != nil {
				prev(info)
			}
		}
		downloadOpts = &cpy
	}

	downloadedPath, err := DownloadFont(font, targetDir, downloadOpts)
	if err != nil {
		return nil, err
	}

	// Check if the downloaded file is an archive.
	// Decision order: extension → header guess → magic bytes (final truth).
	archiveTypeByExt := DetectArchiveType(downloadedPath)
	archiveTypeByHeader := InferArchiveTypeFromHeaders(httpInfo.ContentType, httpInfo.ContentDisposition)
	archiveTypeByMagic := DetectArchiveTypeFromFile(downloadedPath)

	archiveType := archiveTypeByExt
	if archiveType == ArchiveTypeUnknown {
		archiveType = archiveTypeByHeader
	}
	// Magic bytes are final truth. If magic indicates an archive, extract; otherwise do not.
	// This prevents headers from forcing extraction of non-archives.
	if archiveTypeByMagic != ArchiveTypeUnknown {
		archiveType = archiveTypeByMagic
	} else if archiveType != ArchiveTypeUnknown {
		// Header/ext suggested archive but file magic did not confirm.
		archiveType = ArchiveTypeUnknown
	}

	// Optional debug visibility into header/magic decision.
	output.GetDebug().State("DownloadAndExtractFont headers: ext=%v header=%v magic=%v final=%v content-type=%q content-disposition=%q",
		archiveTypeByExt, archiveTypeByHeader, archiveTypeByMagic, archiveType, httpInfo.ContentType, httpInfo.ContentDisposition)
	output.GetDebug().State("DownloadAndExtractFont timing: total=%dms", time.Since(start).Milliseconds())

	if archiveType == ArchiveTypeUnknown {
		// Not an archive: validate it's a real font before returning. This prevents HTML/WAF payloads
		// (or empty/garbage files) being treated as installed fonts later.
		if err := validateFontFile(downloadedPath); err != nil {
			_ = os.Remove(downloadedPath)
			if isProbablyHTML(downloadedPath) {
				return nil, fmt.Errorf("download did not return a font file (received HTML; upstream likely served a challenge page): %w", err)
			}
			return nil, fmt.Errorf("download did not return a valid font file: %w", err)
		}
		return []string{downloadedPath}, nil
	}

	// It's an archive, extract it
	extractDir := filepath.Join(targetDir, "extracted")
	extractedFiles, err := ExtractArchiveWithOptions(downloadedPath, extractDir, &ExtractOptions{
		OnFontFileExtracted: func(done int, total int) {
			if opts != nil && opts.OnExtractProgress != nil {
				opts.OnExtractProgress(done, total)
			}
		},
	})
	if err != nil {
		os.Remove(downloadedPath) // Clean up the archive file
		return nil, fmt.Errorf("failed to extract archive: %w", err)
	}

	// Clean up the archive file
	os.Remove(downloadedPath)

	if len(extractedFiles) == 0 {
		return nil, fmt.Errorf("no font files found in archive")
	}

	// Validate extracted files; drop garbage so we never "install" it.
	valid := make([]string, 0, len(extractedFiles))
	for _, p := range extractedFiles {
		if err := validateFontFile(p); err != nil {
			_ = os.Remove(p)
			continue
		}
		valid = append(valid, p)
	}
	if len(valid) == 0 {
		return nil, fmt.Errorf("no valid font files found after extraction (archive contents were not parseable as fonts)")
	}
	return valid, nil
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
	// Get repository
	r, err := GetRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}
	manifest := r.manifest

	// Normalize font name for comparison
	fontName = strings.ToLower(fontName)
	fontNameNoSpaces := strings.ReplaceAll(fontName, " ", "")

	var matches []FontMatch

	// Search through all sources in source priority order
	sourceOrder := []string{"Google Fonts", "Nerd Fonts", "Font Squirrel"}

	// First check predefined sources in priority order
	for _, sourceName := range sourceOrder {
		if source, exists := manifest.Sources[sourceName]; exists {
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
	}

	// Then check any custom sources (not in predefined list)
	for sourceName, source := range manifest.Sources {
		// Skip if already processed
		isPredefined := false
		for _, predefined := range sourceOrder {
			if sourceName == predefined {
				isPredefined = true
				break
			}
		}
		if isPredefined {
			continue
		}

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

	// Sort the results to make them deterministic (Go map iteration order is not guaranteed)
	sort.Strings(allFonts)

	return allFonts
}
