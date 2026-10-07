package network

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"
)

// StallDetectingReader wraps an io.Reader to detect when data transfer stalls.
// It monitors the time since the last successful read and cancels the operation
// if no data is transferred within the inactivity timeout.
type StallDetectingReader struct {
	reader            io.Reader
	ctx               context.Context
	cancel            context.CancelFunc
	inactivityTimeout time.Duration
	overallTimeout    time.Duration
	startTime         time.Time
	lastActivity      time.Time
	lastActivityMutex sync.RWMutex
	bytesRead         int64
	bytesReadMutex    sync.RWMutex
	err               error
	errMutex          sync.RWMutex
}

// NewStallDetectingReader creates a new StallDetectingReader that wraps the given reader.
//
// Parameters:
//   - reader: The underlying io.Reader to wrap
//   - inactivityTimeout: Maximum time without data transfer before canceling (e.g., 30s)
//   - overallTimeout: Maximum total time for the entire operation (e.g., 30m as safety net, or 0 to disable)
//
// The reader will be canceled if:
//   - No data is transferred for longer than inactivityTimeout (stall detection), OR
//   - The total elapsed time exceeds overallTimeout (if > 0, safety net only)
//
// Note: For downloads, overallTimeout should be very long (30m+) or 0 (disabled) to allow
// slow but steady transfers to complete. The primary timeout mechanism is inactivityTimeout.
func NewStallDetectingReader(reader io.Reader, inactivityTimeout, overallTimeout time.Duration) *StallDetectingReader {
	ctx, cancel := context.WithCancel(context.Background())

	r := &StallDetectingReader{
		reader:            reader,
		ctx:               ctx,
		cancel:            cancel,
		inactivityTimeout: inactivityTimeout,
		overallTimeout:    overallTimeout,
		startTime:         time.Now(),
		lastActivity:      time.Now(),
	}

	// Start monitoring goroutine
	go r.monitor()

	return r
}

// Read implements io.Reader interface.
// It reads from the underlying reader and updates the last activity time on each successful read.
func (r *StallDetectingReader) Read(p []byte) (int, error) {
	// Check if already canceled
	if r.ctx.Err() != nil {
		r.errMutex.RLock()
		err := r.err
		r.errMutex.RUnlock()
		if err != nil {
			return 0, err
		}
		return 0, r.ctx.Err()
	}

	// Read from underlying reader
	n, err := r.reader.Read(p)

	if n > 0 {
		// Update activity tracking
		r.lastActivityMutex.Lock()
		r.lastActivity = time.Now()
		r.lastActivityMutex.Unlock()

		// Update bytes read counter
		r.bytesReadMutex.Lock()
		r.bytesRead += int64(n)
		r.bytesReadMutex.Unlock()
	}

	// If we hit EOF or an error, store it
	if err != nil && err != io.EOF {
		r.errMutex.Lock()
		r.err = err
		r.errMutex.Unlock()
	}

	return n, err
}

// monitor runs in a goroutine and periodically checks for timeouts.
// It cancels the context if either the inactivity timeout or overall timeout is exceeded.
func (r *StallDetectingReader) monitor() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			// Already canceled, exit
			return
		case <-ticker.C:
			// Check overall timeout (safety net only - should be very long or disabled)
			if r.overallTimeout > 0 && time.Since(r.startTime) > r.overallTimeout {
				r.errMutex.Lock()
				r.err = context.DeadlineExceeded
				r.errMutex.Unlock()
				r.cancel()
				return
			}

			// Check inactivity timeout
			r.lastActivityMutex.RLock()
			elapsed := time.Since(r.lastActivity)
			r.lastActivityMutex.RUnlock()

			if elapsed > r.inactivityTimeout {
				r.errMutex.Lock()
				r.err = &StallError{
					InactivityDuration: elapsed,
					Timeout:            r.inactivityTimeout,
				}
				r.errMutex.Unlock()
				r.cancel()
				return
			}
		}
	}
}

// Close cancels the monitoring and closes the underlying reader if it implements io.Closer.
func (r *StallDetectingReader) Close() error {
	r.cancel()
	if closer, ok := r.reader.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// BytesRead returns the total number of bytes read so far.
func (r *StallDetectingReader) BytesRead() int64 {
	r.bytesReadMutex.RLock()
	defer r.bytesReadMutex.RUnlock()
	return r.bytesRead
}

// StallError represents an error when download activity stalls.
type StallError struct {
	InactivityDuration time.Duration
	Timeout            time.Duration
}

func (e *StallError) Error() string {
	return fmt.Sprintf("download stalled: no activity for %v (timeout: %v)",
		e.InactivityDuration.Round(time.Second),
		e.Timeout.Round(time.Second))
}

// WrapReaderWithStallDetection wraps an io.Reader with stall detection using config values.
// This is a convenience function that reads timeout values from config.
//
// Parameters:
//   - reader: The io.Reader to wrap
//   - inactivityTimeout: Maximum time without data transfer (from config, e.g., DownloadTimeout)
//   - overallTimeout: Maximum total time for the operation (0 to disable, or very long like 30m as safety net)
//
// Returns a StallDetectingReader that will cancel if inactivity timeout is exceeded.
// If overallTimeout > 0, it also cancels if total time exceeds it (safety net only).
func WrapReaderWithStallDetection(reader io.Reader, inactivityTimeout, overallTimeout time.Duration) *StallDetectingReader {
	return NewStallDetectingReader(reader, inactivityTimeout, overallTimeout)
}
