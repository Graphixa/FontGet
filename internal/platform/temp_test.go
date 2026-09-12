package platform

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestOperationStagingCleanupLeavesSibling(t *testing.T) {
	a, err := NewOperationStaging()
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewOperationStaging()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = a.Cleanup()
		_ = b.Cleanup()
	})

	marker := filepath.Join(b.Root, "keep.txt")
	if err := os.WriteFile(marker, []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := a.Cleanup(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(a.Root); !os.IsNotExist(err) {
		t.Fatalf("cleaned staging still present: %v", err)
	}
	got, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("sibling staging lost: %v", err)
	}
	if string(got) != "keep" {
		t.Fatalf("sibling contents = %q", got)
	}

	root, err := GetTempDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := CleanupTempDir(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("shared temp root must survive occupied cleanup: %v", err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("occupied child must survive shared-root cleanup: %v", err)
	}
}

func TestOperationStagingConcurrentCleanup(t *testing.T) {
	const n = 8
	stags := make([]*OperationStaging, n)
	for i := 0; i < n; i++ {
		s, err := NewOperationStaging()
		if err != nil {
			t.Fatal(err)
		}
		stags[i] = s
		if err := os.WriteFile(filepath.Join(s.Root, "x"), []byte{byte(i)}, 0644); err != nil {
			t.Fatal(err)
		}
	}
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer wg.Done()
			if err := stags[i].Cleanup(); err != nil {
				t.Errorf("cleanup %d: %v", i, err)
			}
		}()
	}
	wg.Wait()
	for i, s := range stags {
		if _, err := os.Stat(s.Root); !os.IsNotExist(err) {
			t.Errorf("staging %d still present", i)
		}
	}
}
