package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGetDirSize(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create some test files
	files := []struct {
		name string
		size int64
	}{
		{"file1.txt", 100},
		{"file2.txt", 200},
		{"subdir/file3.txt", 150},
	}

	for _, file := range files {
		filePath := filepath.Join(tempDir, file.name)
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		f, err := os.Create(filePath)
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		// Write some data to the file
		data := make([]byte, file.size)
		if _, err := f.Write(data); err != nil {
			t.Fatalf("Failed to write to file: %v", err)
		}
		f.Close()
	}

	// Test getDirSize
	size := getDirSize(tempDir)
	expectedSize := int64(450) // 100 + 200 + 150

	if size != expectedSize {
		t.Errorf("getDirSize() = %d, expected %d", size, expectedSize)
	}
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{
			name:     "bytes",
			bytes:    512,
			expected: "512 B",
		},
		{
			name:     "kilobytes",
			bytes:    1024,
			expected: "1.0 KB",
		},
		{
			name:     "megabytes",
			bytes:    1024 * 1024,
			expected: "1.0 MB",
		},
		{
			name:     "gigabytes",
			bytes:    1024 * 1024 * 1024,
			expected: "1.0 GB",
		},
		{
			name:     "zero bytes",
			bytes:    0,
			expected: "0 B",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatFileSize(tt.bytes)
			if result != tt.expected {
				t.Errorf("formatFileSize() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "seconds",
			duration: 30 * time.Second,
			expected: "30s",
		},
		{
			name:     "minutes",
			duration: 5 * time.Minute,
			expected: "5m",
		},
		{
			name:     "hours",
			duration: 2 * time.Hour,
			expected: "2h",
		},
		{
			name:     "days",
			duration: 3 * 24 * time.Hour,
			expected: "3d",
		},
		{
			name:     "zero duration",
			duration: 0,
			expected: "0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("formatDuration() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestIsValidCacheFile(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		filename string
		content  string
		expected bool
	}{
		{
			name:     "valid JSON file",
			filename: "valid.json",
			content:  `{"name": "test", "value": 123}`,
			expected: true,
		},
		{
			name:     "invalid JSON file",
			filename: "invalid.json",
			content:  `{"name": "test", "value": 123`, // Missing closing brace
			expected: false,
		},
		{
			name:     "empty file",
			filename: "empty.json",
			content:  "",
			expected: false,
		},
		{
			name:     "non-JSON file",
			filename: "text.txt",
			content:  "This is not JSON",
			expected: false,
		},
		{
			name:     "JSON with whitespace",
			filename: "whitespace.json",
			content:  `  {"name": "test"}  `,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, tt.filename)

			if tt.content != "" {
				if err := os.WriteFile(filePath, []byte(tt.content), 0644); err != nil {
					t.Fatalf("Failed to write test file: %v", err)
				}
			} else {
				// Create empty file
				if f, err := os.Create(filePath); err != nil {
					t.Fatalf("Failed to create empty file: %v", err)
				} else {
					f.Close()
				}
			}

			result := isValidCacheFile(filePath)
			if result != tt.expected {
				t.Errorf("isValidCacheFile() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
