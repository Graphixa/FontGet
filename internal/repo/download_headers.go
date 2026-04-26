package repo

import (
	"mime"
	"net/textproto"
	"net/url"
	"path/filepath"
	"strings"
)

// HTTPResponseInfo captures best-effort HTTP response metadata for archive detection.
type HTTPResponseInfo struct {
	ContentType        string
	ContentDisposition string
	FinalURL           string
}

// InferArchiveTypeFromHeaders attempts to infer archive type from HTTP headers.
// This is best-effort only; callers must still confirm using magic bytes.
func InferArchiveTypeFromHeaders(contentType string, contentDisposition string) ArchiveType {
	ct := strings.TrimSpace(contentType)
	if ct != "" {
		if mediaType, _, err := mime.ParseMediaType(ct); err == nil {
			mediaType = strings.ToLower(strings.TrimSpace(mediaType))
			switch mediaType {
			case "application/zip", "application/x-zip-compressed":
				return ArchiveTypeZIP
			case "application/x-7z-compressed":
				return ArchiveType7Z
			case "application/gzip", "application/x-gzip":
				return ArchiveTypeTARGZ
			}
		} else {
			// Some servers return non-conforming content-type; fall back to prefix match.
			ctLower := strings.ToLower(ct)
			if strings.HasPrefix(ctLower, "application/zip") || strings.HasPrefix(ctLower, "application/x-zip-compressed") {
				return ArchiveTypeZIP
			}
			if strings.HasPrefix(ctLower, "application/x-7z-compressed") {
				return ArchiveType7Z
			}
			if strings.HasPrefix(ctLower, "application/gzip") || strings.HasPrefix(ctLower, "application/x-gzip") {
				return ArchiveTypeTARGZ
			}
		}
	}

	fn := filenameFromContentDisposition(contentDisposition)
	if fn == "" {
		return ArchiveTypeUnknown
	}
	lower := strings.ToLower(fn)
	if strings.HasSuffix(lower, ".zip") {
		return ArchiveTypeZIP
	}
	if strings.HasSuffix(lower, ".tar.xz") {
		return ArchiveTypeTARXZ
	}
	if strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz") || strings.HasSuffix(lower, ".gz") {
		return ArchiveTypeTARGZ
	}
	if strings.HasSuffix(lower, ".7z") {
		return ArchiveType7Z
	}

	// If it has some other extension, don't guess.
	_ = filepath.Ext(lower)
	return ArchiveTypeUnknown
}

func filenameFromContentDisposition(cd string) string {
	cd = strings.TrimSpace(cd)
	if cd == "" {
		return ""
	}
	// ParseMediaType handles filename= and filename*= when present.
	_, params, err := mime.ParseMediaType(cd)
	if err != nil {
		return ""
	}

	// RFC 5987 / 6266: filename* takes precedence when present.
	if v := strings.TrimSpace(params["filename*"]); v != "" {
		// Expected format: charset''percent-encoded
		// We only need the final filename for suffix checks, so best-effort decode is enough.
		if parts := strings.SplitN(v, "''", 2); len(parts) == 2 {
			if decoded, err := url.QueryUnescape(parts[1]); err == nil && decoded != "" {
				return decoded
			}
		}
	}
	if v := strings.TrimSpace(params["filename"]); v != "" {
		return textproto.TrimString(v)
	}
	return ""
}

