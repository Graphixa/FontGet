package platform

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestSfntBytesContainTable_singleFace(t *testing.T) {
	b := make([]byte, 12+16)
	copy(b[0:4], []byte("OTTO"))
	binary.BigEndian.PutUint16(b[4:6], 1) // numTables
	copy(b[12:16], []byte("fvar"))
	if !sfntBytesContainTable(b, "fvar") {
		t.Fatal("expected fvar")
	}
	if sfntBytesContainTable(b, "glyf") {
		t.Fatal("did not expect glyf")
	}
}

func TestFontFileContainsOpenTypeTable_invalidTag(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.otf")
	if err := os.WriteFile(p, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := FontFileContainsOpenTypeTable(p, "bad")
	if err == nil {
		t.Fatal("expected error for non-4-byte tag")
	}
}

func TestFontFileContainsOpenTypeTable_tempFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "probe.otf")
	b := make([]byte, 12+16)
	copy(b[0:4], []byte("OTTO"))
	binary.BigEndian.PutUint16(b[4:6], 1)
	copy(b[12:16], []byte("fvar"))
	if err := os.WriteFile(p, b, 0644); err != nil {
		t.Fatal(err)
	}
	ok, err := FontFileContainsOpenTypeTable(p, "fvar")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected true")
	}
}
