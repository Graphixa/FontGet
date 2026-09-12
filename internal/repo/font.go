package repo

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"fontget/internal/config"
	"fontget/internal/logging"
	"fontget/internal/network"
	"fontget/internal/output"
	"fontget/internal/platform"
	"fontget/internal/sources"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const downloadUserAgentFallback = "Mozilla/5.0 (compatible; FontGet/1.0; +https://github.com/Graphixa/FontGet)"

func isValidHeaderValue(s string) bool {
	if strings.ContainsAny(s, "\r\n") {
		return false
	}
	for i := 0; i < len(s); i++ {
		// Reject ASCII control chars and DEL.
		if s[i] < 0x20 || s[i] == 0x7f {
			return false
		}
	}
	return true
}

func resolveDownloadUserAgent() string {
	cfg := config.GetUserPreferences()
	ua := strings.TrimSpace(cfg.Network.DownloadUserAgent)
	if ua != "" && isValidHeaderValue(ua) {
		return ua
	}
	return downloadUserAgentFallback
}

// DownloadUserAgent is Network.DownloadUserAgent from preferences (embedded default if unset).
func DownloadUserAgent() string { return resolveDownloadUserAgent() }

func isZipMagic(b []byte) bool {
	if len(b) < 4 {
		return false
	}
	return b[0] == 'P' && b[1] == 'K' && ((b[2] == 3 && b[3] == 4) || (b[2] == 5 && b[3] == 6) || (b[2] == 7 && b[3] == 8))
}

var (
	downloadHostMu    sync.Mutex
	downloadHostSlots = map[string]chan struct{}{}
)

func acquireDownloadHostSlot(ctx context.Context, host string) (func(), error) {
	if ctx == nil {
		ctx = context.Background()
	}
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return func() {}, nil
	}

	downloadHostMu.Lock()
	ch, ok := downloadHostSlots[host]
	if !ok {
		// Serialize downloads per-host by default (reduces “parallel bot” traffic patterns).
		ch = make(chan struct{}, 1)
		downloadHostSlots[host] = ch
	}
	downloadHostMu.Unlock()

	select {
	case ch <- struct{}{}:
		return func() { <-ch }, nil
	case <-ctx.Done():
		return func() {}, ctx.Err()
	}
}

func fallbackHeadersFromRequest(req *http.Request) map[string]string {
	if req == nil {
		return nil
	}
	fb := map[string]string{"Accept": req.Header.Get("Accept")}
	if ae := req.Header.Get("Accept-Encoding"); ae != "" {
		fb["Accept-Encoding"] = ae
	}
	if ref := req.Header.Get("Referer"); ref != "" {
		fb["Referer"] = ref
	}
	if al := req.Header.Get("Accept-Language"); al != "" {
		fb["Accept-Language"] = al
	}
	return fb
}

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
	// DownloadCandidates is preference-ordered URLs for this variant payload.
	// When non-empty, DownloadCandidates[0] matches DownloadURL. DownloadAndExtractFont
	// tries each URL in order on download/extract/validation failure.
	DownloadCandidates []string
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

	// ArchiveSourcePrefix is the lowercase source segment before the first '.' in a FontGet font ID
	// (e.g. "fontshare", "league", "nerd"). When set, archive extraction may use explicit path rules for
	// that upstream before generic bucket/score selection.
	ArchiveSourcePrefix string

	// ArchiveFontID is the full FontGet font ID (e.g. "nerd.noto-sans-mono") used by source-specific
	// archive selectors when FontFile.Name alone is insufficient.
	ArchiveFontID string

	// Context cancels host-slot waits, HTTP requests, retry backoff and external tools.
	Context context.Context
}

// DownloadFont downloads a font file and verifies its SHA-256 hash if available.
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

	if font == nil {
		return "", fmt.Errorf("font is nil")
	}
	ctx := context.Background()
	if opts != nil && opts.Context != nil {
		ctx = opts.Context
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	if _, err := ParseExpectedSHA256(font.SHA); err != nil {
		return "", err
	}
	if err := checksumAppliesToCandidate(font, font.DownloadURL); err != nil {
		return "", err
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("%w: failed to create target directory: %v", network.ErrLocalFailure, err)
	}

	targetPath := filepath.Join(targetDir, font.Path)

	req, err := http.NewRequestWithContext(ctx, "GET", font.DownloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	appConfig := config.GetUserPreferences()
	downloadTimeout := config.ParseDuration(appConfig.Network.DownloadTimeout, 30*time.Second)
	requestTimeout := config.ParseDuration(appConfig.Network.RequestTimeout, 10*time.Second)
	fallbackEnabled := appConfig.Network.EnableExternalDownloadFallback

	host := ""
	path := ""
	if u, parseErr := url.Parse(font.DownloadURL); parseErr == nil && u.Host != "" {
		host = strings.ToLower(u.Host)
		path = u.Path
	}

	isFontSquirrel := strings.Contains(host, "fontsquirrel.com")
	if isFontSquirrel {
		req.Header.Set("User-Agent", resolveDownloadUserAgent())
		req.Header.Set("Accept", "application/zip, application/octet-stream, */*")
		req.Header.Set("Accept-Encoding", "identity")
		req.Header.Set("Referer", "https://www.fontsquirrel.com/")
	} else {
		req.Header.Set("User-Agent", resolveDownloadUserAgent())
	}
	fastHeaderTimeout := requestTimeout
	if fallbackEnabled && strings.Contains(host, "fontsquirrel.com") {
		if fastHeaderTimeout <= 0 || fastHeaderTimeout > 3*time.Second {
			fastHeaderTimeout = 3 * time.Second
		}
	}

	releaseHost, err := acquireDownloadHostSlot(ctx, host)
	if err != nil {
		return "", err
	}
	defer releaseHost()

	fbOpts := network.DownloadFallbackOptions{
		UserAgent: req.Header.Get("User-Agent"),
		Headers:   fallbackHeadersFromRequest(req),
		Context:   ctx,
		Exec: network.ExecOptions{
			InactivityTimeout: downloadTimeout,
		},
	}
	if downloadTimeout < network.DefaultExternalInactivityTimeout {
		fbOpts.Exec.InactivityTimeout = network.DefaultExternalInactivityTimeout
	}

	completeExternal := func(rep *network.DownloadFallbackReport) (string, error) {
		if rep != nil {
			for _, step := range rep.Steps {
				dbg("DownloadFont fallback step: tool=%s path=%s result=%s detail=%q", step.Tool, step.Path, step.Result, step.Detail)
			}
		}
		if err := completeDownloadedFile(targetPath, font.SHA); err != nil {
			return "", err
		}
		toolName, toolPath := rep.UsedTool()
		dbg("DownloadFont: %s -> %s (via %s)", font.DownloadURL, targetPath, toolName)
		logging.GetLogger().Info("External download succeeded using %s (%s)", toolName, toolPath)
		dbgFileSize(targetPath)
		return targetPath, nil
	}

	resp, err := doDownloadRequestWithHeaderTimeout(req, fastHeaderTimeout, 15*time.Second, start, dbg)
	if err != nil {
		action := network.ClassifyDownloadError(err)
		if action == network.ActionFailPackage || action == network.ActionFailLocal {
			return "", err
		}
		if fallbackEnabled && action == network.ActionExternalFallback {
			dbg("DownloadFont: standard request failed: %v", err)
			rep, fbErr := network.DownloadWithFallbacks(font.DownloadURL, targetPath, fbOpts)
			if fbErr == nil {
				return completeExternal(rep)
			}
			if network.ClassifyDownloadError(fbErr) != network.ActionExternalFallback {
				return "", fbErr
			}
		}
		return "", fmt.Errorf("failed to download font: %w", err)
	}

	action := network.ClassifyHTTPStatus(resp.StatusCode, network.IsBotChallenge(resp))
	if action != network.ActionSuccess {
		retryAfter, _ := network.ParseRetryAfter(resp.Header, time.Now())
		network.DrainAndCloseBody(resp.Body)
		switch action {
		case network.ActionAdvanceCandidate, network.ActionFailCandidate:
			return "", network.NewHTTPStatusError(resp.StatusCode, font.DownloadURL, retryAfter)
		case network.ActionRateLimit:
			if !network.RetryAfterWithinBudget(retryAfter) {
				return "", fmt.Errorf("%w: Retry-After %s exceeds wait budget", network.ErrRateLimited, retryAfter)
			}
			return "", network.NewHTTPStatusError(resp.StatusCode, font.DownloadURL, retryAfter)
		case network.ActionExternalFallback:
			if !fallbackEnabled {
				logging.GetLogger().Info("External download fallback disabled (Network.EnableExternalDownloadFallback=false); not retrying with external tools")
				output.GetVerbose().Warning("Upstream returned HTTP %d (bot/WAF challenge). External download fallback is disabled in config.", resp.StatusCode)
				return "", fmt.Errorf("HTTP %d (blocked by upstream challenge). Enable Network.EnableExternalDownloadFallback in config.yaml or retry later: %s", resp.StatusCode, network.RedactDownloadURL(font.DownloadURL))
			}
			output.GetVerbose().Info("Upstream returned HTTP %d (bot/WAF challenge). Retrying with external download tools if available.", resp.StatusCode)
			rep, fbErr := network.DownloadWithFallbacks(font.DownloadURL, targetPath, fbOpts)
			if fbErr == nil {
				return completeExternal(rep)
			}
			logging.GetLogger().Error("External download fallback failed for %s: %v", font.DownloadURL, fbErr)
			return "", fmt.Errorf("HTTP %d (blocked by upstream challenge): %s", resp.StatusCode, network.RedactDownloadURL(font.DownloadURL))
		case network.ActionRetrySame:
			if fallbackEnabled {
				fbOnce := fbOpts
				fbOnce.MaxAttempts = 1
				dbg("DownloadFont: HTTP %d after native retries, trying external tools once", resp.StatusCode)
				rep, fbErr := network.DownloadWithFallbacks(font.DownloadURL, targetPath, fbOnce)
				if fbErr == nil {
					return completeExternal(rep)
				}
			}
			return "", network.NewHTTPStatusError(resp.StatusCode, font.DownloadURL, retryAfter)
		default:
			return "", network.NewHTTPStatusError(resp.StatusCode, font.DownloadURL, retryAfter)
		}
	}

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

	expectZIP := isFontSquirrel && strings.Contains(path, "/fontfacekit/")
	if !expectZIP {
		ct := strings.ToLower(resp.Header.Get("Content-Type"))
		if isFontSquirrel && strings.Contains(ct, "zip") {
			expectZIP = true
		}
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

	stallReader := network.WrapReaderWithStallDetection(resp.Body, downloadTimeout, 0)
	defer stallReader.Close()

	totalBytes := resp.ContentLength
	if totalBytes <= 0 {
		totalBytes = -1
	}
	var reader io.Reader = stallReader
	if opts != nil && opts.OnBytesDownloaded != nil {
		reader = newProgressReader(stallReader, totalBytes, opts.OnBytesDownloaded)
	}

	if expectZIP {
		br := bufio.NewReader(reader)
		if hdr, peekErr := br.Peek(4); peekErr == nil {
			if !isZipMagic(hdr) {
				return "", fmt.Errorf("download did not return a ZIP archive (possible upstream challenge): %s", network.RedactDownloadURL(font.DownloadURL))
			}
		}
		reader = br
	}

	file, err := os.Create(targetPath)
	if err != nil {
		return "", fmt.Errorf("%w: failed to create file: %v", network.ErrLocalFailure, err)
	}
	if _, err := io.Copy(file, reader); err != nil {
		_ = file.Close()
		_ = os.Remove(targetPath)
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("failed to write file: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(targetPath)
		return "", fmt.Errorf("%w: close download: %v", network.ErrLocalFailure, err)
	}

	if err := completeDownloadedFile(targetPath, font.SHA); err != nil {
		return "", err
	}

	logging.GetLogger().Info("Download complete: %s -> %s", font.Path, targetPath)
	dbg("DownloadFont: %s -> %s", font.DownloadURL, targetPath)
	dbgFileSize(targetPath)
	return targetPath, nil
}

func completeDownloadedFile(path, expectedSHA string) error {
	if err := VerifyFileSHA256(path, expectedSHA); err != nil {
		if rerr := os.Remove(path); rerr != nil && !os.IsNotExist(rerr) {
			return fmt.Errorf("%w (remove rejected file: %v)", err, rerr)
		}
		return err
	}
	return nil
}

func doDownloadRequestWithHeaderTimeout(req *http.Request, fastHeaderTimeout time.Duration, slowHeaderTimeout time.Duration, start time.Time, dbg func(string, ...interface{})) (*http.Response, error) {
	if fastHeaderTimeout <= 0 {
		fastHeaderTimeout = 10 * time.Second
	}
	if slowHeaderTimeout <= fastHeaderTimeout {
		slowHeaderTimeout = fastHeaderTimeout
	}

	// #nosec G404 -- non-cryptographic jitter for HTTP retry backoff only (not security-sensitive).
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	onRedirect := func(from *url.URL, to *url.URL, viaCount int) {
		if dbg == nil || from == nil || to == nil {
			return
		}
		dbg("DownloadFont: redirect %d %s -> %s", viaCount, from.String(), to.String())
	}

	doOnce := func(timeout time.Duration, forceHTTP1 bool) (*http.Response, error) {
		client := network.NewDownloadHTTPClient(timeout, forceHTTP1, onRedirect)
		return client.Do(req)
	}

	const maxTransientAttempts = 3
	backoff := 150 * time.Millisecond

	forceHTTP1 := false
	timeout := fastHeaderTimeout

	for attempt := 1; attempt <= maxTransientAttempts; attempt++ {
		resp, err := doOnce(timeout, forceHTTP1)
		if err != nil {
			// Retry only on the specific slow-header case seen with Font Squirrel.
			// First retry: longer timeout and force HTTP/1.1 (some sites behave better without HTTP/2).
			if isHTTP2HeaderTimeout(err) && slowHeaderTimeout > fastHeaderTimeout && !forceHTTP1 {
				if dbg != nil {
					dbg("DownloadFont: header timeout hit (%v), retrying with %v (http1)", fastHeaderTimeout, slowHeaderTimeout)
				} else {
					output.GetDebug().State("DownloadFont: header timeout hit (%v), retrying with %v (http1) (%dms)", fastHeaderTimeout, slowHeaderTimeout, time.Since(start).Milliseconds())
				}
				forceHTTP1 = true
				timeout = slowHeaderTimeout
				continue
			}
			return nil, err
		}

		if resp != nil && network.ShouldRetryGoDownloadStatus(resp.StatusCode) && attempt < maxTransientAttempts {
			wait := backoff + time.Duration(rng.Intn(120))*time.Millisecond
			if resp.StatusCode == http.StatusTooManyRequests {
				if ra, ok := network.ParseRetryAfter(resp.Header, time.Now()); ok {
					if !network.RetryAfterWithinBudget(ra) {
						network.DrainAndCloseBody(resp.Body)
						return nil, fmt.Errorf("%w: Retry-After %s exceeds wait budget", network.ErrRateLimited, ra)
					}
					wait = ra
				}
			}
			network.DrainAndCloseBody(resp.Body)
			if dbg != nil {
				dbg("DownloadFont: transient HTTP %d, retrying (attempt %d/%d) hdr=%s", resp.StatusCode, attempt, maxTransientAttempts, network.FormatHTTPHeadersForDebug(resp.Header))
			}
			if err := sleepRequest(req, wait); err != nil {
				return nil, err
			}
			backoff *= 2
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("download request failed after %d attempts", maxTransientAttempts)
}

func sleepRequest(req *http.Request, d time.Duration) error {
	ctx := context.Background()
	if req != nil && req.Context() != nil {
		ctx = req.Context()
	}
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
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

// DownloadAndExtractFont downloads a font file (which may be an archive) and extracts it if needed.
// When FontFile.DownloadCandidates is set, formats are tried in preference order on failure.
func DownloadAndExtractFont(font *FontFile, targetDir string, opts *DownloadFontOptions) ([]string, error) {
	if font == nil {
		return nil, fmt.Errorf("font is nil")
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create target directory: %w", err)
	}

	candidates := font.DownloadCandidates
	if len(candidates) == 0 && strings.TrimSpace(font.DownloadURL) != "" {
		candidates = []string{font.DownloadURL}
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no download URL for %s", font.Name)
	}

	var lastErr error
	var outcomes []string
	primaryURL := strings.TrimSpace(font.DownloadURL)
	if primaryURL == "" {
		primaryURL = candidates[0]
	}
	for i, candidateURL := range candidates {
		candidateURL = strings.TrimSpace(candidateURL)
		if candidateURL == "" {
			continue
		}
		if opts != nil && opts.Context != nil {
			if err := opts.Context.Err(); err != nil {
				return nil, err
			}
		}

		attemptDir, err := os.MkdirTemp(targetDir, fmt.Sprintf("attempt-%d-*", i+1))
		if err != nil {
			return nil, fmt.Errorf("%w: attempt staging: %v", network.ErrLocalFailure, err)
		}

		font.DownloadURL = candidateURL
		if isArchiveFile(candidateURL) {
			font.Path = filepath.Base(candidateURL)
		} else {
			font.Path = createFontFileName(font.Name, font.Variant, candidateURL)
		}

		if applyErr := checksumAppliesToCandidate(&FontFile{
			SHA:                font.SHA,
			DownloadURL:        primaryURL,
			DownloadCandidates: candidates,
		}, candidateURL); applyErr != nil {
			_ = os.RemoveAll(attemptDir)
			lastErr = applyErr
			outcomes = append(outcomes, fmt.Sprintf("%s: %v", network.RedactDownloadURL(candidateURL), applyErr))
			if errors.Is(applyErr, ErrChecksumUnassociated) || errors.Is(applyErr, ErrMalformedChecksum) {
				return nil, applyErr
			}
			continue
		}

		output.GetDebug().State("DownloadAndExtractFont: attempt %d/%d url=%s dir=%s", i+1, len(candidates), candidateURL, attemptDir)
		paths, err := attemptDownloadAndExtract(font, attemptDir, opts)
		if err == nil {
			if i > 0 {
				output.GetDebug().State("DownloadAndExtractFont: succeeded on format candidate %d/%d url=%s", i+1, len(candidates), candidateURL)
			}
			return paths, nil
		}
		lastErr = err
		outcomes = append(outcomes, fmt.Sprintf("%s: %v", network.RedactDownloadURL(candidateURL), err))
		output.GetDebug().State("DownloadAndExtractFont: candidate %d/%d failed: %v", i+1, len(candidates), err)
		_ = os.RemoveAll(attemptDir)

		if errors.Is(err, ErrChecksumMismatch) || errors.Is(err, ErrMalformedChecksum) || errors.Is(err, ErrChecksumUnassociated) {
			return nil, err
		}
		action := network.ClassifyDownloadError(err)
		switch action {
		case network.ActionFailPackage, network.ActionFailLocal:
			return nil, err
		case network.ActionAdvanceCandidate, network.ActionFailCandidate, network.ActionRetrySame, network.ActionRateLimit, network.ActionExternalFallback:
			if i+1 < len(candidates) {
				output.GetDebug().State("DownloadAndExtractFont: trying next format candidate")
			}
		default:
			if i+1 < len(candidates) {
				output.GetDebug().State("DownloadAndExtractFont: trying next format candidate")
			}
		}
	}

	if lastErr == nil {
		return nil, fmt.Errorf("no download URL for %s", font.Name)
	}
	return nil, fmt.Errorf("%w: %s (%s)", ErrCandidatesExhausted, font.Name, strings.Join(outcomes, "; "))
}

// attemptDownloadAndExtract performs one download + optional extract + validation for font.DownloadURL.
func attemptDownloadAndExtract(font *FontFile, targetDir string, opts *DownloadFontOptions) ([]string, error) {
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

	selCtx := ArchiveSelectionContext{}
	if opts != nil {
		selCtx.SourcePrefix = strings.ToLower(strings.TrimSpace(opts.ArchiveSourcePrefix))
		selCtx.FontID = strings.TrimSpace(opts.ArchiveFontID)
	}
	if font != nil {
		selCtx.FontName = font.Name
	}

	if archiveType == ArchiveTypeUnknown {
		// Not an archive: validate it's a real font before returning. This prevents HTML/WAF payloads
		// (or empty/garbage files) being treated as installed fonts later.
		htmlPayload := isProbablyHTML(downloadedPath)
		if err := ValidateFontFile(downloadedPath); err != nil {
			_ = os.Remove(downloadedPath)
			if htmlPayload {
				return nil, fmt.Errorf("download did not return a font file (received HTML; upstream likely served a challenge page): %w", err)
			}
			return nil, fmt.Errorf("download did not return a valid font file: %w", err)
		}
		// Also require sfnt parseability for install.
		if _, err := platform.ExtractFontMetadata(downloadedPath); err != nil {
			_ = os.Remove(downloadedPath)
			if htmlPayload {
				return nil, fmt.Errorf("download did not return a font file (received HTML; upstream likely served a challenge page): %w", err)
			}
			return nil, fmt.Errorf("download did not return a valid font file: %w", err)
		}
		return []string{downloadedPath}, nil
	}

	// It's an archive: ZIP and compressed TAR select before extract; 7Z streams with hard budgets.
	extractDir := filepath.Join(targetDir, "extracted")
	cleanupStaging := func(primary error) error {
		var cleanupNotes []string
		if rerr := os.RemoveAll(extractDir); rerr != nil && !os.IsNotExist(rerr) {
			cleanupNotes = append(cleanupNotes, fmt.Sprintf("remove extract dir: %v", rerr))
		}
		if rerr := os.Remove(downloadedPath); rerr != nil && !os.IsNotExist(rerr) {
			cleanupNotes = append(cleanupNotes, fmt.Sprintf("remove archive: %v", rerr))
		}
		if len(cleanupNotes) == 0 {
			return primary
		}
		return fmt.Errorf("%w (%s)", primary, strings.Join(cleanupNotes, "; "))
	}

	extractedFiles, err := ExtractArchiveWithOptions(downloadedPath, extractDir, &ExtractOptions{
		OnFontFileExtracted: func(done int, total int) {
			if opts != nil && opts.OnExtractProgress != nil {
				opts.OnExtractProgress(done, total)
			}
		},
		Selection: &selCtx,
	})
	if err != nil {
		return nil, cleanupStaging(fmt.Errorf("failed to extract archive: %w", err))
	}

	if rerr := os.Remove(downloadedPath); rerr != nil && !os.IsNotExist(rerr) {
		_ = os.RemoveAll(extractDir)
		return nil, fmt.Errorf("remove archive after extract: %w", rerr)
	}

	if len(extractedFiles) == 0 {
		_ = os.RemoveAll(extractDir)
		return nil, fmt.Errorf("no font files found in archive")
	}

	packageMode := isNerdPackageSource(selCtx.SourcePrefix)

	// Validate extracted files (header + sfnt). Package mode requires every selected file to
	// validate; non-package archives may drop unparseable members.
	valid := make([]string, 0, len(extractedFiles))
	for _, p := range extractedFiles {
		if err := ValidateFontFile(p); err != nil {
			if packageMode {
				return nil, cleanupStaging(fmt.Errorf("package font invalid %q: %w", filepath.Base(p), err))
			}
			_ = os.Remove(p)
			continue
		}
		if _, err := platform.ExtractFontMetadata(p); err != nil {
			if packageMode {
				return nil, cleanupStaging(fmt.Errorf("package font unparseable %q: %w", filepath.Base(p), err))
			}
			_ = os.Remove(p)
			continue
		}
		valid = append(valid, p)
	}
	if len(valid) == 0 {
		_ = os.RemoveAll(extractDir)
		return nil, fmt.Errorf("no valid font files found after extraction (archive contents were not parseable as fonts)")
	}
	if packageMode && len(valid) != len(extractedFiles) {
		return nil, cleanupStaging(fmt.Errorf("package install incomplete: validated %d of %d extracted fonts", len(valid), len(extractedFiles)))
	}

	// Post-extract policy: static vs variable preference (needs files on disk).
	// Path/source selection already ran before ZIP/TAR extract.
	valid = applyArchiveInstallPolicy(valid)
	if len(valid) == 0 {
		_ = os.RemoveAll(extractDir)
		return nil, fmt.Errorf("no font files selected for installation from archive")
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
	sourceOrder := sources.DefaultSourceNamesInPriorityOrder()

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
	r, err := GetRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}
	id, info, _, ok := lookupFontByIDInManifest(r.manifest, fontID)
	if !ok {
		return nil, fmt.Errorf("font not found: %s", fontID)
	}
	return convertFontInfoToFontFiles(info, id)
}

// GetFontByIDCached returns font files for a Font ID using only the on-disk manifest cache (no network refresh, no UI).
// Resolution order matches MatchRepositoryFontByID (source priority, deterministic duplicate-ID winner).
func GetFontByIDCached(fontID string) ([]FontFile, error) {
	if strings.TrimSpace(fontID) == "" {
		return nil, fmt.Errorf("empty font id")
	}
	manifest, err := GetCachedManifest()
	if err != nil {
		return nil, err
	}
	id, info, _, ok := lookupFontByIDInManifest(manifest, fontID)
	if !ok {
		return nil, fmt.Errorf("font not found: %s", fontID)
	}
	return convertFontInfoToFontFiles(info, id)
}

// convertFontInfoToFontFiles converts FontInfo to []FontFile
func convertFontInfoToFontFiles(font FontInfo, fontID string) ([]FontFile, error) {
	var fonts []FontFile
	seenURLs := make(map[string]bool) // Track seen primary URLs to avoid duplicate variants

	// Process each variant using the preserved variant-file mapping
	for _, variantName := range font.Variants {
		var files map[string]string

		// Use variant-specific files if available
		if font.VariantFiles != nil {
			if variantFiles, exists := font.VariantFiles[variantName]; exists {
				files = variantFiles
			}
		}

		// Fallback to general files if variant-specific not found
		if len(files) == 0 {
			files = font.Files
		}

		candidates := downloadCandidateURLs(files)
		if len(candidates) == 0 {
			continue
		}
		downloadURL := candidates[0]

		// Check if we've already processed this primary URL (for duplicate variants)
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
			Name:               font.Name,
			Variant:            variantName,
			Path:               fileName,
			DownloadURL:        downloadURL,
			DownloadCandidates: candidates,
		})
	}
	if len(fonts) == 0 {
		return nil, fmt.Errorf("no valid font files found for %s", fontID)
	}

	return fonts, nil
}

// isArchiveFile checks if a URL points to an archive file
func isArchiveFile(downloadURL string) bool {
	ext := strings.ToLower(filepath.Ext(downloadURL))
	if ext == ".zip" || ext == ".xz" || ext == ".7z" || strings.HasSuffix(strings.ToLower(downloadURL), ".tar.xz") {
		return true
	}
	if u, err := url.Parse(downloadURL); err == nil && strings.Contains(strings.ToLower(u.Path), "/fontfacekit/") {
		// Font Squirrel kits are ZIPs; URLs often have no file extension.
		return true
	}
	return false
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

// isFontFile checks if a file is a font file (desktop containers FontGet may install).
func isFontFile(filename string) bool {
	lower := strings.ToLower(filename)
	return strings.HasSuffix(lower, ".ttf") ||
		strings.HasSuffix(lower, ".otf") ||
		strings.HasSuffix(lower, ".ttc") ||
		strings.HasSuffix(lower, ".otc")
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
