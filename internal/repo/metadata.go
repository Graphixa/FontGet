package repo

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
