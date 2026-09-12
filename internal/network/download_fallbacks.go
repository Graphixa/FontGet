package network

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type CommandRunner interface {
	LookPath(file string) (string, error)
	CombinedOutput(name string, args ...string) ([]byte, error)
}

type execRunner struct{}

func (execRunner) LookPath(file string) (string, error) { return exec.LookPath(file) }
func (execRunner) CombinedOutput(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.CombinedOutput()
}

// DownloadFallbackOptions controls how external download tools are invoked.
type DownloadFallbackOptions struct {
	UserAgent string
	Headers   map[string]string
	// Context cancels tool execution, host waits and retry backoff. Nil uses Background.
	Context context.Context
	// MaxAttempts bounds per-tool HTTP retries. Zero means 3. Use 1 when native retries already ran.
	MaxAttempts int
	// Exec carries inactivity, terminate-wait and output caps. Zero values use documented defaults.
	Exec ExecOptions
}

func (opts DownloadFallbackOptions) ctx() context.Context {
	if opts.Context != nil {
		return opts.Context
	}
	return context.Background()
}

func (opts DownloadFallbackOptions) maxAttempts() int {
	if opts.MaxAttempts > 0 {
		return opts.MaxAttempts
	}
	return 3
}

func isZipMagic(b []byte) bool {
	if len(b) < 4 {
		return false
	}
	return b[0] == 'P' && b[1] == 'K' && ((b[2] == 3 && b[3] == 4) || (b[2] == 5 && b[3] == 6) || (b[2] == 7 && b[3] == 8))
}

// DownloadFallbackStep records one candidate tool in the fallback chain.
type DownloadFallbackStep struct {
	Tool   string // logical name: curl, wget, pwsh, powershell
	Path   string // resolved binary path, or empty if not installed
	Result string // skipped | failed | ok
	Detail string // error text, "not found in PATH", or empty when ok
}

// DownloadFallbackReport describes the full fallback attempt sequence.
type DownloadFallbackReport struct {
	Steps []DownloadFallbackStep
}

// UsedTool returns the logical tool name and path that succeeded, if any.
func (r *DownloadFallbackReport) UsedTool() (name, path string) {
	if r == nil {
		return "", ""
	}
	for i := len(r.Steps) - 1; i >= 0; i-- {
		s := r.Steps[i]
		if s.Result == "ok" {
			return s.Tool, s.Path
		}
	}
	return "", ""
}

// FallbackAttemptError summarizes failures across the fallback chain.
type FallbackAttemptError struct {
	URL     string
	Report  *DownloadFallbackReport
	attempt []string
	cause   error
}

func (e *FallbackAttemptError) Error() string {
	if e == nil {
		return "download fallback failed"
	}
	if len(e.attempt) > 0 {
		return fmt.Sprintf("download fallback failed for %s (%s)", RedactDownloadURL(e.URL), strings.Join(e.attempt, "; "))
	}
	return fmt.Sprintf("download fallback failed for %s", RedactDownloadURL(e.URL))
}

func (e *FallbackAttemptError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

// DownloadWithFallbacks attempts to download the URL to targetPath using optional external tools.
// It is capability-first: tools are only attempted if found, but a found tool that fails will not
// stop the chain unless the failure is a terminal HTTP status (404/410/other permanent 4xx).
func DownloadWithFallbacks(url, targetPath string, opts DownloadFallbackOptions) (*DownloadFallbackReport, error) {
	return downloadWithFallbacks(execRunner{}, url, targetPath, opts)
}

func downloadWithFallbacks(runner CommandRunner, url, targetPath string, opts DownloadFallbackOptions) (*DownloadFallbackReport, error) {
	rep := &DownloadFallbackReport{}
	var compact []string
	ctx := opts.ctx()
	opts.Exec.ProgressPath = targetPath

	appendFailed := func(tool, bin, msg string) {
		compact = append(compact, tool+": "+msg)
		rep.Steps = append(rep.Steps, DownloadFallbackStep{
			Tool: tool, Path: bin, Result: "failed", Detail: msg,
		})
	}

	validateDownloadedFile := func() error {
		fi, err := os.Stat(targetPath)
		if err != nil {
			return fmt.Errorf("output file missing: %v", err)
		}
		if fi.Size() <= 0 {
			return fmt.Errorf("output file is empty")
		}
		f, err := os.Open(targetPath)
		if err != nil {
			return nil
		}
		defer f.Close()
		var buf [512]byte
		n, _ := f.Read(buf[:])
		b := bytes.TrimSpace(bytes.ToLower(buf[:n]))
		if bytes.HasPrefix(b, []byte("<!doctype html")) || bytes.HasPrefix(b, []byte("<html")) {
			return fmt.Errorf("output looks like HTML (likely upstream challenge page)")
		}
		expectZip := strings.HasSuffix(strings.ToLower(targetPath), ".zip")
		if !expectZip {
			accept := strings.ToLower(strings.TrimSpace(opts.Headers["Accept"]))
			expectZip = strings.Contains(accept, "application/zip")
		}
		if expectZip {
			if !isZipMagic(buf[:n]) {
				return fmt.Errorf("output is not a ZIP archive (missing PK header)")
			}
		}
		return nil
	}

	stopChain := func(err error) (*DownloadFallbackReport, error) {
		action := ClassifyDownloadError(err)
		cause := err
		if action == ActionAdvanceCandidate || action == ActionFailCandidate || action == ActionFailPackage || action == ActionFailLocal || action == ActionRateLimit {
			return rep, &FallbackAttemptError{URL: url, Report: rep, attempt: compact, cause: cause}
		}
		return nil, err
	}

	tryTool := func(logical, bin string, run func() (string, error)) (ok bool, terminal error) {
		status, err := run()
		if err != nil {
			appendFailed(logical, bin, err.Error())
			action := ClassifyDownloadError(err)
			var httpErr *HTTPStatusError
			if errors.As(err, &httpErr) {
				action = ClassifyHTTPStatus(httpErr.StatusCode, false)
			} else if code, ok := parseHTTPStatus(status); ok {
				action = ClassifyHTTPStatus(code, false)
				err = NewHTTPStatusError(code, url, 0)
			}
			switch action {
			case ActionAdvanceCandidate, ActionFailCandidate, ActionFailPackage, ActionFailLocal, ActionRateLimit:
				return false, err
			default:
				return false, nil
			}
		}
		if vErr := validateDownloadedFile(); vErr != nil {
			appendFailed(logical, bin, vErr.Error())
			return false, nil
		}
		detail := ""
		if status != "" {
			detail = "http_status=" + status
		}
		rep.Steps = append(rep.Steps, DownloadFallbackStep{Tool: logical, Path: bin, Result: "ok", Detail: detail})
		return true, nil
	}

	if curlPath, err := runner.LookPath("curl"); err != nil || curlPath == "" {
		rep.Steps = append(rep.Steps, DownloadFallbackStep{Tool: "curl", Result: "skipped", Detail: "not found in PATH"})
	} else {
		ok, term := tryTool("curl", curlPath, func() (string, error) {
			return runCurl(ctx, runner, curlPath, url, targetPath, opts)
		})
		if ok {
			return rep, nil
		}
		if term != nil {
			return stopChain(term)
		}
	}

	if err := ctx.Err(); err != nil {
		return rep, err
	}

	if wgetPath, err := runner.LookPath("wget"); err != nil || wgetPath == "" {
		rep.Steps = append(rep.Steps, DownloadFallbackStep{Tool: "wget", Result: "skipped", Detail: "not found in PATH"})
	} else {
		ok, term := tryTool("wget", wgetPath, func() (string, error) {
			return runWget(ctx, runner, wgetPath, url, targetPath, opts)
		})
		if ok {
			return rep, nil
		}
		if term != nil {
			return stopChain(term)
		}
	}

	if err := ctx.Err(); err != nil {
		return rep, err
	}

	if pwshPath, err := runner.LookPath("pwsh"); err != nil || pwshPath == "" {
		rep.Steps = append(rep.Steps, DownloadFallbackStep{Tool: "pwsh", Result: "skipped", Detail: "not found in PATH"})
	} else {
		ok, term := tryTool("pwsh", pwshPath, func() (string, error) {
			return runPowerShell(ctx, runner, pwshPath, url, targetPath, opts)
		})
		if ok {
			return rep, nil
		}
		if term != nil {
			return stopChain(term)
		}
	}

	if err := ctx.Err(); err != nil {
		return rep, err
	}

	if psPath, err := runner.LookPath("powershell"); err != nil || psPath == "" {
		rep.Steps = append(rep.Steps, DownloadFallbackStep{Tool: "powershell", Result: "skipped", Detail: "not found in PATH"})
	} else {
		ok, term := tryTool("powershell", psPath, func() (string, error) {
			return runPowerShell(ctx, runner, psPath, url, targetPath, opts)
		})
		if ok {
			return rep, nil
		}
		if term != nil {
			return stopChain(term)
		}
	}

	if len(rep.Steps) == 0 {
		return rep, errors.New("no external download tools available")
	}
	return rep, &FallbackAttemptError{URL: url, Report: rep, attempt: compact}
}

func runTool(ctx context.Context, runner CommandRunner, opts DownloadFallbackOptions, name string, args ...string) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if cr, ok := runner.(ContextCommandRunner); ok {
		return cr.CombinedOutputContext(ctx, opts.Exec, name, args...)
	}
	return runner.CombinedOutput(name, args...)
}

func runCurl(ctx context.Context, runner CommandRunner, curlPath, url, targetPath string, opts DownloadFallbackOptions) (finalStatus string, _ error) {
	const writeOutStatusPrefix = "FONTGET_HTTP_STATUS="
	reStatus := regexp.MustCompile(writeOutStatusPrefix + `(\d{3})`)

	baseArgs := []string{
		"-L",
		"--fail",
		"--silent",
		"--show-error",
		"--write-out", writeOutStatusPrefix + "%{http_code}",
	}
	if opts.UserAgent != "" {
		baseArgs = append(baseArgs, "-A", opts.UserAgent)
	}
	for k, v := range opts.Headers {
		baseArgs = append(baseArgs, "-H", fmt.Sprintf("%s: %s", k, v))
	}

	maxAttempts := opts.maxAttempts()
	backoff := 250 * time.Millisecond
	// #nosec G404 -- non-cryptographic jitter for download retry backoff only (not security-sensitive).
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return finalStatus, err
		}
		args := append([]string{}, baseArgs...)
		args = append(args, "-o", targetPath, url)

		out, err := runTool(ctx, runner, opts, curlPath, args...)
		if err != nil {
			finalStatus = matchHTTPStatus(reStatus, out)
			if code, ok := parseHTTPStatus(finalStatus); ok {
				action := ClassifyHTTPStatus(code, false)
				httpErr := NewHTTPStatusError(code, url, 0)
				switch action {
				case ActionAdvanceCandidate, ActionFailCandidate, ActionFailPackage, ActionFailLocal:
					return finalStatus, httpErr
				case ActionRateLimit:
					return finalStatus, httpErr
				case ActionRetrySame:
					lastErr = httpErr
					if attempt == maxAttempts {
						return finalStatus, httpErr
					}
					if err := sleepCtx(ctx, backoff+time.Duration(rng.Intn(200))*time.Millisecond); err != nil {
						return finalStatus, err
					}
					backoff *= 2
					continue
				}
			}
			if isCancelErr(err) {
				return finalStatus, err
			}
			return finalStatus, fmt.Errorf("%s", normalizeToolError(out, err))
		}

		finalStatus = matchHTTPStatus(reStatus, out)
		if finalStatus == "" || finalStatus == "200" {
			return finalStatus, nil
		}

		code, _ := strconv.Atoi(finalStatus)
		httpErr := NewHTTPStatusError(code, url, 0)
		action := ClassifyHTTPStatus(code, false)
		switch action {
		case ActionSuccess:
			return finalStatus, nil
		case ActionAdvanceCandidate, ActionFailCandidate, ActionFailPackage, ActionFailLocal, ActionRateLimit:
			return finalStatus, httpErr
		case ActionRetrySame:
			lastErr = httpErr
			if attempt == maxAttempts {
				return finalStatus, httpErr
			}
			if err := sleepCtx(ctx, backoff+time.Duration(rng.Intn(200))*time.Millisecond); err != nil {
				return finalStatus, err
			}
			backoff *= 2
		default:
			return finalStatus, httpErr
		}
	}

	if lastErr != nil {
		return finalStatus, lastErr
	}
	return finalStatus, fmt.Errorf("unexpected HTTP status %s", finalStatus)
}

func matchHTTPStatus(re *regexp.Regexp, out []byte) string {
	m := re.FindStringSubmatch(string(out))
	if len(m) == 2 {
		return m[1]
	}
	return ""
}

func parseHTTPStatus(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	code, err := strconv.Atoi(s)
	if err != nil || code < 100 || code > 599 {
		return 0, false
	}
	return code, true
}

func runWget(ctx context.Context, runner CommandRunner, wgetPath, url, targetPath string, opts DownloadFallbackOptions) (string, error) {
	args := []string{
		"-q",
		"-O", targetPath,
		"--server-response",
	}
	if opts.UserAgent != "" {
		args = append(args, "--user-agent", opts.UserAgent)
	}
	for k, v := range opts.Headers {
		args = append(args, "--header", fmt.Sprintf("%s: %s", k, v))
	}
	args = append(args, url)

	out, err := runTool(ctx, runner, opts, wgetPath, args...)
	status := scrapeHTTPStatus(out, err)
	if err != nil {
		if isCancelErr(err) {
			return status, err
		}
		if code, ok := parseHTTPStatus(status); ok {
			return status, NewHTTPStatusError(code, url, 0)
		}
		return status, fmt.Errorf("%s", normalizeToolError(out, err))
	}
	return status, nil
}

func runPowerShell(ctx context.Context, runner CommandRunner, psPath, url, targetPath string, opts DownloadFallbackOptions) (string, error) {
	ua := opts.UserAgent
	if ua == "" {
		ua = "Mozilla/5.0"
	}

	esc := func(s string) string { return strings.ReplaceAll(s, "'", "''") }
	var headerPairs []string
	for k, v := range opts.Headers {
		headerPairs = append(headerPairs, fmt.Sprintf("'%s'='%s'", esc(k), esc(v)))
	}
	headerLiteral := "@{}"
	if len(headerPairs) > 0 {
		headerLiteral = "@{" + strings.Join(headerPairs, ";") + "}"
	}

	script := strings.Join([]string{
		"$ProgressPreference='SilentlyContinue'",
		fmt.Sprintf("$u='%s'", esc(url)),
		fmt.Sprintf("$p='%s'", esc(targetPath)),
		fmt.Sprintf("$h=%s", headerLiteral),
		"try {",
		fmt.Sprintf("  Invoke-WebRequest -Uri $u -OutFile $p -Headers $h -UserAgent '%s' -ErrorAction Stop | Out-Null", esc(ua)),
		"  Write-Output 'FONTGET_HTTP_STATUS=200'",
		"} catch {",
		"  $code = 0",
		"  if ($_.Exception.Response -ne $null) { $code = [int]$_.Exception.Response.StatusCode }",
		"  Write-Output ('FONTGET_HTTP_STATUS=' + $code)",
		"  throw",
		"}",
	}, "; ")

	args := []string{"-NoProfile", "-NonInteractive", "-Command", script}
	out, err := runTool(ctx, runner, opts, psPath, args...)
	status := scrapeHTTPStatus(out, err)
	if err != nil {
		if isCancelErr(err) {
			return status, err
		}
		if code, ok := parseHTTPStatus(status); ok && code != 0 {
			return status, NewHTTPStatusError(code, url, 0)
		}
		return status, fmt.Errorf("%s", normalizeToolError(out, err))
	}
	return status, nil
}

var httpStatusScrapers = []*regexp.Regexp{
	regexp.MustCompile(`(?i)FONTGET_HTTP_STATUS=(\d{3})`),
	regexp.MustCompile(`(?i)HTTP[/\d.]*\s+(\d{3})\b`),
	regexp.MustCompile(`(?i)\bERROR\s+(\d{3})\b`),
	regexp.MustCompile(`(?i)\bstatus(?:\s+code)?[=:\s]+(\d{3})\b`),
}

func scrapeHTTPStatus(out []byte, err error) string {
	blob := string(out)
	if err != nil {
		blob += "\n" + err.Error()
	}
	for _, re := range httpStatusScrapers {
		if m := re.FindStringSubmatch(blob); len(m) == 2 {
			return m[1]
		}
	}
	return ""
}

func sleepCtx(ctx context.Context, d time.Duration) error {
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

func normalizeToolError(out []byte, err error) string {
	msg := strings.TrimSpace(string(bytes.TrimSpace(out)))
	if msg != "" {
		return msg
	}
	return err.Error()
}
