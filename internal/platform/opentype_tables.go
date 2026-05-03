package platform

import (
	"encoding/binary"
	"fmt"
	"os"
)

// Defensive upper bounds when scanning SFNT table directories (malformed / hostile files).
const (
	maxTTCollectionFonts = 64
	maxOpenTypeTables    = 256
)

// FontFileContainsOpenTypeTable reports whether the font file contains an OpenType table
// with the given four-byte tag (e.g. "fvar" for variable fonts). Supports single TTF/OTF
// and TTC/OTC collections (checks every face in the collection).
//
// tag must be exactly four bytes (e.g. "fvar", "glyf"); otherwise returns an error.
func FontFileContainsOpenTypeTable(fontPath, tag string) (bool, error) {
	if len(tag) != 4 {
		return false, fmt.Errorf("OpenType table tag must be exactly 4 bytes, got %d", len(tag))
	}
	data, err := os.ReadFile(fontPath)
	if err != nil {
		return false, err
	}
	return sfntBytesContainTable(data, tag), nil
}

// FontFileIsVariableFont reports whether the file is an OpenType variable font (fvar table present).
func FontFileIsVariableFont(fontPath string) (bool, error) {
	return FontFileContainsOpenTypeTable(fontPath, "fvar")
}

func sfntBytesContainTable(data []byte, wantTag string) bool {
	if len(data) < 12 {
		return false
	}
	sig := binary.BigEndian.Uint32(data[0:4])
	if sig == 0x74746366 { // "ttcf" — font collection
		n := binary.BigEndian.Uint32(data[8:12])
		if n == 0 || n > maxTTCollectionFonts {
			return false
		}
		pos := 12
		for i := uint32(0); i < n; i++ {
			if pos+4 > len(data) {
				return false
			}
			off := binary.BigEndian.Uint32(data[pos : pos+4])
			if sfntFaceContainsTable(data, off, wantTag) {
				return true
			}
			pos += 4
		}
		return false
	}
	return sfntFaceContainsTable(data, 0, wantTag)
}

func sfntFaceContainsTable(data []byte, fontOffset uint32, wantTag string) bool {
	fo := int(fontOffset)
	if fo < 0 || fo+12 > len(data) {
		return false
	}
	numTables := int(binary.BigEndian.Uint16(data[fo+4 : fo+6]))
	if numTables < 1 || numTables > maxOpenTypeTables {
		return false
	}
	for i := 0; i < numTables; i++ {
		rec := fo + 12 + i*16
		if rec+4 > len(data) {
			return false
		}
		if string(data[rec:rec+4]) == wantTag {
			return true
		}
	}
	return false
}
