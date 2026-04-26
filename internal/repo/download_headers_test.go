package repo

import "testing"

func TestInferArchiveTypeFromHeaders_ContentTypeZip(t *testing.T) {
	if got := InferArchiveTypeFromHeaders("application/zip", ""); got != ArchiveTypeZIP {
		t.Fatalf("got %v, want zip", got)
	}
	if got := InferArchiveTypeFromHeaders("application/zip; charset=binary", ""); got != ArchiveTypeZIP {
		t.Fatalf("got %v, want zip (params)", got)
	}
}

func TestInferArchiveTypeFromHeaders_ContentDispositionFilenameZip(t *testing.T) {
	cd := `attachment; filename="fontpack.zip"`
	if got := InferArchiveTypeFromHeaders("", cd); got != ArchiveTypeZIP {
		t.Fatalf("got %v, want zip", got)
	}
}

func TestInferArchiveTypeFromHeaders_ContentDispositionFilenameStarZip(t *testing.T) {
	cd := `attachment; filename*=UTF-8''font%20pack.zip`
	if got := InferArchiveTypeFromHeaders("", cd); got != ArchiveTypeZIP {
		t.Fatalf("got %v, want zip", got)
	}
}

