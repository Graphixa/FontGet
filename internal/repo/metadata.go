package repo

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

// FontMetadata represents the parsed METADATA.pb file
type FontMetadata struct {
	Name        string
	Designer    string
	License     string
	Category    string
	DateAdded   string
	Version     string
	Description string
	Subsets     []string
	Variants    []string
}

// parseMetadataPB parses a Protocol Buffer encoded METADATA.pb file
func parseMetadataPB(data []byte) (*FontMetadata, error) {
	metadata := &FontMetadata{}
	reader := strings.NewReader(string(data))

	for {
		// Read field number and wire type
		header, err := binary.ReadUvarint(reader)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read field header: %w", err)
		}

		fieldNumber := header >> 3
		wireType := header & 0x7

		// Read field value based on wire type
		switch wireType {
		case 0: // Varint
			value, err := binary.ReadUvarint(reader)
			if err != nil {
				return nil, fmt.Errorf("failed to read varint: %w", err)
			}
			// Handle field based on field number
			switch fieldNumber {
			case 1: // name
				metadata.Name = string(value)
			case 2: // designer
				metadata.Designer = string(value)
			case 3: // license
				metadata.License = string(value)
			case 4: // category
				metadata.Category = string(value)
			case 5: // date_added
				metadata.DateAdded = string(value)
			case 6: // version
				metadata.Version = string(value)
			case 7: // description
				metadata.Description = string(value)
			}
		case 2: // Length-delimited
			length, err := binary.ReadUvarint(reader)
			if err != nil {
				return nil, fmt.Errorf("failed to read length: %w", err)
			}
			value := make([]byte, length)
			if _, err := io.ReadFull(reader, value); err != nil {
				return nil, fmt.Errorf("failed to read value: %w", err)
			}
			// Handle field based on field number
			switch fieldNumber {
			case 8: // subsets
				metadata.Subsets = append(metadata.Subsets, string(value))
			case 9: // variants
				metadata.Variants = append(metadata.Variants, string(value))
			}
		}
	}

	return metadata, nil
}
