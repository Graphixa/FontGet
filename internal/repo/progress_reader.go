package repo

import (
	"io"
	"time"
)

// progressReader wraps an io.Reader and emits byte progress updates.
// It throttles updates to avoid overwhelming UIs.
type progressReader struct {
	r         io.Reader
	total     int64
	cb        func(done int64, total int64)
	done      int64
	lastEmit  time.Time
	lastDone  int64
	minEvery  time.Duration
	minBytes  int64
	emitFirst bool
}

func newProgressReader(r io.Reader, total int64, cb func(done int64, total int64)) *progressReader {
	return &progressReader{
		r:         r,
		total:     total,
		cb:        cb,
		minEvery:  120 * time.Millisecond,
		minBytes:  256 * 1024, // 256KB
		emitFirst: true,
	}
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	if n > 0 {
		p.done += int64(n)
	}

	if p.cb != nil {
		now := time.Now()
		shouldEmit := false
		if p.emitFirst {
			shouldEmit = true
			p.emitFirst = false
		} else if p.done-p.lastDone >= p.minBytes {
			shouldEmit = true
		} else if !p.lastEmit.IsZero() && now.Sub(p.lastEmit) >= p.minEvery {
			shouldEmit = true
		}

		// Always emit on EOF so UIs can reach 100%.
		if err == io.EOF {
			shouldEmit = true
		}

		if shouldEmit {
			p.lastEmit = now
			p.lastDone = p.done
			p.cb(p.done, p.total)
		}
	}

	return n, err
}

