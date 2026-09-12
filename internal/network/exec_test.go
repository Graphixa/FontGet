package network

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
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
	name, args := descendantSleepArgs()
	if _, err := exec.LookPath(name); err != nil {
		t.Skip(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		_, err := RunCancellable(ctx, ExecOptions{InactivityTimeout: time.Minute, TerminateWait: 500 * time.Millisecond}, name, args...)
		errCh <- err
	}()
	time.Sleep(200 * time.Millisecond)
	cancel()
	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected cancellation error")
		}
	case <-time.After(8 * time.Second):
		t.Fatal("descendant process tree did not stop after cancel")
	}
}

func descendantSleepArgs() (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/c", "timeout", "/t", "30", "/nobreak"}
	}
	return "sh", []string{"-c", "sleep 30"}
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
