package repo

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const sourceCacheFilePerm = 0600

// catalogFontCountHeader is a slim decode of a FontGet-Sources file: enough to
// validate JSON-as-object and read source_info.total_fonts without building a FontManifest.
type catalogFontCountHeader struct {
	SourceInfo struct {
		TotalFonts int `json:"total_fonts"`
	} `json:"source_info"`
}

// SourceCachePath returns the cache file for a source (~/.fontget/sources/{name}.json).
func SourceCachePath(sourceName string) (string, error) {
	dir, err := sourcesCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, sanitizeSourceNameForFilename(sourceName)+".json"), nil
}

func sourcesCacheDir() (string, error) {
	cacheDir, err := getFontGetDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "sources"), nil
}

// FontCountFromCatalogJSON returns source_info.total_fonts from a FontGet-Sources catalog.
// If total_fonts is missing or zero, it falls back to the number of keys in "fonts".
// A typed decode is used instead of json.Unmarshal into interface{} so invalid JSON fails
// and we never allocate a full FontManifest just to count.
func FontCountFromCatalogJSON(body []byte) (int, error) {
	var header catalogFontCountHeader
	if err := json.Unmarshal(body, &header); err != nil {
		return 0, fmt.Errorf("invalid JSON: %w", err)
	}
	if header.SourceInfo.TotalFonts > 0 {
		return header.SourceInfo.TotalFonts, nil
	}
	var fontsOnly struct {
		Fonts map[string]json.RawMessage `json:"fonts"`
	}
	if err := json.Unmarshal(body, &fontsOnly); err != nil {
		return 0, nil
	}
	return len(fontsOnly.Fonts), nil
}

// CachedSourceFontCount returns total_fonts from an existing cache file, or 0 if missing or invalid.
func CachedSourceFontCount(sourceName string) int {
	path, err := SourceCachePath(sourceName)
	if err != nil {
		return 0
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	n, err := FontCountFromCatalogJSON(data)
	if err != nil {
		return 0
	}
	return n
}

// PersistSourceCatalog validates catalog JSON and writes the raw downloaded body to cache
// via a temp file + rename. Pretty-print is skipped so the hot path does not re-encode 16MB catalogs.
func PersistSourceCatalog(sourceName string, body []byte) (fontCount int, cachePath string, err error) {
	n, err := FontCountFromCatalogJSON(body)
	if err != nil {
		return 0, "", err
	}
	path, err := WriteSourceCacheAtomic(sourceName, body)
	if err != nil {
		return 0, path, err
	}
	return n, path, nil
}

// WriteSourceCacheAtomic writes body to the source cache using a same-directory temp file
// and rename so a crash cannot leave a truncated JSON catalog.
func WriteSourceCacheAtomic(sourceName string, body []byte) (string, error) {
	path, err := SourceCachePath(sourceName)
	if err != nil {
		return "", err
	}
	if err := writeFileAtomic(path, body, sourceCacheFilePerm); err != nil {
		return path, err
	}
	return path, nil
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create cache directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp cache file: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp cache file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp cache file: %w", err)
	}
	_ = os.Chmod(tmpName, perm)
	if err := renameReplace(tmpName, path); err != nil {
		return fmt.Errorf("install cache file: %w", err)
	}
	cleanup = false
	return nil
}

func renameReplace(tmp, dest string) error {
	if err := os.Rename(tmp, dest); err == nil {
		return nil
	}
	// Windows cannot rename over an existing file; remove dest then retry.
	if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(tmp, dest)
}

// PruneStaleSourceCaches removes cache files that do not belong to keepSourceNames
// (currently enabled/present sources). Include failed sources in keepSourceNames so
// their previous good cache is retained. Leftover temp files are also removed.
func PruneStaleSourceCaches(keepSourceNames []string) error {
	dir, err := sourcesCacheDir()
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	keep := make(map[string]struct{}, len(keepSourceNames))
	for _, name := range keepSourceNames {
		keep[sanitizeSourceNameForFilename(name)+".json"] = struct{}{}
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if _, ok := keep[name]; ok {
			continue
		}
		_ = os.Remove(filepath.Join(dir, name))
	}
	return nil
}
