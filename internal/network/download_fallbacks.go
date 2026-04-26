package network

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
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
		return nil
	}

	// curl
	if curlPath, err := runner.LookPath("curl"); err != nil || curlPath == "" {
		rep.Steps = append(rep.Steps, DownloadFallbackStep{Tool: "curl", Result: "skipped", Detail: "not found in PATH"})
	} else if err := runCurl(runner, curlPath, url, targetPath, opts); err == nil {
		if vErr := validateDownloadedFile(); vErr == nil {
			rep.Steps = append(rep.Steps, DownloadFallbackStep{Tool: "curl", Path: curlPath, Result: "ok"})
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

func runCurl(runner CommandRunner, curlPath, url, targetPath string, opts DownloadFallbackOptions) error {
	args := []string{
		"-L",
		// Treat 4xx/5xx as failures. (202/3xx are not failures in curl, so we validate output separately.)
		"--fail",
		"--silent",
		"--show-error",
	}
	if opts.UserAgent != "" {
		args = append(args, "-A", opts.UserAgent)
	}
	for k, v := range opts.Headers {
		args = append(args, "-H", fmt.Sprintf("%s: %s", k, v))
	}
	args = append(args, "-o", targetPath, url)

	out, err := runner.CombinedOutput(curlPath, args...)
	if err != nil {
		return fmt.Errorf("%s", normalizeToolError(out, err))
	}
	return nil
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
