package network

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRunCancellable_NoOutputHang(t *testing.T) {
	name, args := longSleepArgs()
	if _, err := exec.LookPath(name); err != nil {
		t.Skip(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()
	_, err := RunCancellable(ctx, ExecOptions{InactivityTimeout: time.Minute, TerminateWait: 500 * time.Millisecond}, name, args...)
	if err == nil {
		t.Fatal("expected cancel or stall")
	}
}

func TestRunCancellable_InactivityStall(t *testing.T) {
	name, args := longSleepArgs()
	if _, err := exec.LookPath(name); err != nil {
		t.Skip(err)
	}
	dir := t.TempDir()
	progress := filepath.Join(dir, "out.bin")
	_, err := RunCancellable(context.Background(), ExecOptions{
		InactivityTimeout: 300 * time.Millisecond,
		TerminateWait:     500 * time.Millisecond,
		ProgressPath:      progress,
	}, name, args...)
	if err == nil {
		t.Fatal("expected stall")
	}
}

func TestRunCancellable_KillsDescendants(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "alive")
	name, args := descendantMarkerArgs(marker)
	if _, err := exec.LookPath(name); err != nil {
		t.Skip(err)
	}
	_ = os.Remove(marker)
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		_, err := RunCancellable(ctx, ExecOptions{InactivityTimeout: time.Minute, TerminateWait: time.Second}, name, args...)
		errCh <- err
	}()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(marker); err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatal("child never created alive marker")
	}
	cancel()
	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected cancellation error")
		}
	case <-time.After(8 * time.Second):
		t.Fatal("runner did not return after cancel")
	}
	// Child should stop updating the marker; remove and ensure it is not recreated.
	_ = os.Remove(marker)
	time.Sleep(800 * time.Millisecond)
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("descendant still alive after cancel (marker recreated)")
	}
}

func descendantMarkerArgs(marker string) (string, []string) {
	if runtime.GOOS == "windows" {
		script := fmt.Sprintf(`while ($true) { Set-Content -LiteralPath '%s' -Value 'alive' -Encoding ascii; Start-Sleep -Seconds 1 }`, marker)
		return "powershell", []string{"-NoProfile", "-NonInteractive", "-Command", script}
	}
	script := `touch "` + marker + `"; while true; do touch "` + marker + `"; sleep 1; done`
	return "sh", []string{"-c", script}
}

func longSleepArgs() (string, []string) {
	if runtime.GOOS == "windows" {
		return "timeout", []string{"/t", "30", "/nobreak"}
	}
	return "sleep", []string{"30"}
}

func TestLimitedBufferBoundsOutput(t *testing.T) {
	var buf limitedBuffer
	buf.max = 8
	n, err := buf.Write([]byte("abcdefghijklmnop"))
	if err != nil || n != 16 {
		t.Fatalf("write n=%d err=%v", n, err)
	}
	if len(buf.Bytes()) != 8 {
		t.Fatalf("captured %d", len(buf.Bytes()))
	}
}

func TestRunCancellable_ReleasedOnCancel(t *testing.T) {
	name, args := longSleepArgs()
	if _, err := exec.LookPath(name); err != nil {
		t.Skip(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		_, err := RunCancellable(ctx, ExecOptions{InactivityTimeout: time.Minute, TerminateWait: 500 * time.Millisecond}, name, args...)
		errCh <- err
	}()
	time.Sleep(150 * time.Millisecond)
	cancel()
	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected cancellation error")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("process did not stop after cancel")
	}
}

func TestScrapeHTTPStatus(t *testing.T) {
	if got := scrapeHTTPStatus([]byte("FONTGET_HTTP_STATUS=410"), nil); got != "410" {
		t.Fatalf("got %q", got)
	}
	if got := scrapeHTTPStatus([]byte("HTTP/1.1 404 Not Found"), nil); got != "404" {
		t.Fatalf("got %q", got)
	}
	if got := scrapeHTTPStatus(nil, errWith("ERROR 429: Too Many Requests")); got != "429" {
		t.Fatalf("got %q", got)
	}
	_ = strconv.Itoa(1)
	_ = strings.TrimSpace("x")
}

type errWith string

func (e errWith) Error() string { return string(e) }
