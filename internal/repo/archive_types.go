package repo

import "errors"

const (
	// Default extraction budgets. Fonts are typically small; these are generous to avoid
	// false positives while preventing archive bombs and disk exhaustion.
	defaultMaxExtractFileBytes  int64 = 200 << 20 // 200 MiB per file
	defaultMaxExtractTotalBytes int64 = 1 << 30   // 1 GiB total
	defaultMaxArchiveEntries          = 10_000
	defaultMaxSelectedFiles           = 512
)

// Sentinel errors for archive safety invariants. Wrap with context via fmt.Errorf("%w: ...").
var (
	ErrArchiveEntryTooLarge     = errors.New("archive entry exceeds extraction limit")
	ErrArchiveTotalLimit        = errors.New("archive exceeds total extraction limit")
	ErrArchiveEntryCountLimit   = errors.New("archive contains too many entries")
	ErrArchiveSelectedFileLimit = errors.New("archive selects too many font files")
	ErrArchiveUnsafePath        = errors.New("archive contains unsafe path")
	ErrArchivePathCollision     = errors.New("archive contains colliding paths")
	ErrInvalidFontPayload       = errors.New("archive entry is not a valid supported font")
)

// ArchiveEntryType classifies an archive member for selection and telemetry.
type ArchiveEntryType int

const (
	ArchiveEntryUnknown ArchiveEntryType = iota
	ArchiveEntryFont
	ArchiveEntryDir
	ArchiveEntryOther
)

// ArchiveEntry is format-agnostic metadata for one archive member.
type ArchiveEntry struct {
	Name             string
	IsDir            bool
	UncompressedSize uint64
	Type             ArchiveEntryType
	NormalizedPath   string // cleaned relative path using forward slashes
	Extension        string
}

// ExtractionPolicy holds hard resource limits for archive extraction.
// Values are internal defaults only; do not expose persistent user overrides.
type ExtractionPolicy struct {
	MaxFileBytes      int64
	MaxTotalBytes     int64
	MaxArchiveEntries int
	MaxSelectedFiles  int
}

// DefaultExtractionPolicy returns FontGet's built-in hard limits.
func DefaultExtractionPolicy() ExtractionPolicy {
	return ExtractionPolicy{
		MaxFileBytes:      defaultMaxExtractFileBytes,
		MaxTotalBytes:     defaultMaxExtractTotalBytes,
		MaxArchiveEntries: defaultMaxArchiveEntries,
		MaxSelectedFiles:  defaultMaxSelectedFiles,
	}
}

// resolveExtractionPolicy returns opts.Policy when set, otherwise defaults.
func resolveExtractionPolicy(opts *ExtractOptions) ExtractionPolicy {
	if opts != nil && opts.Policy != nil {
		p := *opts.Policy
		if p.MaxFileBytes <= 0 {
			p.MaxFileBytes = defaultMaxExtractFileBytes
		}
		if p.MaxTotalBytes <= 0 {
			p.MaxTotalBytes = defaultMaxExtractTotalBytes
		}
		if p.MaxArchiveEntries <= 0 {
			p.MaxArchiveEntries = defaultMaxArchiveEntries
		}
		if p.MaxSelectedFiles <= 0 {
			p.MaxSelectedFiles = defaultMaxSelectedFiles
		}
		return p
	}
	return DefaultExtractionPolicy()
}

// ArchiveSelectionContext supplies source-aware selection hints without coupling
// extractors to the full FontGet manifest model.
type ArchiveSelectionContext struct {
	SourcePrefix string
	FontID       string
	FontName     string
}
