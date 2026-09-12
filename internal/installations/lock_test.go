package installations

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestLockDestinationSerializesOverlappingWork(t *testing.T) {
	dir := t.TempDir()
	var overlapping int
	var maxOverlapping int
	var mu sync.Mutex
	var wg sync.WaitGroup
	const n = 4
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			unlock, err := LockDestination(context.Background(), dir)
			if err != nil {
				t.Errorf("lock: %v", err)
				return
			}
			mu.Lock()
			overlapping++
			if overlapping > maxOverlapping {
				maxOverlapping = overlapping
			}
			mu.Unlock()
			time.Sleep(20 * time.Millisecond)
			mu.Lock()
			overlapping--
			mu.Unlock()
			unlock()
		}()
	}
	wg.Wait()
	if maxOverlapping != 1 {
		t.Fatalf("overlapping holders=%d want 1", maxOverlapping)
	}
}

func TestLockDestinationCancelWhileWaiting(t *testing.T) {
	dir := t.TempDir()
	unlock, err := LockDestination(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	_, err = LockDestination(ctx, dir)
	if err == nil {
		t.Fatal("expected cancelled wait while dest is held")
	}
}
