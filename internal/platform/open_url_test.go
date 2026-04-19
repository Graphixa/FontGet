package platform

import "testing"

func TestOpenURL_Invalid(t *testing.T) {
	t.Parallel()
	if err := OpenURL(""); err == nil {
		t.Fatal("expected error for empty URL")
	}
	if err := OpenURL("not-a-url"); err == nil {
		t.Fatal("expected error for non-http URL")
	}
}
