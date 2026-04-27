package network

import (
	"bytes"
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
}

func (e *FallbackAttemptError) Error() string {
	if e == nil {
		return "download fallback failed"
	}
	if len(e.attempt) > 0 {
		return fmt.Sprintf("download fallback failed for %s (%s)", e.URL, strings.Join(e.attempt, "; "))
	}
	return fmt.Sprintf("download fallback failed for %s", e.URL)
}

// DownloadWithFallbacks attempts to download the URL to targetPath using optional external tools.
// It is capability-first: tools are only attempted if found, but a found tool that fails will not
// stop the chain. On success, the returned report includes all steps (skipped, failed, and ok).
func DownloadWithFallbacks(url, targetPath string, opts DownloadFallbackOptions) (*DownloadFallbackReport, error) {
	return downloadWithFallbacks(execRunner{}, url, targetPath, opts)
}

func downloadWithFallbacks(runner CommandRunner, url, targetPath string, opts DownloadFallbackOptions) (*DownloadFallbackReport, error) {
	rep := &DownloadFallbackReport{}
	var compact []string

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
		// Cheap WAF/HTML detection: many challenges return an HTML page.
		f, err := os.Open(targetPath)
		if err != nil {
			return nil // can't inspect; keep it permissive
		}
		defer f.Close()
		var buf [512]byte
		n, _ := f.Read(buf[:])
		b := bytes.TrimSpace(bytes.ToLower(buf[:n]))
		if bytes.HasPrefix(b, []byte("<!doctype html")) || bytes.HasPrefix(b, []byte("<html")) {
			return fmt.Errorf("output looks like HTML (likely upstream challenge page)")
		}
		// If this download was intended to be a ZIP, require ZIP magic so a 200 HTML/challenge body
		// can't be treated as a successful archive download.
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

	// curl
	if curlPath, err := runner.LookPath("curl"); err != nil || curlPath == "" {
		rep.Steps = append(rep.Steps, DownloadFallbackStep{Tool: "curl", Result: "skipped", Detail: "not found in PATH"})
	} else if status, err := runCurl(runner, curlPath, url, targetPath, opts); err == nil {
		if vErr := validateDownloadedFile(); vErr == nil {
			detail := ""
			if status != "" {
				detail = "http_status=" + status
			}
			rep.Steps = append(rep.Steps, DownloadFallbackStep{Tool: "curl", Path: curlPath, Result: "ok", Detail: detail})
			return rep, nil
		} else {
			appendFailed("curl", curlPath, vErr.Error())
		}
	} else {
		appendFailed("curl", curlPath, err.Error())
	}

	// wget
	if wgetPath, err := runner.LookPath("wget"); err != nil || wgetPath == "" {
		rep.Steps = append(rep.Steps, DownloadFallbackStep{Tool: "wget", Result: "skipped", Detail: "not found in PATH"})
	} else if err := runWget(runner, wgetPath, url, targetPath, opts); err == nil {
		if vErr := validateDownloadedFile(); vErr == nil {
			rep.Steps = append(rep.Steps, DownloadFallbackStep{Tool: "wget", Path: wgetPath, Result: "ok"})
			return rep, nil
		} else {
			appendFailed("wget", wgetPath, vErr.Error())
		}
	} else {
		appendFailed("wget", wgetPath, err.Error())
	}

	// pwsh
	if pwshPath, err := runner.LookPath("pwsh"); err != nil || pwshPath == "" {
		rep.Steps = append(rep.Steps, DownloadFallbackStep{Tool: "pwsh", Result: "skipped", Detail: "not found in PATH"})
	} else if err := runPowerShell(runner, pwshPath, url, targetPath, opts); err == nil {
		if vErr := validateDownloadedFile(); vErr == nil {
			rep.Steps = append(rep.Steps, DownloadFallbackStep{Tool: "pwsh", Path: pwshPath, Result: "ok"})
			return rep, nil
		} else {
			appendFailed("pwsh", pwshPath, vErr.Error())
		}
	} else {
		appendFailed("pwsh", pwshPath, err.Error())
	}

	// powershell
	if psPath, err := runner.LookPath("powershell"); err != nil || psPath == "" {
		rep.Steps = append(rep.Steps, DownloadFallbackStep{Tool: "powershell", Result: "skipped", Detail: "not found in PATH"})
	} else if err := runPowerShell(runner, psPath, url, targetPath, opts); err == nil {
		if vErr := validateDownloadedFile(); vErr == nil {
			rep.Steps = append(rep.Steps, DownloadFallbackStep{Tool: "powershell", Path: psPath, Result: "ok"})
			return rep, nil
		} else {
			appendFailed("powershell", psPath, vErr.Error())
		}
	} else {
		appendFailed("powershell", psPath, err.Error())
	}

	if len(rep.Steps) == 0 {
		return rep, errors.New("no external download tools available")
	}
	return rep, &FallbackAttemptError{URL: url, Report: rep, attempt: compact}
}

func runCurl(runner CommandRunner, curlPath, url, targetPath string, opts DownloadFallbackOptions) (finalStatus string, _ error) {
	// We explicitly capture the final HTTP status code because curl considers 202 a success.
	// Some upstream WAFs respond with 202 + an empty/challenge body.
	const writeOutStatusPrefix = "FONTGET_HTTP_STATUS="
	reStatus := regexp.MustCompile(writeOutStatusPrefix + `(\d{3})`)

	baseArgs := []string{
		"-L",
		// Treat 4xx/5xx as failures. (202/3xx are not failures in curl, so we validate output separately.)
		"--fail",
		"--silent",
		"--show-error",
		// Emit final status code to stdout so we can treat non-200 as failure even when curl exits 0.
		"--write-out", writeOutStatusPrefix + "%{http_code}",
	}
	if opts.UserAgent != "" {
		baseArgs = append(baseArgs, "-A", opts.UserAgent)
	}
	for k, v := range opts.Headers {
		baseArgs = append(baseArgs, "-H", fmt.Sprintf("%s: %s", k, v))
	}

	// Bounded retry for transient statuses (helps with upstream WAF challenges).
	// Keep total added time small to avoid hangs in interactive UI.
	maxAttempts := 3
	backoff := 250 * time.Millisecond
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		args := append([]string{}, baseArgs...)
		args = append(args, "-o", targetPath, url)

		out, err := runner.CombinedOutput(curlPath, args...)
		if err != nil {
			// If curl failed but still printed a status, we may be able to classify it.
			m := reStatus.FindStringSubmatch(string(out))
			if len(m) == 2 {
				finalStatus = m[1]
			}
			return "", fmt.Errorf("%s", normalizeToolError(out, err))
		}

		m := reStatus.FindStringSubmatch(string(out))
		if len(m) == 2 {
			finalStatus = m[1]
		}
		if finalStatus == "" || finalStatus == "200" {
			return finalStatus, nil
		}

		code, _ := strconv.Atoi(finalStatus)
		shouldRetry := code == 202 || code == 429 || (code >= 500 && code <= 599)
		if code == 403 {
			return finalStatus, fmt.Errorf("unexpected HTTP status %s", finalStatus)
		}
		if !shouldRetry || attempt == maxAttempts {
			return finalStatus, fmt.Errorf("unexpected HTTP status %s", finalStatus)
		}

		// jitter in [0, 200ms]
		j := time.Duration(rng.Intn(200)) * time.Millisecond
		time.Sleep(backoff + j)
		backoff *= 2
	}

	return finalStatus, fmt.Errorf("unexpected HTTP status %s", finalStatus)
}

func runWget(runner CommandRunner, wgetPath, url, targetPath string, opts DownloadFallbackOptions) error {
	args := []string{
		"-q",
		"-O", targetPath,
	}
	if opts.UserAgent != "" {
		args = append(args, "--user-agent", opts.UserAgent)
	}
	for k, v := range opts.Headers {
		args = append(args, "--header", fmt.Sprintf("%s: %s", k, v))
	}
	args = append(args, url)

	out, err := runner.CombinedOutput(wgetPath, args...)
	if err != nil {
		return fmt.Errorf("%s", normalizeToolError(out, err))
	}
	return nil
}

func runPowerShell(runner CommandRunner, psPath, url, targetPath string, opts DownloadFallbackOptions) error {
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
		fmt.Sprintf("Invoke-WebRequest -Uri $u -OutFile $p -Headers $h -UserAgent '%s' -ErrorAction Stop | Out-Null", esc(ua)),
	}, "; ")

	args := []string{"-NoProfile", "-NonInteractive", "-Command", script}
	out, err := runner.CombinedOutput(psPath, args...)
	if err != nil {
		return fmt.Errorf("%s", normalizeToolError(out, err))
	}
	return nil
}

func normalizeToolError(out []byte, err error) string {
	msg := strings.TrimSpace(string(bytes.TrimSpace(out)))
	if msg != "" {
		return msg
	}
	return err.Error()
}
