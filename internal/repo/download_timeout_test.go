package repo

import (
	"errors"
	"testing"
)

func TestIsHTTP2HeaderTimeout(t *testing.T) {
	if !isHTTP2HeaderTimeout(errors.New("http2: timeout awaiting response headers")) {
		t.Fatalf("expected true")
	}
	if isHTTP2HeaderTimeout(errors.New("some other error")) {
		t.Fatalf("expected false")
	}
}

