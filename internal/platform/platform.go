package platform

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"unicode/utf16"

	"golang.org/x/image/font/sfnt"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// InstallationScope defines where fonts should be installed
type InstallationScope string

const (
	// UserScope installs fonts for the current user only
	UserScope InstallationScope = "user"
	// MachineScope installs fonts system-wide
	MachineScope InstallationScope = "machine"
)

// FontManager defines the interface for platform-specific font operations
type FontManager interface {
	// InstallFont installs a font file to the system
	InstallFont(fontPath string, scope InstallationScope, force bool) error
	// RemoveFont removes a font from the system
	RemoveFont(fontName string, scope InstallationScope) error
	// GetFontDir returns the system's font directory for the given scope
	GetFontDir(scope InstallationScope) string
	// RequiresElevation returns whether the given scope requires elevation
	RequiresElevation(scope InstallationScope) bool
	// IsElevated checks if the current process is running with elevated privileges
	IsElevated() (bool, error)
	// GetElevationCommand returns the command to run the current process with elevation
	GetElevationCommand() (string, []string, error)
}

// Common helper functions
func ensureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

func copyFile(src, dst string) error {
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Get source file info for permissions
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get source file info: %w", err)
	}

	// Create destination file with same permissions
	dstFile, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	// Copy file contents in chunks
	buf := make([]byte, 32*1024) // 32KB buffer
	for {
		n, err := srcFile.Read(buf)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read source file: %w", err)
		}
		if n == 0 {
			break
		}
		if _, err := dstFile.Write(buf[:n]); err != nil {
			return fmt.Errorf("failed to write destination file: %w", err)
		}
	}

	// Ensure all data is written to disk
	if err := dstFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync destination file: %w", err)
	}

	return nil
}

// getFontName extracts the font filename from the path
func getFontName(fontPath string) string {
	return filepath.Base(fontPath)
}

// ListInstalledFonts returns a list of font files in the specified directory
func ListInstalledFonts(dir string) ([]string, error) {
	var fonts []string

	// Walk through the directory
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if the file is a font file
		ext := strings.ToLower(filepath.Ext(path))
		if isFontFile(ext) {
			fonts = append(fonts, filepath.Base(path))
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list fonts: %w", err)
	}

	return fonts, nil
}

// isFontFile checks if the file extension indicates a font file
func isFontFile(ext string) bool {
	switch ext {
	case ".ttf", ".otf", ".ttc", ".otc", ".pfb", ".pfm", ".pfa", ".bdf", ".pcf", ".psf", ".psfu":
		return true
	default:
		return false
	}
}

// OpenDirectory opens a directory using the platform's default file manager
func OpenDirectory(path string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default: // Linux and others
		cmd = exec.Command("xdg-open", path)
	}

	return cmd.Start()
}

// FontMetadata represents extracted font metadata
type FontMetadata struct {
	FamilyName        string
	StyleName         string
	FullName          string
	TypographicFamily string // raw NameID16 if present
	TypographicStyle  string // raw NameID17 if present
}

// ExtractFontMetadata extracts font family and style names from a font file
func ExtractFontMetadata(fontPath string) (*FontMetadata, error) {
	// Early font type detection - check file extension first
	ext := strings.ToLower(filepath.Ext(fontPath))
	if !isFontFile(ext) {
		// Not a font file, use filename parsing directly
		filename := filepath.Base(fontPath)
		family, style := parseFontNameImproved(filename)
		return &FontMetadata{
			FamilyName: family,
			StyleName:  style,
			FullName:   family + " " + style,
		}, nil
	}

	// Try header-only approach first for better performance
	metadata, err := extractFontMetadataHeaderOnly(fontPath)
	if err == nil {
		return metadata, nil
	}

	// Fall back to full file reading if header-only fails
	return extractFontMetadataFullFile(fontPath)
}

// extractFontMetadataHeaderOnly attempts to extract font metadata by reading only the header and name table
func extractFontMetadataHeaderOnly(fontPath string) (*FontMetadata, error) {
	file, err := os.Open(fontPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open font file: %w", err)
	}
	defer file.Close()

	// Read SFNT header (first 12 bytes)
	header := make([]byte, 12)
	if _, err := file.ReadAt(header, 0); err != nil {
		return nil, fmt.Errorf("failed to read SFNT header: %w", err)
	}

	// Check if it's a valid SFNT font
	if len(header) < 12 {
		return nil, fmt.Errorf("invalid SFNT header")
	}

	// Read number of tables (bytes 4-6)
	numTables := int(header[4])<<8 | int(header[5])
	if numTables < 1 || numTables > 100 { // Sanity check
		return nil, fmt.Errorf("invalid number of tables: %d", numTables)
	}

	// Read table directory (16 bytes per table)
	tableDirSize := numTables * 16
	tableDir := make([]byte, tableDirSize)
	if _, err := file.ReadAt(tableDir, 12); err != nil {
		return nil, fmt.Errorf("failed to read table directory: %w", err)
	}

	// Find the 'name' table
	var nameTableOffset, nameTableLength int
	for i := 0; i < numTables; i++ {
		offset := i * 16
		// Check if this is the 'name' table (tag is first 4 bytes)
		if string(tableDir[offset:offset+4]) == "name" {
			// Read offset and length (bytes 8-12 and 12-16)
			nameTableOffset = int(tableDir[offset+8])<<24 | int(tableDir[offset+9])<<16 |
				int(tableDir[offset+10])<<8 | int(tableDir[offset+11])
			nameTableLength = int(tableDir[offset+12])<<24 | int(tableDir[offset+13])<<16 |
				int(tableDir[offset+14])<<8 | int(tableDir[offset+15])
			break
		}
	}

	if nameTableOffset == 0 {
		return nil, fmt.Errorf("name table not found")
	}

	// Read the name table
	nameTableData := make([]byte, nameTableLength)
	if _, err := file.ReadAt(nameTableData, int64(nameTableOffset)); err != nil {
		return nil, fmt.Errorf("failed to read name table: %w", err)
	}

	// Parse the name table to extract font names
	return parseNameTable(nameTableData)
}

// extractFontMetadataFullFile falls back to reading the entire font file
func extractFontMetadataFullFile(fontPath string) (*FontMetadata, error) {
	// Read the font file
	fontData, err := os.ReadFile(fontPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read font file: %w", err)
	}

	// Parse the font using SFNT (supports both TTF and OTF)
	font, err := sfnt.Parse(fontData)
	if err != nil {
		// If font parsing fails, fall back to improved filename parsing
		filename := filepath.Base(fontPath)
		family, style := parseFontNameImproved(filename)
		return &FontMetadata{
			FamilyName: family,
			StyleName:  style,
			FullName:   family + " " + style,
		}, nil
	}

	var buf sfnt.Buffer
	metadata := &FontMetadata{}

	// Extract family name
	familyName, err := font.Name(&buf, sfnt.NameIDFamily)
	if err != nil || familyName == "" {
		// If we can't get the family name, fall back to filename parsing
		filename := filepath.Base(fontPath)
		family, _ := parseFontNameImproved(filename)
		metadata.FamilyName = family
	} else {
		metadata.FamilyName = familyName
	}

	// Extract style name
	styleName, err := font.Name(&buf, sfnt.NameIDSubfamily)
	if err != nil || styleName == "" {
		// If we can't get the style name, fall back to filename parsing
		filename := filepath.Base(fontPath)
		_, style := parseFontNameImproved(filename)
		metadata.StyleName = style
	} else {
		metadata.StyleName = styleName
	}

	// Extract full name
	fullName, err := font.Name(&buf, sfnt.NameIDFull)
	if err != nil || fullName == "" {
		// If we can't get the full name, combine family and style
		metadata.FullName = metadata.FamilyName + " " + metadata.StyleName
	} else {
		metadata.FullName = fullName
	}

	return metadata, nil
}

// parseNameTable parses the name table data to extract font names
func parseNameTable(nameTableData []byte) (*FontMetadata, error) {
	if len(nameTableData) < 6 {
		return nil, fmt.Errorf("name table too short")
	}

	// Read name table header
	count := int(nameTableData[2])<<8 | int(nameTableData[3])
	stringOffset := int(nameTableData[4])<<8 | int(nameTableData[5])

	if count == 0 || stringOffset >= len(nameTableData) {
		return nil, fmt.Errorf("invalid name table format")
	}

	metadata := &FontMetadata{}

	// Parse name records
	var typographicFamily string
	var legacyFamily string
	var typographicStyle string
	var legacyStyle string
	// Preferred (Microsoft/English) variants
	var prefTypoFamily string
	var prefTypoStyle string
	var prefLegacyFamily string
	var prefLegacyStyle string
	for i := 0; i < count; i++ {
		recordOffset := 6 + (i * 12)
		if recordOffset+12 > len(nameTableData) {
			break
		}

		// Read name record
		platformID := int(nameTableData[recordOffset])<<8 | int(nameTableData[recordOffset+1])
		_ = int(nameTableData[recordOffset+2])<<8 | int(nameTableData[recordOffset+3]) // encodingID (unused)
		languageID := int(nameTableData[recordOffset+4])<<8 | int(nameTableData[recordOffset+5])
		nameID := int(nameTableData[recordOffset+6])<<8 | int(nameTableData[recordOffset+7])
		length := int(nameTableData[recordOffset+8])<<8 | int(nameTableData[recordOffset+9])
		offset := int(nameTableData[recordOffset+10])<<8 | int(nameTableData[recordOffset+11])

		// Process common platforms: Microsoft(3), Unicode(0), Macintosh(1)
		if platformID != 3 && platformID != 0 && platformID != 1 {
			continue
		}

		// Calculate actual string offset
		stringStart := stringOffset + offset
		stringEnd := stringStart + length

		if stringEnd > len(nameTableData) {
			continue
		}

		// Extract the string
		stringData := nameTableData[stringStart:stringEnd]
		var name string

		switch platformID {
		case 3, 0:
			// Unicode (UTF-16BE)
			if len(stringData) >= 2 {
				// Convert UTF-16BE to UTF-8
				utf16Data := make([]uint16, len(stringData)/2)
				for j := 0; j < len(stringData); j += 2 {
					utf16Data[j/2] = uint16(stringData[j])<<8 | uint16(stringData[j+1])
				}
				name = string(utf16.Decode(utf16Data))
			}
		case 1:
			// Macintosh Roman (single-byte); best-effort cast
			name = string(stringData)
		}

		// Accumulate raw candidates, decide after loop
		isPreferred := (platformID == 3 && languageID == 1033)
		switch nameID {
		case 16:
			if name != "" {
				typographicFamily = name
				if isPreferred {
					prefTypoFamily = name
				}
			}
		case 1:
			if name != "" {
				if legacyFamily == "" {
					legacyFamily = name
				}
				if isPreferred && prefLegacyFamily == "" {
					prefLegacyFamily = name
				}
			}
		case 17:
			if name != "" {
				typographicStyle = name
				if isPreferred {
					prefTypoStyle = name
				}
			}
		case 2:
			if name != "" {
				if legacyStyle == "" {
					legacyStyle = name
				}
				if isPreferred && prefLegacyStyle == "" {
					prefLegacyStyle = name
				}
			}
		case 4:
			if metadata.FullName == "" && name != "" {
				metadata.FullName = name
			}
		}
	}

	// Decide winners with absolute precedence for typographic names
	// Prefer preferred variants when available
	if prefTypoFamily != "" {
		metadata.TypographicFamily = prefTypoFamily
	} else {
		metadata.TypographicFamily = typographicFamily
	}
	if prefTypoStyle != "" {
		metadata.TypographicStyle = prefTypoStyle
	} else {
		metadata.TypographicStyle = typographicStyle
	}

	if metadata.TypographicFamily != "" {
		metadata.FamilyName = typographicFamily
	} else {
		// If we captured a preferred legacy, use it; else general legacy
		if prefLegacyFamily != "" {
			metadata.FamilyName = prefLegacyFamily
		} else {
			metadata.FamilyName = legacyFamily
		}
	}
	if metadata.TypographicStyle != "" {
		metadata.StyleName = typographicStyle
	} else {
		if prefLegacyStyle != "" {
			metadata.StyleName = prefLegacyStyle
		} else {
			metadata.StyleName = legacyStyle
		}
	}

	// Fallback to filename parsing if we didn't get good names
	if metadata.FamilyName == "" || metadata.StyleName == "" {
		return nil, fmt.Errorf("insufficient name data in name table")
	}

	// Set full name if not found
	if metadata.FullName == "" {
		metadata.FullName = metadata.FamilyName + " " + metadata.StyleName
	}

	return metadata, nil
}

// parseFontNameImproved provides better font name parsing than the original
func parseFontNameImproved(filename string) (family, style string) {
	// Remove file extension
	name := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Remove variation parameters
	if idx := strings.Index(name, "["); idx != -1 {
		name = name[:idx]
	}

	// Remove "webfont" suffix if present
	name = strings.TrimSuffix(name, "-webfont")

	// Handle common compressed name patterns
	// Insert spaces before capital letters that follow lowercase letters
	// This handles cases like "SourceCodePro" -> "Source Code Pro"
	expanded := regexp.MustCompile(`([a-z])([A-Z])`).ReplaceAllString(name, `$1 $2`)

	// For fonts with spaces in their names
	if strings.Contains(expanded, " ") {
		// Extract the base family name
		parts := strings.Split(expanded, " ")
		if len(parts) > 0 {
			family = parts[0]
			// The rest is the style
			if len(parts) > 1 {
				style = strings.Join(parts[1:], " ")
			} else {
				style = "Regular"
			}
			return family, style
		}
	}

	// For other fonts, split by hyphens
	parts := strings.Split(expanded, "-")
	if len(parts) == 1 {
		return parts[0], "Regular"
	}

	// If we have multiple parts, assume the last part is the style
	// and everything else is the family name
	family = strings.Join(parts[:len(parts)-1], "-")
	style = cases.Title(language.English, cases.NoLower).String(parts[len(parts)-1])
	return family, style
}
