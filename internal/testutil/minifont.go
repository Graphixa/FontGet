package testutil

import (
	"encoding/binary"
	"fmt"
	"math"
	"unicode/utf16"
)

// MinimalTTF returns a tiny SFNT with a name table so ExtractFontMetadata succeeds.
func MinimalTTF(family, style string) []byte {
	if family == "" {
		family = "TestFamily"
	}
	if style == "" {
		style = "Regular"
	}
	full := family + " " + style
	type rec struct {
		id   uint16
		text string
	}
	names := []rec{{1, family}, {2, style}, {4, full}}

	var store []byte
	type nr struct {
		id, length, offset uint16
	}
	var recs []nr
	for _, n := range names {
		u := utf16.Encode([]rune(n.text))
		off := len(store)
		for _, r := range u {
			store = binary.BigEndian.AppendUint16(store, r)
		}
		recs = append(recs, nr{
			id:     n.id,
			length: toUint16(len(u) * 2),
			offset: toUint16(off),
		})
	}

	count := toUint16(len(recs))
	stringOffset := toUint16(6 + 12*int(count))
	nameTable := make([]byte, 0, int(stringOffset)+len(store))
	nameTable = append(nameTable, 0, 0) // format
	nameTable = binary.BigEndian.AppendUint16(nameTable, count)
	nameTable = binary.BigEndian.AppendUint16(nameTable, stringOffset)
	for _, r := range recs {
		nameTable = binary.BigEndian.AppendUint16(nameTable, 3)      // platform
		nameTable = binary.BigEndian.AppendUint16(nameTable, 1)      // encoding
		nameTable = binary.BigEndian.AppendUint16(nameTable, 0x0409) // language
		nameTable = binary.BigEndian.AppendUint16(nameTable, r.id)
		nameTable = binary.BigEndian.AppendUint16(nameTable, r.length)
		nameTable = binary.BigEndian.AppendUint16(nameTable, r.offset)
	}
	nameTable = append(nameTable, store...)
	for len(nameTable)%4 != 0 {
		nameTable = append(nameTable, 0)
	}

	var checksum uint32
	for i := 0; i+3 < len(nameTable); i += 4 {
		checksum += binary.BigEndian.Uint32(nameTable[i : i+4])
	}

	header := make([]byte, 12)
	binary.BigEndian.PutUint32(header[0:4], 0x00010000)
	binary.BigEndian.PutUint16(header[4:6], 1)  // numTables
	binary.BigEndian.PutUint16(header[6:8], 16) // searchRange
	binary.BigEndian.PutUint16(header[8:10], 0) // entrySelector
	binary.BigEndian.PutUint16(header[10:12], 0)

	tableDir := make([]byte, 16)
	copy(tableDir[0:4], []byte("name"))
	binary.BigEndian.PutUint32(tableDir[4:8], checksum)
	binary.BigEndian.PutUint32(tableDir[8:12], 28) // offset
	binary.BigEndian.PutUint32(tableDir[12:16], toUint32(len(nameTable)))

	out := append([]byte{}, header...)
	out = append(out, tableDir...)
	out = append(out, nameTable...)
	if len(out) < 1024 {
		out = append(out, make([]byte, 1024-len(out))...)
	}
	return out
}

func toUint16(n int) uint16 {
	if n < 0 || n > math.MaxUint16 {
		panic(fmt.Sprintf("uint16 overflow: %d", n))
	}
	return uint16(n) // #nosec G115 -- range-checked above for SFNT name-table field widths
}

func toUint32(n int) uint32 {
	if n < 0 || n > math.MaxUint32 {
		panic(fmt.Sprintf("uint32 overflow: %d", n))
	}
	return uint32(n) // #nosec G115 -- range-checked above for SFNT table length
}
