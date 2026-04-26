package network

import (
	"errors"
	"os"
	"strings"
	"testing"
)

type fakeRunner struct {
	paths   map[string]string
	results map[string]fakeResult
	calls   []string
}

type fakeResult struct {
	out string
	err error
}

func (r *fakeRunner) LookPath(file string) (string, error) {
	if p, ok := r.paths[file]; ok && p != "" {
		return p, nil
	}
	return "", errors.New("not found")
}

func (r *fakeRunner) CombinedOutput(name string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, name+" "+strings.Join(args, " "))
	if res, ok := r.results[name]; ok {
		// Simulate successful tools creating the output file.
		// Our production code validates that the targetPath exists and is non-empty.
		if res.err == nil {
			var outPath string
			for i := 0; i < len(args)-1; i++ {
				// curl: -o <path>
				if args[i] == "-o" && i+1 < len(args) {
					outPath = args[i+1]
					break
				}
				// wget: -O <path>
				if args[i] == "-O" && i+1 < len(args) {
					outPath = args[i+1]
					break
				}
			}
			if outPath != "" {
				_ = os.WriteFile(outPath, []byte("dummy payload"), 0644)
			}
		}
		return []byte(res.out), res.err
	}
	return []byte("no result configured"), errors.New("failed")
}

func TestDownloadWithFallbacks_NoTools(t *testing.T) {
	r := &fakeRunner{
		paths:   map[string]string{},
		results: map[string]fakeResult{},
	}
	rep, err := downloadWithFallbacks(r, "https://example.com/file.zip", "/tmp/file.zip", DownloadFallbackOptions{UserAgent: "ua"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "download fallback failed") {
		t.Fatalf("unexpected error: %v", err)
	}
	if rep == nil || len(rep.Steps) == 0 {
		t.Fatalf("expected report with steps, got rep=%v", rep)
	}
}

func TestDownloadWithFallbacks_CurlSuccess(t *testing.T) {
	r := &fakeRunner{
		paths: map[string]string{
			"curl": "/usr/bin/curl",
		},
		results: map[string]fakeResult{
			"/usr/bin/curl": {out: "", err: nil},
		},
	}
	rep, err := downloadWithFallbacks(r, "https://example.com/file.zip", "/tmp/file.zip", DownloadFallbackOptions{
		UserAgent: "ua",
		Headers:   map[string]string{"Accept": "*/*"},
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	name, path := rep.UsedTool()
	if name != "curl" || path != "/usr/bin/curl" {
		t.Fatalf("UsedTool: got %q %q", name, path)
	}
	if len(r.calls) != 1 || !strings.HasPrefix(r.calls[0], "/usr/bin/curl ") {
		t.Fatalf("expected curl call, got %#v", r.calls)
	}
}

func TestDownloadWithFallbacks_CurlFails_WgetSucceeds(t *testing.T) {
	r := &fakeRunner{
		paths: map[string]string{
			"curl": "/usr/bin/curl",
			"wget": "/usr/bin/wget",
		},
		results: map[string]fakeResult{
			"/usr/bin/curl": {out: "curl failed", err: errors.New("exit 22")},
			"/usr/bin/wget": {out: "", err: nil},
		},
	}
	rep, err := downloadWithFallbacks(r, "https://example.com/file.zip", "/tmp/file.zip", DownloadFallbackOptions{UserAgent: "ua"})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	name, _ := rep.UsedTool()
	if name != "wget" {
		t.Fatalf("expected wget success, got %q", name)
	}
	if len(r.calls) != 2 {
		t.Fatalf("expected 2 calls (curl then wget), got %#v", r.calls)
	}
	if !strings.HasPrefix(r.calls[0], "/usr/bin/curl ") || !strings.HasPrefix(r.calls[1], "/usr/bin/wget ") {
		t.Fatalf("unexpected call order: %#v", r.calls)
	}
}
