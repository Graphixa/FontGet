package network

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

const (
	// DefaultExternalInactivityTimeout is the stall bound when the target file does not grow.
	// Measured from process start until first byte progress, then reset by further size growth.
	// Log chatter is not progress. Tests may override via ExecOptions.InactivityTimeout.
	DefaultExternalInactivityTimeout = 120 * time.Second
	// DefaultExternalTerminateWait is how long we wait after cancellation before forced kill.
	DefaultExternalTerminateWait = 5 * time.Second
	// DefaultMaxCapturedOutput bounds retained stdout/stderr from an external downloader.
	DefaultMaxCapturedOutput = 256 << 10
	// defaultWaitAfterKill bounds how long we wait for Wait() after forced termination.
	defaultWaitAfterKill = 10 * time.Second
)

// ExecOptions controls cancellable external process execution.
type ExecOptions struct {
	InactivityTimeout time.Duration
	TerminateWait     time.Duration
	MaxOutputBytes    int
	// ProgressPath, when set, is stat'd to detect downloaded-byte progress (file size growth).
	ProgressPath string
}

func (o ExecOptions) inactivity() time.Duration {
	if o.InactivityTimeout > 0 {
		return o.InactivityTimeout
	}
	return DefaultExternalInactivityTimeout
}

func (o ExecOptions) terminateWait() time.Duration {
	if o.TerminateWait > 0 {
		return o.TerminateWait
	}
	return DefaultExternalTerminateWait
}

func (o ExecOptions) maxOutput() int {
	if o.MaxOutputBytes > 0 {
		return o.MaxOutputBytes
	}
	return DefaultMaxCapturedOutput
}

// ContextCommandRunner is an optional CommandRunner extension used by production execution.
type ContextCommandRunner interface {
	CommandRunner
	CombinedOutputContext(ctx context.Context, opts ExecOptions, name string, args ...string) ([]byte, error)
}

func (execRunner) CombinedOutputContext(ctx context.Context, opts ExecOptions, name string, args ...string) ([]byte, error) {
	return RunCancellable(ctx, opts, name, args...)
}

// RunCancellable starts name with args (no shell), waits until completion, cancellation, or stall.
// Cancellation is owned entirely here: we do not use exec.CommandContext so Go's default
// Process.Kill cannot race process-tree teardown that needs the parent PID.
func RunCancellable(ctx context.Context, opts ExecOptions, name string, args ...string) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	cmd := exec.Command(name, args...)
	prepareProcessGroup(cmd)

	var out limitedBuffer
	out.max = opts.maxOutput()

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		_ = stdoutR.Close()
		_ = stdoutW.Close()
		return nil, err
	}
	cmd.Stdout = stdoutW
	cmd.Stderr = stderrW

	var copyWG sync.WaitGroup
	copyWG.Add(2)
	go func() {
		defer copyWG.Done()
		_, _ = io.Copy(&out, stdoutR)
		_ = stdoutR.Close()
	}()
	go func() {
		defer copyWG.Done()
		_, _ = io.Copy(&out, stderrR)
		_ = stderrR.Close()
	}()

	if err := cmd.Start(); err != nil {
		_ = stdoutW.Close()
		_ = stderrW.Close()
		copyWG.Wait()
		return out.Bytes(), err
	}
	_ = stdoutW.Close()
	_ = stderrW.Close()

	done := make(chan error, 1)
	go func() {
		waitErr := cmd.Wait()
		copyWG.Wait()
		done <- waitErr
	}()

	stallCtx, stallCancel := context.WithCancel(ctx)
	defer stallCancel()
	if opts.ProgressPath != "" {
		go watchDownloadProgress(stallCtx, stallCancel, opts.ProgressPath, opts.inactivity())
	} else {
		go func() {
			timer := time.NewTimer(opts.inactivity())
			defer timer.Stop()
			select {
			case <-stallCtx.Done():
			case <-timer.C:
				stallCancel()
			}
		}()
	}

	select {
	case err := <-done:
		return out.Bytes(), err
	case <-stallCtx.Done():
		cause := stallCtx.Err()
		if errors.Is(ctx.Err(), context.Canceled) {
			cause = ctx.Err()
		} else if ctx.Err() == nil {
			cause = fmt.Errorf("%w: no download progress for %s", context.DeadlineExceeded, opts.inactivity())
		}
		_ = terminateProcessTree(cmd, opts.terminateWait())
		select {
		case <-done:
		case <-time.After(opts.terminateWait()):
			_ = killProcessTree(cmd)
			select {
			case <-done:
			case <-time.After(defaultWaitAfterKill):
				return out.Bytes(), fmt.Errorf("%w: process did not exit after kill", cause)
			}
		}
		return out.Bytes(), cause
	}
}

func watchDownloadProgress(ctx context.Context, cancel context.CancelFunc, path string, inactivity time.Duration) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	lastSize := int64(-1)
	lastGrowth := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fi, err := os.Stat(path)
			var size int64
			if err == nil {
				size = fi.Size()
			}
			if size > lastSize {
				lastSize = size
				lastGrowth = time.Now()
				continue
			}
			if time.Since(lastGrowth) >= inactivity {
				cancel()
				return
			}
		}
	}
}

type limitedBuffer struct {
	max int
	mu  sync.Mutex
	buf bytes.Buffer
}

func (l *limitedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	remain := l.max - l.buf.Len()
	if remain <= 0 {
		return len(p), nil
	}
	if len(p) > remain {
		_, _ = l.buf.Write(p[:remain])
		return len(p), nil
	}
	return l.buf.Write(p)
}

func (l *limitedBuffer) Bytes() []byte {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]byte(nil), l.buf.Bytes()...)
}
