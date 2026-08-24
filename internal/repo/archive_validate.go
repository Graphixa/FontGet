package repo

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// FontFormat is a supported on-disk font container detected by magic bytes.
type FontFormat int

const (
	FontFormatUnknown FontFormat = iota
	FontFormatTrueType           // sfnt 0x00010000
	FontFormatOpenTypeCFF        // OTTO
	FontFormatTrueTypeCollection // ttcf
	FontFormatWOFF
	FontFormatWOFF2
)

// DetectFontFormat identifies a supported font container from the start of r.
func DetectFontFormat(r io.Reader) (FontFormat, error) {
	var hdr [4]byte
	n, err := io.ReadFull(r, hdr[:])
	if err == io.ErrUnexpectedEOF || err == io.EOF {
		if n < 4 {
			return FontFormatUnknown, fmt.Errorf("%w: truncated header", ErrInvalidFontPayload)
		}
	} else if err != nil {
		return FontFormatUnknown, fmt.Errorf("%w: %v", ErrInvalidFontPayload, err)
	}

	switch {
	case binary.BigEndian.Uint32(hdr[:]) == 0x00010000:
		return FontFormatTrueType, nil
	case string(hdr[:]) == "OTTO":
		return FontFormatOpenTypeCFF, nil
	case string(hdr[:]) == "ttcf":
		return FontFormatTrueTypeCollection, nil
	case string(hdr[:]) == "wOFF":
		return FontFormatWOFF, nil
	case string(hdr[:]) == "wOF2":
		return FontFormatWOFF2, nil
	default:
		return FontFormatUnknown, fmt.Errorf("%w: unrecognized signature %q", ErrInvalidFontPayload, hdr[:])
	}
}

// ValidateFontFile checks that path begins with a supported font signature.
// WOFF/WOFF2 are recognized but rejected for installation (desktop TTF/OTF/TTC only).
func ValidateFontFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("%w: open %q: %v", ErrInvalidFontPayload, path, err)
	}
	defer f.Close()

	format, err := DetectFontFormat(f)
	if err != nil {
		return err
	}
	switch format {
	case FontFormatTrueType, FontFormatOpenTypeCFF, FontFormatTrueTypeCollection:
		return nil
	case FontFormatWOFF, FontFormatWOFF2:
		return fmt.Errorf("%w: web font container not supported for install", ErrInvalidFontPayload)
	default:
		return fmt.Errorf("%w: unsupported format", ErrInvalidFontPayload)
	}
}
