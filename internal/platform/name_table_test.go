package platform

import (
	"encoding/binary"
	"testing"
)

func utf16be(s string) []byte {
	// Best-effort: all test strings are ASCII so this is fine.
	// Encode as UTF-16BE with no BOM.
	out := make([]byte, 0, len(s)*2)
	for _, r := range s {
		u := uint16(r)
		out = append(out, byte(u>>8), byte(u))
	}
	return out
}

type nameRecord struct {
	platformID uint16
	encodingID uint16
	languageID uint16
	nameID     uint16
	data       []byte
}

func buildNameTable(records []nameRecord) []byte {
	// name table header:
	// u16 format (0)
	// u16 count
	// u16 stringOffset (from start of table)
	count := uint16(len(records))
	stringOffset := uint16(6 + 12*len(records))

	// Build string storage and compute offsets
	strings := make([]byte, 0, 256)
	type recWithOff struct {
		nameRecord
		length uint16
		offset uint16
	}
	withOff := make([]recWithOff, 0, len(records))
	for _, r := range records {
		off := uint16(len(strings))
		strings = append(strings, r.data...)
		withOff = append(withOff, recWithOff{
			nameRecord: r,
			length:     uint16(len(r.data)),
			offset:     off,
		})
	}

	buf := make([]byte, 0, int(stringOffset)+len(strings))
	h := make([]byte, 6)
	binary.BigEndian.PutUint16(h[0:2], 0)
	binary.BigEndian.PutUint16(h[2:4], count)
	binary.BigEndian.PutUint16(h[4:6], stringOffset)
	buf = append(buf, h...)

	for _, r := range withOff {
		rec := make([]byte, 12)
		binary.BigEndian.PutUint16(rec[0:2], r.platformID)
		binary.BigEndian.PutUint16(rec[2:4], r.encodingID)
		binary.BigEndian.PutUint16(rec[4:6], r.languageID)
		binary.BigEndian.PutUint16(rec[6:8], r.nameID)
		binary.BigEndian.PutUint16(rec[8:10], r.length)
		binary.BigEndian.PutUint16(rec[10:12], r.offset)
		buf = append(buf, rec...)
	}

	buf = append(buf, strings...)
	return buf
}

func TestParseNameTable_PrefTypographicWinsAndIsUsed(t *testing.T) {
	// Preferred typographic family/style should win and should populate FamilyName/StyleName.
	// Platform 3 (Microsoft), language 1033 (en-US) is considered preferred by parseNameTable.
	nameTable := buildNameTable([]nameRecord{
		// Non-preferred typographic (different language)
		{platformID: 3, encodingID: 1, languageID: 1031, nameID: 16, data: utf16be("Typo Fam DE")},
		{platformID: 3, encodingID: 1, languageID: 1031, nameID: 17, data: utf16be("Typo Style DE")},

		// Preferred typographic
		{platformID: 3, encodingID: 1, languageID: 1033, nameID: 16, data: utf16be("Typo Family")},
		{platformID: 3, encodingID: 1, languageID: 1033, nameID: 17, data: utf16be("Typo Style")},

		// Irrelevant record (should be skipped without decoding)
		{platformID: 3, encodingID: 1, languageID: 1033, nameID: 6, data: utf16be("PostScriptName")},
	})

	md, err := parseNameTable(nameTable)
	if err != nil {
		t.Fatalf("parseNameTable returned error: %v", err)
	}

	if md.TypographicFamily != "Typo Family" {
		t.Fatalf("TypographicFamily = %q, want %q", md.TypographicFamily, "Typo Family")
	}
	if md.TypographicStyle != "Typo Style" {
		t.Fatalf("TypographicStyle = %q, want %q", md.TypographicStyle, "Typo Style")
	}
	if md.FamilyName != "Typo Family" {
		t.Fatalf("FamilyName = %q, want %q", md.FamilyName, "Typo Family")
	}
	if md.StyleName != "Typo Style" {
		t.Fatalf("StyleName = %q, want %q", md.StyleName, "Typo Style")
	}
	if md.FullName != "Typo Family Typo Style" {
		t.Fatalf("FullName = %q, want %q", md.FullName, "Typo Family Typo Style")
	}
}

func TestParseNameTable_LegacyPreferredWhenNoTypographic(t *testing.T) {
	nameTable := buildNameTable([]nameRecord{
		// Legacy family/style, preferred (en-US)
		{platformID: 3, encodingID: 1, languageID: 1033, nameID: 1, data: utf16be("Legacy Family")},
		{platformID: 3, encodingID: 1, languageID: 1033, nameID: 2, data: utf16be("Regular")},
	})

	md, err := parseNameTable(nameTable)
	if err != nil {
		t.Fatalf("parseNameTable returned error: %v", err)
	}

	if md.TypographicFamily != "" || md.TypographicStyle != "" {
		t.Fatalf("unexpected typographic fields: %+v", md)
	}
	if md.FamilyName != "Legacy Family" {
		t.Fatalf("FamilyName = %q, want %q", md.FamilyName, "Legacy Family")
	}
	if md.StyleName != "Regular" {
		t.Fatalf("StyleName = %q, want %q", md.StyleName, "Regular")
	}
	if md.FullName != "Legacy Family Regular" {
		t.Fatalf("FullName = %q, want %q", md.FullName, "Legacy Family Regular")
	}
}

func BenchmarkParseNameTable_Large(b *testing.B) {
	records := make([]nameRecord, 0, 256)

	// Lots of irrelevant records (should be skipped before decode)
	for i := 0; i < 200; i++ {
		records = append(records, nameRecord{
			platformID: 3, encodingID: 1, languageID: 1033, nameID: 6,
			data: utf16be("SomeIrrelevantNameThatShouldBeSkipped"),
		})
	}

	// Preferred typographic at the end (forces scan until found)
	records = append(records,
		nameRecord{platformID: 3, encodingID: 1, languageID: 1033, nameID: 16, data: utf16be("Bench Family")},
		nameRecord{platformID: 3, encodingID: 1, languageID: 1033, nameID: 17, data: utf16be("Bench Style")},
	)

	nameTable := buildNameTable(records)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		md, err := parseNameTable(nameTable)
		if err != nil || md.FamilyName == "" || md.StyleName == "" {
			b.Fatalf("unexpected failure: md=%+v err=%v", md, err)
		}
	}
}

